package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"

	"gopkg.in/yaml.v3"

	"github.com/jonbaldie/go-mutesting/internal/annotation"
	"github.com/jonbaldie/go-mutesting/internal/console"
	"github.com/jonbaldie/go-mutesting/internal/coverage"
	"github.com/jonbaldie/go-mutesting/internal/filter"
	"github.com/jonbaldie/go-mutesting/internal/gitdiff"
	"github.com/jonbaldie/go-mutesting/internal/importing"
	"github.com/jonbaldie/go-mutesting/internal/models"
	"github.com/jonbaldie/go-mutesting/internal/parser"
	"github.com/jonbaldie/go-mutesting/internal/reportmaker"
	"github.com/jessevdk/go-flags"
	"github.com/zimmski/osutil"

	"github.com/jonbaldie/go-mutesting"
	"github.com/jonbaldie/go-mutesting/astutil"
	"github.com/jonbaldie/go-mutesting/mutator"
	_ "github.com/jonbaldie/go-mutesting/mutator/arithmetic"
	_ "github.com/jonbaldie/go-mutesting/mutator/branch"
	_ "github.com/jonbaldie/go-mutesting/mutator/conditional"
	_ "github.com/jonbaldie/go-mutesting/mutator/expression"
	_ "github.com/jonbaldie/go-mutesting/mutator/loop"
	_ "github.com/jonbaldie/go-mutesting/mutator/numbers"
	_ "github.com/jonbaldie/go-mutesting/mutator/statement"
)

const (
	returnOk = iota
	returnHelp
	returnBashCompletion
	returnError
	returnMsiThresholdNotMet // exit 4: quality gate failed
)

func checkArguments(args []string, opts *models.Options) (bool, int) {
	p := flags.NewNamedParser("go-mutesting", flags.None)

	p.ShortDescription = "Mutation testing for Go source code"

	if _, err := p.AddGroup("go-mutesting", "go-mutesting arguments", opts); err != nil {
		return true, exitError(err.Error())
	}

	completion := len(os.Getenv("GO_FLAGS_COMPLETION")) > 0

	_, err := p.ParseArgs(args)
	if (opts.General.Help || len(args) == 0) && !completion {
		p.WriteHelp(os.Stdout)

		return true, returnOk // exit 0 is conventional for --help
	} else if opts.Mutator.ListMutators {
		for _, name := range mutator.List() {
			fmt.Println(name)
		}

		return true, returnOk
	}

	if err != nil {
		return true, exitError(err.Error())
	}

	if completion {
		return true, returnBashCompletion
	}

	if opts.General.Debug {
		opts.General.Verbose = true
	}

	if opts.General.Config != "" {
		yamlFile, err := os.ReadFile(opts.General.Config)
		if err != nil {
			return true, exitError("Could not read config file: %q", opts.General.Config)
		}
		err = yaml.Unmarshal(yamlFile, &opts.Config)
		if err != nil {
			return true, exitError("Could not unmarshall config file: %q, %v", opts.General.Config, err)
		}
	}

	return false, 0
}

func exitError(format string, args ...interface{}) int {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)

	return returnError
}

type mutatorItem struct {
	Name    string
	Mutator mutator.Mutator
}

