package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
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
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/jonbaldie/go-mutesting/v2/internal/annotation"
	"github.com/jonbaldie/go-mutesting/v2/internal/baseline"
	"github.com/jonbaldie/go-mutesting/v2/internal/console"
	"github.com/jonbaldie/go-mutesting/v2/internal/coverage"
	"github.com/jonbaldie/go-mutesting/v2/internal/filter"
	"github.com/jonbaldie/go-mutesting/v2/internal/gitdiff"
	"github.com/jonbaldie/go-mutesting/v2/internal/importing"
	"github.com/jonbaldie/go-mutesting/v2/internal/models"
	"github.com/jonbaldie/go-mutesting/v2/internal/parser"
	"github.com/jonbaldie/go-mutesting/v2/internal/reportmaker"
	"github.com/jessevdk/go-flags"
	"github.com/zimmski/osutil"

	"github.com/jonbaldie/go-mutesting/v2"
	"github.com/jonbaldie/go-mutesting/v2/astutil"
	"github.com/jonbaldie/go-mutesting/v2/mutator"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/arithmetic"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/branch"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/concurrency"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/conditional"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/expression"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/loop"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/numbers"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/select"
	_ "github.com/jonbaldie/go-mutesting/v2/mutator/statement"
)

const (
	returnOk = iota
	returnHelp
	returnBashCompletion
	returnError
	returnMsiThresholdNotMet // exit 4: quality gate failed
)

