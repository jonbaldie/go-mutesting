package models

// Options Main config structure
type Options struct {
	General struct {
		Debug                bool   `long:"debug" description:"Debug log output"`
		DoNotRemoveTmpFolder bool   `long:"do-not-remove-tmp-folder" description:"Do not remove the tmp folder where all mutations are saved to"`
		Help                 bool   `long:"help" description:"Show this help message"`
		Verbose              bool   `long:"verbose" description:"Verbose log output"`
		Config               string `long:"config" description:"Path to config file"`
		HTMLOutput           bool   `long:"html-output" description:"Generates a go-mutesting-report.html file after testing is complete"`
	} `group:"General options"`

	Files struct {
		Blacklist []string `long:"blacklist" description:"List of MD5 checksums of mutations which should be ignored. Each checksum must end with a new line character."`
		ListFiles bool     `long:"list-files" description:"List found files"`
		PrintAST  bool     `long:"print-ast" description:"Print the ASTs of all given files and exit"`
	} `group:"File options"`

	Mutator struct {
		DisableMutators []string `long:"disable" description:"Disable mutator by their name or using * as a suffix pattern (in order to check remaining enabled mutators use --verbose option)"`
		ListMutators    bool     `long:"list-mutators" description:"List all available mutators (including disabled)"`
	} `group:"Mutator options"`

	Filter struct {
		Match string `long:"match" description:"Only functions are mutated that confirm to the arguments regex"`
	} `group:"Filter options"`

	Exec struct {
		Exec     string `long:"exec" description:"Execute this command for every mutation (by default the built-in exec command is used)"`
		NoExec   bool   `long:"no-exec" description:"Skip the built-in exec command and just generate the mutations"`
		Timeout  uint   `long:"exec-timeout" description:"Sets a timeout for the command execution (in seconds)" default:"10"`
		Coverage bool   `long:"coverage" description:"Run go test -coverprofile before mutating to compute covered-code MSI and mark uncovered mutants"`
	} `group:"Exec options"`

	// GitDiff limits mutation to lines changed since a git base ref.
	// Pair with --ignore-msi-with-no-mutations for clean CI on unchanged packages.
	GitDiff struct {
		Lines bool   `long:"git-diff-lines" description:"Only mutate lines changed since the git diff base"`
		Base  string `long:"git-diff-base" description:"Git ref to diff against for --git-diff-lines" default:"master"`
	} `group:"Git diff options"`

	Logger struct {
		GitHub bool `long:"logger-github" description:"Emit escaped mutants as GitHub Actions ::warning annotations"`
	} `group:"Logger options"`

	Test struct {
		Recursive bool `long:"test-recursive" description:"Defines if the executer should test recursively"`
	} `group:"Test options"`

	// Quality gates: fail with exit code 4 when metrics fall below thresholds.
	// -1 is the "not set" sentinel so that --min-msi 0 is distinguishable from
	// "flag not provided", and CLI always takes precedence over config file.
	Score struct {
		MinMsi                  float64 `long:"min-msi" description:"Minimum required MSI (0-100). Exit code 4 when not met." default:"-1"`
		MinCoveredMsi           float64 `long:"min-covered-msi" description:"Minimum required covered-MSI (0-100). Exit code 4 when not met." default:"-1"`
		IgnoreMsiWithNoMutations bool   `long:"ignore-msi-with-no-mutations" description:"Exit 0 even when MSI thresholds are not met if no mutations were generated (useful with --git-diff-lines)"`
	} `group:"Score options"`

	Remaining struct {
		Targets []string `description:"Packages, directories and files even with patterns (by default the current directory)"`
	} `positional-args:"true" required:"true"`

	Config struct {
		SkipFileWithoutTest  bool     `yaml:"skip_without_test"`
		SkipFileWithBuildTag bool     `yaml:"skip_with_build_tags"`
		JSONOutput           bool     `yaml:"json_output"`
		HTMLOutput           bool     `yaml:"html_output"`
		SilentMode           bool     `yaml:"silent_mode"`
		ExcludeDirs          []string `yaml:"exclude_dirs"`
		MinMsi               float64  `yaml:"min_msi"`
		MinCoveredMsi        float64  `yaml:"min_covered_msi"`
	}
}