func mainCmd(args []string) int {
	var opts = &models.Options{}
	var mutationBlackList = map[string]struct{}{}

	if exit, exitCode := checkArguments(args, opts); exit {
		return exitCode
	}

	files := importing.FilesOfArgs(opts.Remaining.Targets, opts)
	if len(files) == 0 {
		return exitError("Could not find any suitable Go source files")
	}

	if opts.Files.ListFiles {
		for _, file := range files {
			fmt.Println(file)
		}

		return returnOk
	} else if opts.Files.PrintAST {
		for _, file := range files {
			fmt.Println(file)

			src, _, err := parser.ParseFile(file)
			if err != nil {
				return exitError("Could not open file %q: %v", file, err)
			}

			mutesting.PrintWalk(src)

			fmt.Println()
		}

		return returnOk
	}

	if len(opts.Files.Blacklist) > 0 {
		for _, f := range opts.Files.Blacklist {
			c, err := os.ReadFile(f)
			if err != nil {
				return exitError("Cannot read blacklist file %q: %v", f, err)
			}

			for _, line := range strings.Split(string(c), "\n") {
				if line == "" {
					continue
				}

				if len(line) != 32 {
					return exitError("%q is not a MD5 checksum", line)
				}

				mutationBlackList[line] = struct{}{}
			}
		}
	}

	var mutators []mutatorItem

MUTATOR:
	for _, name := range mutator.List() {
		if len(opts.Mutator.DisableMutators) > 0 {
			for _, d := range opts.Mutator.DisableMutators {
				pattern := strings.HasSuffix(d, "*")

				if (pattern && strings.HasPrefix(name, d[:len(d)-2])) || (!pattern && name == d) {
					continue MUTATOR
				}
			}
		}

		console.Verbose(opts, "Enable mutator %q", name)

		m, _ := mutator.New(name)
		mutators = append(mutators, mutatorItem{
			Name:    name,
			Mutator: m,
		})
	}

	tmpDir, err := os.MkdirTemp("", "go-mutesting-")
	if err != nil {
		panic(err)
	}
	console.Verbose(opts, "Save mutations into %q", tmpDir)

	var execs []string
	if opts.Exec.Exec != "" {
		execs = strings.Split(opts.Exec.Exec, " ")
	}

	report := &models.Report{}
	var reportMu sync.Mutex

	// Detect module path for coverage profile matching.
	modulePath := detectModulePath()

	// Load git diff changed lines when --git-diff-lines is set.
	var gitChangedLines gitdiff.ChangedLines
	if opts.GitDiff.Lines {
		var err error
		gitChangedLines, err = gitdiff.ParseChangedLines(opts.GitDiff.Base)
		if err != nil {
			return exitError("Cannot load git diff: %v", err)
		}
		console.Verbose(opts, "Git diff filter active against %q (%d changed files)", opts.GitDiff.Base, len(gitChangedLines))
	}

	// Group files by package to enable per-package coverage runs.
	pkgs := importing.PackagesWithFilesOfArgs(opts.Remaining.Targets, opts)

	// coverProfileForPkg runs go test -coverprofile for pkg and returns the profile.
	// Returns nil when coverage is disabled or unavailable (soft failure).
	coverProfileForPkg := func(pkgFiles []string) *coverage.Profile {
		if opts.Exec.NoExec || !opts.Exec.Coverage {
			return nil
		}
		pkgPath := packageImportPath(pkgFiles)
		if pkgPath == "" {
			return nil
		}
		profileDir := filepath.Join(tmpDir, "coverage", filepath.FromSlash(pkgPath))
		if err := os.MkdirAll(profileDir, 0755); err != nil {
			console.Verbose(opts, "Cannot create coverage dir for %q: %v", pkgPath, err)
			return nil
		}
		profilePath := filepath.Join(profileDir, "coverage.out")
		if err := runCoverageProfile(pkgPath, profilePath); err != nil {
			console.Verbose(opts, "Coverage unavailable for %q: %v", pkgPath, err)
			return nil
		}
		prof, err := coverage.ParseProfile(profilePath, modulePath)
		if err != nil {
			console.Verbose(opts, "Coverage parse failed for %q: %v", pkgPath, err)
			return nil
		}
		return prof
	}

	for _, importPkg := range pkgs {
		coverProfile := coverProfileForPkg(importPkg.Files)
		if coverProfile != nil {
			report.HasCoverage = true
		}

		for _, file := range importPkg.Files {
			console.Verbose(opts, "Mutate %q", file)

			annotationProcessor := annotation.NewProcessor()
			skipFilterProcessor := filter.NewSkipMakeArgsFilter()

			collectors := []filter.NodeCollector{
				annotationProcessor,
				skipFilterProcessor,
			}

			nodeFilters := []filter.NodeFilter{
				annotationProcessor,
				skipFilterProcessor,
			}

			src, fset, pkg, info, err := parser.ParseAndTypeCheckFile(file, collectors)
			if err != nil {
				return exitError(err.Error())
			}

			err = os.MkdirAll(tmpDir+"/"+filepath.Dir(file), 0755)
			if err != nil {
				panic(err)
			}

			tmpFile := tmpDir + "/" + file

			originalFile := fmt.Sprintf("%s.original", tmpFile)
			err = osutil.CopyFile(file, originalFile)
			if err != nil {
				panic(err)
			}
			console.Debug(opts, "Save original into %q", originalFile)

			absFile, _ := filepath.Abs(file)

			mutationID := 0

			if opts.Filter.Match != "" {
				m, err := regexp.Compile(opts.Filter.Match)
				if err != nil {
					return exitError("Match regex is not valid: %v", err)
				}

				for _, f := range astutil.Functions(src) {
					if m.MatchString(f.Name.Name) {
						mutationID = mutate(opts, mutators, mutationBlackList, mutationID, pkg, info, file, fset, src, f, tmpFile, execs, report, &reportMu, nodeFilters, absFile, coverProfile, gitChangedLines)
					}
				}
			} else {
				_ = mutate(opts, mutators, mutationBlackList, mutationID, pkg, info, file, fset, src, src, tmpFile, execs, report, &reportMu, nodeFilters, absFile, coverProfile, gitChangedLines)
			}
		}
	}

	if !opts.General.DoNotRemoveTmpFolder {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			panic(err)
		}
		console.Debug(opts, "Remove %q", tmpDir)
	}

	report.Calculate()

	if !opts.Exec.NoExec {
		if !opts.Config.SilentMode {
			printSummary(report)
		}
		if opts.Logger.GitHub {
			printGitHubAnnotations(report)
		}
	} else {
		fmt.Println("Cannot do a mutation testing summary since no exec command was executed.")
	}

	err = reportmaker.MakeJSONReport(*report)
	if err != nil {
		return exitError(err.Error())
	}

	console.Verbose(opts, "Save report into %q", models.ReportFileName)

	if opts.Config.HTMLOutput || opts.General.HTMLOutput {
		err = reportmaker.MakeHTMLReport(*report)
		if err != nil {
			return exitError(err.Error())
		}

		console.Verbose(opts, "Save report into %q", models.ReportHTMLFileName)
	}

	return checkQualityGates(opts, report)
}