// isTerminal reports whether stderr is an interactive terminal.
func isTerminal() bool {
	fi, err := os.Stderr.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

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

// execJob carries everything a parallel worker needs to test one mutation.
type execJob struct {
	opts            *models.Options
	pkg             *types.Package
	originalFile    string
	mutationFile    string
	mutant          models.Mutant
	absFile         string
	coverProfile    *coverage.Profile
	gitChangedLines gitdiff.ChangedLines
	execs           []string
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

	// Load baseline once; nil means no baseline file active (opt-in feature).
	bl, err := baseline.Load(opts.Baseline.File)
	if err != nil {
		return exitError("Cannot load baseline: %v", err)
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
		execs = strings.Fields(opts.Exec.Exec)
	}

	report := &models.Report{}
	var reportMu sync.Mutex

	// Detect module path for coverage profile matching, and module root for relative path output.
	modulePath := detectModulePath()
	moduleRoot := detectModuleRoot()

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

	// Noop check: run the test suite once without mutations to confirm it passes.
	// Only applies to the built-in exec; custom --exec scripts depend on mutation
	// environment variables and cannot be invoked safely here.
	if opts.General.Noop && !opts.Exec.NoExec {
		if len(execs) > 0 {
			fmt.Fprintln(os.Stderr, "Warning: --noop is not supported with --exec; skipping initial test run")
		} else {
			for _, importPkg := range pkgs {
				pkgPath := packageImportPath(importPkg.Files)
				if pkgPath == "" {
					continue
				}
				cmd := exec.Command("go", "test", "-timeout", fmt.Sprintf("%ds", opts.Exec.Timeout), pkgPath)
				cmd.Env = os.Environ()
				if out, err := cmd.CombinedOutput(); err != nil {
					fmt.Fprintf(os.Stderr, "Noop check failed for %q — fix your tests before running mutation testing:\n%s\n", pkgPath, out)
					return returnError
				}
			}
			console.Verbose(opts, "Noop check passed — all packages green before mutation")
		}
	}

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

	// Set up the parallel worker pool for mutation execution.
	// Custom --exec scripts may not be thread-safe, so force 1 worker when set.
	numWorkers := opts.General.Workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if len(execs) > 0 {
		numWorkers = 1
	}
	console.Verbose(opts, "Running with %d parallel worker(s)", numWorkers)

	var (
		jobs  chan execJob
		jobWg sync.WaitGroup
	)
	if !opts.Exec.NoExec {
		jobs = make(chan execJob, numWorkers*2)
		for i := 0; i < numWorkers; i++ {
			jobWg.Add(1)
			go func() {
				defer jobWg.Done()
				for job := range jobs {
					runExecJob(job, report, &reportMu)
				}
			}()
		}
	}

	// Live progress on stderr: show running kill/escape/skip counts.
	// Suppressed when verbose/debug (individual lines already appear) or silent.
	var stopProgress chan struct{}
	if isTerminal() && !opts.General.Verbose && !opts.General.Debug && !opts.Config.SilentMode && !opts.Exec.NoExec {
		stopProgress = make(chan struct{})
		go func() {
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					reportMu.Lock()
					k := report.Stats.KilledCount
					e := report.Stats.EscapedCount
					s := report.Stats.SkippedCount
					n := report.Stats.NotCoveredCount
					reportMu.Unlock()
					fmt.Fprintf(os.Stderr, "\rMutating: killed=%-4d escaped=%-4d skip=%-4d not-covered=%-4d",
						k, e, s, n)
				case <-stopProgress:
					fmt.Fprintf(os.Stderr, "\r\033[K")
					return
				}
			}
		}()
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
						mutationID = mutate(opts, mutators, mutationBlackList, mutationID, pkg, info, file, fset, src, f, tmpFile, execs, report, &reportMu, nodeFilters, absFile, coverProfile, gitChangedLines, jobs)
					}
				}
			} else {
				_ = mutate(opts, mutators, mutationBlackList, mutationID, pkg, info, file, fset, src, src, tmpFile, execs, report, &reportMu, nodeFilters, absFile, coverProfile, gitChangedLines, jobs)
			}
		}
	}

	// Wait for all parallel workers to finish before computing the report.
	if jobs != nil {
		close(jobs)
		jobWg.Wait()
	}

	// Stop live progress before printing the summary line.
	if stopProgress != nil {
		close(stopProgress)
	}

	if !opts.General.DoNotRemoveTmpFolder {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			panic(err)
		}
		console.Debug(opts, "Remove %q", tmpDir)
	}

	report.Calculate()

	// Write baseline and exit 0 — do not run quality gates.
	if opts.Baseline.Update {
		if err := baseline.Write(opts.Baseline.File, report.Escaped, moduleRoot); err != nil {
			return exitError("Cannot write baseline: %v", err)
		}
		fmt.Printf("Baseline written to %q (%d surviving mutant(s))\n", opts.Baseline.File, len(report.Escaped))
		return returnOk
	}

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

	if opts.General.Config == "" || opts.Config.JSONOutput {
		err = reportmaker.MakeJSONReport(*report)
		if err != nil {
			return exitError(err.Error())
		}
		console.Verbose(opts, "Save report into %q", models.ReportFileName)
	}

	if opts.Logger.SummaryJSON {
		if err = reportmaker.MakeSummaryJSONReport(report.Stats); err != nil {
			return exitError(err.Error())
		}
		console.Verbose(opts, "Save summary into %q", models.ReportSummaryJSONFileName)
	}

	if opts.Logger.AgenticJSON {
		if err = reportmaker.MakeAgenticJSONReport(*report, moduleRoot); err != nil {
			return exitError(err.Error())
		}
		console.Verbose(opts, "Save agentic report into %q", models.ReportAgenticJSONFileName)
	}

	if opts.Config.HTMLOutput || opts.General.HTMLOutput {
		err = reportmaker.MakeHTMLReport(*report)
		if err != nil {
			return exitError(err.Error())
		}

		console.Verbose(opts, "Save report into %q", models.ReportHTMLFileName)
	}

	return checkQualityGates(opts, report, bl, moduleRoot)
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
	if report.HasCoverage {
		fmt.Printf("The covered-code mutation score is %.2f%%\n", covMsiPct)
	}

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
func checkQualityGates(opts *models.Options, report *models.Report, bl *baseline.File, moduleRoot string) int {
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
	if opts.Score.FailOnEscaped {
		// With a baseline active, only new escapes (not in baseline) trigger failure.
		newEscapes := bl.NewEscapes(report.Escaped, moduleRoot)
		if len(newEscapes) > 0 {
			qualifier := ""
			if bl != nil {
				qualifier = "new "
			}
			fmt.Fprintf(os.Stderr, "%d %smutant(s) escaped — kill them or run --update-baseline to accept\n", len(newEscapes), qualifier)
			failed = true
		}
	}
	if minMsi >= 0 && msiPct < minMsi {
		fmt.Fprintf(os.Stderr, "MSI %.2f%% is below minimum required %.2f%%\n", msiPct, minMsi)
		failed = true
	}
	if minCoveredMsi > 0 {
		if !report.HasCoverage {
			fmt.Fprintf(os.Stderr, "Covered MSI cannot be checked: --coverage was not enabled (score is always 0 without a profile)\n")
			failed = true
		} else if covMsiPct < minCoveredMsi {
			fmt.Fprintf(os.Stderr, "Covered MSI %.2f%% is below minimum required %.2f%%\n", covMsiPct, minCoveredMsi)
			failed = true
		}
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
	jobs chan<- execJob,
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
				out := fmt.Sprintf("INTERNAL ERROR %s\n", err.Error())
				fmt.Printf("%s", out)
				mutant.ProcessOutput = out
				mu.Lock()
				stats.Errored = append(stats.Errored, mutant)
				stats.Stats.ErrorCount++
				mu.Unlock()
			} else if duplicate {
				console.Debug(opts, "%q is a duplicate, we ignore it", mutationFile)
				mu.Lock()
				stats.Stats.DuplicatedCount++
				mu.Unlock()
			} else {
				console.Debug(opts, "Save mutation into %q with checksum %s", mutationFile, checksum)
				if jobs != nil {
					jobs <- execJob{
						opts:            opts,
						pkg:             pkg,
						originalFile:    originalFile,
						mutationFile:    mutationFile,
						mutant:          mutant,
						absFile:         absFile,
						coverProfile:    coverProfile,
						gitChangedLines: gitChangedLines,
						execs:           execs,
					}
				}
			}

			// Release the MutateWalk goroutine to reset the AST and advance.
			changed <- true
			<-changed
			changed <- true

			mutationID++
		}
	}

	return mutationID
}

