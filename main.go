package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/reporter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
	"gopkg.in/yaml.v3"
)

var version = "dev"

func displayVersion(ldflag string, info *debug.BuildInfo, ok bool) string {
	if ldflag != "" && ldflag != "dev" {
		return ldflag
	}
	if !ok || info == nil {
		if ldflag == "" {
			return "dev"
		}
		return ldflag
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		if ldflag == "" {
			return "dev"
		}
		return ldflag
	}
	return v
}

func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	return displayVersion(version, info, ok)
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

type stringOrList []string

func (s *stringOrList) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*s = splitAndTrim(value.Value)
		return nil
	}
	var list []string
	if err := value.Decode(&list); err != nil {
		return err
	}
	*s = list
	return nil
}

type config struct {
	Path    stringOrList `yaml:"path"`
	Exclude stringOrList `yaml:"exclude"`
	Allow   stringOrList `yaml:"allow"`
	Strict  *bool        `yaml:"strict"`
	Format  string       `yaml:"format"`
}

func loadConfig(path string) (*config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // config path is user-specified CLI input
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}

func discoverConfig() (*config, error) {
	cfg, err := loadConfig(".olm-lint.yaml")
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return nil, nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	var (
		path        string
		exclude     string
		allow       string
		strict      bool
		format      string
		configPath  string
		showVersion bool
		listRules   bool
	)

	fs := flag.NewFlagSet("olm-annotation-lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&path, "path", ".", "Path or comma-separated paths to scan")
	fs.StringVar(&path, "p", ".", "Path or comma-separated paths to scan (shorthand)")
	fs.StringVar(&exclude, "exclude", "", "Comma-separated paths to exclude")
	fs.StringVar(&exclude, "e", "", "Comma-separated paths to exclude (shorthand)")
	fs.StringVar(&allow, "allow", "", "Comma-separated annotation keys to allow (bypass unknown annotation errors)")
	fs.StringVar(&allow, "a", "", "Comma-separated annotation keys to allow (shorthand)")
	fs.BoolVar(&strict, "strict", false, "Treat warnings as errors")
	fs.BoolVar(&strict, "s", false, "Treat warnings as errors (shorthand)")
	fs.StringVar(&format, "format", "text", "Output format: text, json, github, junit, sarif")
	fs.StringVar(&format, "f", "text", "Output format: text, json, github, junit, sarif (shorthand)")
	fs.StringVar(&configPath, "config", "", "Path to config file (default: .olm-lint.yaml in current directory)")
	fs.StringVar(&configPath, "c", "", "Path to config file (shorthand)")
	fs.BoolVar(&showVersion, "version", false, "Print version and exit")
	fs.BoolVar(&showVersion, "v", false, "Print version and exit (shorthand)")
	fs.BoolVar(&listRules, "list-rules", false, "List all known OLM annotations and exit")
	fs.BoolVar(&listRules, "l", false, "List all known OLM annotations and exit (shorthand)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if showVersion {
		_, _ = fmt.Fprintln(stdout, currentVersion())
		return 0
	}

	if listRules {
		rules.PrintRules(stdout)
		return 0
	}

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })

	var cfg *config
	var err error
	if configPath != "" {
		cfg, err = loadConfig(configPath)
	} else {
		cfg, err = discoverConfig()
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	paths := splitAndTrim(path)
	var excludePaths []string
	if exclude != "" {
		excludePaths = splitAndTrim(exclude)
	}
	var allowedAnnotations []string
	if allow != "" {
		allowedAnnotations = splitAndTrim(allow)
	}

	if cfg != nil {
		if !setFlags["path"] && !setFlags["p"] && len(cfg.Path) > 0 {
			paths = cfg.Path
		}
		if !setFlags["exclude"] && !setFlags["e"] && len(cfg.Exclude) > 0 {
			excludePaths = cfg.Exclude
		}
		if !setFlags["allow"] && !setFlags["a"] && len(cfg.Allow) > 0 {
			allowedAnnotations = cfg.Allow
		}
		if !setFlags["strict"] && !setFlags["s"] && cfg.Strict != nil {
			strict = *cfg.Strict
		}
		if !setFlags["format"] && !setFlags["f"] && cfg.Format != "" {
			format = cfg.Format
		}
	}

	violations, err := linter.Run(linter.Options{
		Paths:              paths,
		Exclude:            excludePaths,
		AllowedAnnotations: allowedAnnotations,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	outputFormat, err := reporter.ParseFormat(format)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	if len(violations) > 0 {
		reporter.Report(stdout, violations, outputFormat, currentVersion())
	}

	var errorCount, warningCount int
	for _, v := range violations {
		switch v.Severity {
		case rules.SeverityError:
			errorCount++
		case rules.SeverityWarning:
			warningCount++
		}
	}

	hasErrors := reporter.HasErrors(violations, strict)

	if ghOutput := os.Getenv("GITHUB_OUTPUT"); ghOutput != "" {
		writeGitHubOutputs(ghOutput, errorCount, warningCount, len(violations), hasErrors)
	}

	if hasErrors {
		_, _ = fmt.Fprintf(stderr, "\nFound %d error(s) and %d warning(s)\n", errorCount, warningCount)
		return 1
	}

	if warningCount > 0 {
		_, _ = fmt.Fprintf(stderr, "\nFound %d warning(s)\n", warningCount)
	}

	return 0
}

func writeGitHubOutputs(path string, errors, warnings, total int, hasErrors bool) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // GITHUB_OUTPUT path is set by the Actions runner
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: cannot write GitHub outputs: %v\n", err)
		return
	}
	defer func() { _ = f.Close() }()

	_, _ = fmt.Fprintf(f, "error-count=%d\n", errors)
	_, _ = fmt.Fprintf(f, "warning-count=%d\n", warnings)
	_, _ = fmt.Fprintf(f, "total-count=%d\n", total)
	_, _ = fmt.Fprintf(f, "has-errors=%t\n", hasErrors)
}