// printSummary prints the final mutation testing summary including per-mutator breakdown.
func printSummary(report *models.Report) {
	msiPct := report.Stats.Msi * 100
	covMsiPct := report.Stats.CoveredCodeMsi * 100
	fmt.Printf(
		"The mutation score is %.2f%% (%d killed, %d escaped, %d errored, %d not covered, %d skipped, %d total)\n",
		msiPct,
		report.Stats.KilledCount,
		report.Stats.EscapedCount,
		report.Stats.ErrorCount,
		report.Stats.NotCoveredCount,
		report.Stats.SkippedCount,
		report.Stats.TotalMutantsCount,
	)
	fmt.Printf("The covered-code mutation score is %.2f%%\n", covMsiPct)

	if len(report.MutatorStats) > 0 {
		fmt.Println("\nPer-mutator breakdown:")
		// Sort by name for stable output.
		sorted := make([]models.MutatorStats, len(report.MutatorStats))
		copy(sorted, report.MutatorStats)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
		for _, ms := range sorted {
			killRate := 0.0
			if ms.Total > 0 {
				killRate = float64(ms.Killed) / float64(ms.Total) * 100
			}
			fmt.Printf("  %-35s  killed %3d / %-3d  (%.0f%%)\n", ms.Name, ms.Killed, ms.Total, killRate)
		}
	}
}

// printGitHubAnnotations writes escaped mutants as GitHub Actions ::warning
// annotations so they appear inline in PR diffs. File paths are made relative
// to the repo root so GitHub can match them against the diff.
func printGitHubAnnotations(report *models.Report) {
	repoRoot := ""
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		repoRoot = strings.TrimSpace(string(out))
	}

	for _, m := range report.Escaped {
		filePath := filepath.ToSlash(m.Mutator.OriginalFilePath)
		if repoRoot != "" {
			if rel, err := filepath.Rel(repoRoot, m.Mutator.OriginalFilePath); err == nil {
				filePath = filepath.ToSlash(rel)
			}
		}
		fmt.Printf("::warning file=%s,line=%d,title=Mutant escaped (%s)::Escaped mutation at %s:%d — add a test to kill it\n",
			filePath,
			m.Mutator.OriginalStartLine,
			m.Mutator.MutatorName,
			filePath,
			m.Mutator.OriginalStartLine,
		)
	}
}