// runExecJob executes a single mutation job in a worker goroutine.
// It applies the git-diff filter, runs go test via overlay (or --exec),
// checks coverage, and records the result under mu.
func runExecJob(job execJob, stats *models.Report, mu *sync.Mutex) {
	opts := job.opts
	mutant := job.mutant

	// Git diff filter: skip mutations not on changed lines.
	if job.gitChangedLines != nil {
		diffOut, _ := exec.Command("diff", "--label=Original", "--label=New", "-u", job.originalFile, job.mutationFile).CombinedOutput()
		lineNum := int(parser.FindOriginalStartLine(diffOut))
		if !gitdiff.IsLineChanged(job.gitChangedLines, job.absFile, lineNum) {
			console.Debug(opts, "Skip %q at line %d (not in git diff)", job.mutationFile, lineNum)
			return
		}
	}

	execExitCode := mutateExec(opts, job.pkg, job.originalFile, job.mutationFile, job.execs, &mutant)

	console.Debug(opts, "Exited with %d", execExitCode)

	mutatedSourceCode, err := os.ReadFile(job.mutationFile)
	if err != nil {
		log.Fatal(err)
	}
	mutant.Mutator.MutatedSourceCode = string(mutatedSourceCode)

	startLine := mutant.Mutator.OriginalStartLine
	notCovered := job.coverProfile != nil && startLine > 0 && !job.coverProfile.IsCovered(job.absFile, int(startLine))

	loc := mutant.Mutator.OriginalFilePath
	if rel, err := filepath.Rel(".", loc); err == nil {
		loc = filepath.ToSlash(rel)
	}
	if mutant.Mutator.OriginalStartLine > 0 {
		loc = fmt.Sprintf("%s:%d", loc, mutant.Mutator.OriginalStartLine)
	}
	msg := fmt.Sprintf("%s (%s)", loc, mutant.Mutator.MutatorName)

	mu.Lock()
	defer mu.Unlock()

	if notCovered {
		out := fmt.Sprintf("NOT COVERED %s\n", msg)
		if !opts.Config.SilentMode && !opts.General.Quiet {
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

		// Build a per-mutation overlay JSON so go test sees the mutated file
		// without touching the real source. Each parallel worker creates its own
		// overlay file, so there are no conflicts.
		absOrig, _ := filepath.Abs(file)
		absMut, _ := filepath.Abs(mutationFile)
		overlayData, _ := json.Marshal(struct {
			Replace map[string]string `json:"Replace"`
		}{Replace: map[string]string{absOrig: absMut}})

		overlayFile, err := os.CreateTemp("", "go-mutesting-overlay-*.json")
		if err != nil {
			panic(err)
		}
		if _, err := overlayFile.Write(overlayData); err != nil {
			overlayFile.Close()
			os.Remove(overlayFile.Name())
			panic(err)
		}
		overlayFile.Close()
		defer os.Remove(overlayFile.Name())

		pkgName := pkg.Path()
		if opts.Test.Recursive {
			pkgName += "/..."
		}

		goTestCmd := exec.Command("go", "test",
			"-overlay="+overlayFile.Name(),
			"-timeout", fmt.Sprintf("%ds", opts.Exec.Timeout),
			pkgName)
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

// detectModuleRoot returns the directory containing the module's go.mod file.
// Used to compute relative file paths for baseline and agentic JSON output.
func detectModuleRoot() string {
	cmd := exec.Command("go", "env", "GOMOD")
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		return ""
	}
	return filepath.Dir(gomod)
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