// checkQualityGates returns returnMsiThresholdNotMet if configured thresholds
// are not met, otherwise returnOk.
func checkQualityGates(opts *models.Options, report *models.Report) int {
	// When no mutations were generated (e.g. --git-diff-lines on an unchanged
	// package), skip threshold checks rather than failing with 0% MSI.
	if opts.Score.IgnoreMsiWithNoMutations && report.Stats.TotalMutantsCount == 0 {
		return returnOk
	}

	msiPct := report.Stats.Msi * 100
	covMsiPct := report.Stats.CoveredCodeMsi * 100

	// CLI flag is -1 when not provided; config file defaults to 0 when not set.
	// CLI always wins when explicitly set (>= 0); fall back to config otherwise.
	minMsi := opts.Score.MinMsi
	if minMsi < 0 {
		minMsi = opts.Config.MinMsi
	}
	minCoveredMsi := opts.Score.MinCoveredMsi
	if minCoveredMsi < 0 {
		minCoveredMsi = opts.Config.MinCoveredMsi
	}

	failed := false
	if opts.Score.FailOnEscaped && report.Stats.EscapedCount > 0 {
		fmt.Fprintf(os.Stderr, "%d mutant(s) escaped — use --fail-on-escaped requires all mutants to be killed\n", report.Stats.EscapedCount)
		failed = true
	}
	if minMsi >= 0 && msiPct < minMsi {
		fmt.Fprintf(os.Stderr, "MSI %.2f%% is below minimum required %.2f%%\n", msiPct, minMsi)
		failed = true
	}
	if minCoveredMsi >= 0 && covMsiPct < minCoveredMsi {
		fmt.Fprintf(os.Stderr, "Covered MSI %.2f%% is below minimum required %.2f%%\n", covMsiPct, minCoveredMsi)
		failed = true
	}
	if failed {
		return returnMsiThresholdNotMet
	}
	return returnOk
}

func mutate(
	opts *models.Options,
	mutators []mutatorItem,
	mutationBlackList map[string]struct{},
	mutationID int,
	pkg *types.Package,
	info *types.Info,
	originalFile string,
	fset *token.FileSet,
	src ast.Node,
	node ast.Node,
	mutatedFile string,
	execs []string,
	stats *models.Report,
	mu *sync.Mutex,
	filters []filter.NodeFilter,
	absFile string,
	coverProfile *coverage.Profile,
	gitChangedLines gitdiff.ChangedLines,
) int {
	for _, m := range mutators {
		console.Debug(opts, "Mutator %s", m.Name)

		mutatorAnnotated := annotation.DecoratorFilter(m.Mutator, m.Name, filters...)

		changed := mutesting.MutateWalk(pkg, info, node, mutatorAnnotated)

		for {
			_, ok := <-changed

			if !ok {
				break
			}

			originalSourceCode, err := os.ReadFile(originalFile)
			if err != nil {
				log.Fatal(err)
			}

			mutant := models.Mutant{}
			mutant.Mutator.MutatorName = m.Name
			mutant.Mutator.OriginalFilePath = originalFile
			mutant.Mutator.OriginalSourceCode = string(originalSourceCode)

			mutationFile := fmt.Sprintf("%s.%d", mutatedFile, mutationID)
			checksum, duplicate, err := saveAST(mutationBlackList, mutationFile, fset, src)
			if err != nil {
				fmt.Printf("INTERNAL ERROR %s\n", err.Error())
			} else if duplicate {
				console.Debug(opts, "%q is a duplicate, we ignore it", mutationFile)

				mu.Lock()
				stats.Stats.DuplicatedCount++
				mu.Unlock()
			} else {
				console.Debug(opts, "Save mutation into %q with checksum %s", mutationFile, checksum)

				if !opts.Exec.NoExec {
					// Git diff filter: skip mutations not on changed lines.
					// Run a fast local diff to get the mutated line number before
					// invoking go test (which is expensive).
					if gitChangedLines != nil {
						diffOut, _ := exec.Command("diff", "--label=Original", "--label=New", "-u", originalFile, mutationFile).CombinedOutput()
						lineNum := int(parser.FindOriginalStartLine(diffOut))
						if !gitdiff.IsLineChanged(gitChangedLines, absFile, lineNum) {
							console.Debug(opts, "Skip %q at line %d (not in git diff)", mutationFile, lineNum)
							goto advanceMutation
						}
					}

					execExitCode := mutateExec(opts, pkg, originalFile, mutationFile, execs, &mutant)

					console.Debug(opts, "Exited with %d", execExitCode)

					mutatedSourceCode, err := os.ReadFile(mutationFile)
					if err != nil {
						log.Fatal(err)
					}
					mutant.Mutator.MutatedSourceCode = string(mutatedSourceCode)

					// Check coverage before recording result.
					startLine := mutant.Mutator.OriginalStartLine
					notCovered := coverProfile != nil && startLine > 0 && !coverProfile.IsCovered(absFile, int(startLine))

					// Build a human-readable location string: relative source path + line.
					loc := mutant.Mutator.OriginalFilePath
					if rel, err := filepath.Rel(".", loc); err == nil {
						loc = filepath.ToSlash(rel)
					}
					if mutant.Mutator.OriginalStartLine > 0 {
						loc = fmt.Sprintf("%s:%d", loc, mutant.Mutator.OriginalStartLine)
					}
					msg := fmt.Sprintf("%s (%s)", loc, mutant.Mutator.MutatorName)

					mu.Lock()
					if notCovered {
						out := fmt.Sprintf("NOT COVERED %s\n", msg)
						if !opts.Config.SilentMode {
							console.PrintSkip(out)
						}
						mutant.ProcessOutput = out
						stats.NotCovered = append(stats.NotCovered, mutant)
						stats.Stats.NotCoveredCount++
					} else {
						switch execExitCode {
						case 0: // Tests failed → mutation killed
							out := fmt.Sprintf("PASS %s\n", msg)
							if !opts.Config.SilentMode && !opts.General.Quiet {
								console.PrintPass(out)
							}
							mutant.ProcessOutput = out
							stats.Killed = append(stats.Killed, mutant)
							stats.Stats.KilledCount++
						case 1: // Tests passed → mutation escaped
							out := fmt.Sprintf("FAIL %s\n", msg)
							if !opts.Config.SilentMode {
								console.PrintFail(out)
							}
							mutant.ProcessOutput = out
							stats.Escaped = append(stats.Escaped, mutant)
							stats.Stats.EscapedCount++
						case 2: // Did not compile → skip
							out := fmt.Sprintf("SKIP %s\n", msg)
							if !opts.Config.SilentMode && !opts.General.Quiet {
								console.PrintSkip(out)
							}
							mutant.ProcessOutput = out
							stats.Stats.SkippedCount++
						default:
							out := fmt.Sprintf("UNKNOWN exit code for %s\n", msg)
							if !opts.Config.SilentMode {
								console.PrintUnknown(out)
							}
							mutant.ProcessOutput = out
							stats.Errored = append(stats.Errored, mutant)
							stats.Stats.ErrorCount++
						}
					}
					mu.Unlock()
				}
			}

		advanceMutation:
			changed <- true

			// Ignore original state
			<-changed
			changed <- true

			mutationID++
		}
	}

	return mutationID
}

func mutateExec(
	opts *models.Options,
	pkg *types.Package,
	file string,
	mutationFile string,
	execs []string,
	mutant *models.Mutant,
) (execExitCode int) {
	if len(execs) == 0 {
		console.Debug(opts, "Execute built-in exec command for mutation")

		diff, err := exec.Command("diff", "--label=Original", "--label=New", "-u", file, mutationFile).CombinedOutput()

		startLine := parser.FindOriginalStartLine(diff)
		mutant.Mutator.OriginalStartLine = startLine

		if err == nil {
			execExitCode = 0
		} else if e, ok := err.(*exec.ExitError); ok {
			execExitCode = e.Sys().(syscall.WaitStatus).ExitStatus()
		} else {
			panic(err)
		}
		if execExitCode != 0 && execExitCode != 1 {
			fmt.Printf("%s\n", diff)

			panic("Could not execute diff on mutation file")
		}

		defer func() {
			_ = os.Rename(file+".tmp", file)
		}()

		err = os.Rename(file, file+".tmp")
		if err != nil {
			panic(err)
		}
		err = osutil.CopyFile(mutationFile, file)
		if err != nil {
			panic(err)
		}

		pkgName := pkg.Path()
		if opts.Test.Recursive {
			pkgName += "/..."
		}

		goTestCmd := exec.Command("go", "test", "-timeout", fmt.Sprintf("%ds", opts.Exec.Timeout), pkgName)
		goTestCmd.Env = os.Environ()

		test, err := goTestCmd.CombinedOutput()
		if err == nil {
			execExitCode = 0
		} else if e, ok := err.(*exec.ExitError); ok {
			execExitCode = e.Sys().(syscall.WaitStatus).ExitStatus()
		} else {
			panic(err)
		}

		if opts.General.Debug {
			fmt.Printf("%s\n", test)
		}

		mutant.Diff = string(diff)

		switch execExitCode {
		case 0: // Tests passed → FAIL (mutation escaped)
			if !opts.Config.SilentMode {
				console.PrintDiff(diff)
			}

			execExitCode = 1
		case 1: // Tests failed → PASS (mutation killed)
			if opts.General.Debug {
				console.PrintDiff(diff)
			}

			execExitCode = 0
		case 2: // Did not compile → SKIP
			if opts.General.Verbose {
				fmt.Println("Mutation did not compile")
			}

			if opts.General.Debug {
				console.PrintDiff(diff)
			}
		default: // Unknown exit code
			if !opts.Config.SilentMode {
				fmt.Println("Unknown exit code")
				console.PrintDiff(diff)
			}
		}

		return execExitCode
	}

	console.Debug(opts, "Execute %q for mutation", opts.Exec.Exec)

	// Compute diff so OriginalStartLine is available for --logger-github and --coverage.
	// diff exits 1 when files differ (normal), so discard the error.
	extDiff, _ := exec.Command("diff", "--label=Original", "--label=New", "-u", file, mutationFile).CombinedOutput()
	mutant.Mutator.OriginalStartLine = parser.FindOriginalStartLine(extDiff)
	mutant.Diff = string(extDiff)

	execCommand := exec.Command(execs[0], execs[1:]...)

	execCommand.Stderr = os.Stderr
	execCommand.Stdout = os.Stdout

	execCommand.Env = append(os.Environ(), []string{
		"MUTATE_CHANGED=" + mutationFile,
		fmt.Sprintf("MUTATE_DEBUG=%t", opts.General.Debug),
		"MUTATE_ORIGINAL=" + file,
		"MUTATE_PACKAGE=" + pkg.Path(),
		fmt.Sprintf("MUTATE_TIMEOUT=%d", opts.Exec.Timeout),
		fmt.Sprintf("MUTATE_VERBOSE=%t", opts.General.Verbose),
	}...)
	if opts.Test.Recursive {
		execCommand.Env = append(execCommand.Env, "TEST_RECURSIVE=true")
	}

	err := execCommand.Start()
	if err != nil {
		panic(err)
	}

	err = execCommand.Wait()

	if err == nil {
		execExitCode = 0
	} else if e, ok := err.(*exec.ExitError); ok {
		execExitCode = e.Sys().(syscall.WaitStatus).ExitStatus()
	} else {
		panic(err)
	}

	return execExitCode
}

func main() {
	os.Exit(mainCmd(os.Args[1:]))
}

func saveAST(mutationBlackList map[string]struct{}, file string, fset *token.FileSet, node ast.Node) (string, bool, error) {
	var buf bytes.Buffer

	h := md5.New()

	err := printer.Fprint(io.MultiWriter(h, &buf), fset, node)
	if err != nil {
		return "", false, err
	}

	checksum := fmt.Sprintf("%x", h.Sum(nil))

	if _, ok := mutationBlackList[checksum]; ok {
		return checksum, true, nil
	}

	mutationBlackList[checksum] = struct{}{}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return "", false, err
	}

	err = os.WriteFile(file, src, 0666)
	if err != nil {
		return "", false, err
	}

	return checksum, false, nil
}

// detectModulePath returns the current module path via `go list -m`.
// This works regardless of where go.mod lives relative to the working directory.
func detectModulePath() string {
	cmd := exec.Command("go", "list", "-m")
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// runCoverageProfile runs go test -coverprofile for pkg and writes output to profilePath.
// Test failures are tolerated — the profile may still be written.
func runCoverageProfile(pkg, profilePath string) error {
	cmd := exec.Command("go", "test", "-coverprofile="+profilePath, pkg)
	cmd.Env = os.Environ()
	// We intentionally ignore test failures: a package with failing tests should
	// still produce a (partial) coverage profile so we can identify covered lines.
	_ = cmd.Run()
	if _, err := os.Stat(profilePath); err != nil {
		return fmt.Errorf("coverage profile not created for %q", pkg)
	}
	return nil
}

// packageImportPath returns the import path for the package containing files,
// by parsing the first file's package declaration.
func packageImportPath(files []string) string {
	if len(files) == 0 {
		return ""
	}
	f, err := filepath.Abs(files[0])
	if err != nil {
		return ""
	}
	dir := filepath.Dir(f)
	cmd := exec.Command("go", "list", dir)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
