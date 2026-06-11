package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/reporter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
	"gopkg.in/yaml.v3"
)

var version = "dev"

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

	flag.StringVar(&path, "path", ".", "Path or comma-separated paths to scan")
	flag.StringVar(&path, "p", ".", "Path or comma-separated paths to scan (shorthand)")
	flag.StringVar(&exclude, "exclude", "", "Comma-separated paths to exclude")
	flag.StringVar(&exclude, "e", "", "Comma-separated paths to exclude (shorthand)")
	flag.StringVar(&allow, "allow", "", "Comma-separated annotation keys to allow (bypass unknown annotation errors)")
	flag.StringVar(&allow, "a", "", "Comma-separated annotation keys to allow (shorthand)")
	flag.BoolVar(&strict, "strict", false, "Treat warnings as errors")
	flag.BoolVar(&strict, "s", false, "Treat warnings as errors (shorthand)")
	flag.StringVar(&format, "format", "text", "Output format: text, json, github")
	flag.StringVar(&format, "f", "text", "Output format: text, json, github (shorthand)")
	flag.StringVar(&configPath, "config", "", "Path to config file (default: .olm-lint.yaml in current directory)")
	flag.StringVar(&configPath, "c", "", "Path to config file (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&showVersion, "v", false, "Print version and exit (shorthand)")
	flag.BoolVar(&listRules, "list-rules", false, "List all known OLM annotations and exit")
	flag.BoolVar(&listRules, "l", false, "List all known OLM annotations and exit (shorthand)")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	if listRules {
		rules.PrintRules(os.Stdout)
		return
	}

	setFlags := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })

	var cfg *config
	var err error
	if configPath != "" {
		cfg, err = loadConfig(configPath)
	} else {
		cfg, err = discoverConfig()
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
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
	}

	violations, err := linter.Run(linter.Options{
		Paths:              paths,
		Exclude:            excludePaths,
		AllowedAnnotations: allowedAnnotations,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	var outputFormat reporter.Format
	switch format {
	case "json":
		outputFormat = reporter.FormatJSON
	case "github":
		outputFormat = reporter.FormatGitHub
	default:
		outputFormat = reporter.FormatText
	}

	if len(violations) > 0 {
		reporter.Report(os.Stdout, violations, outputFormat)
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

	if reporter.HasErrors(violations, strict) {
		_, _ = fmt.Fprintf(os.Stderr, "\nFound %d error(s) and %d warning(s)\n", errorCount, warningCount)
		os.Exit(1)
	}

	if warningCount > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "\nFound %d warning(s)\n", warningCount)
	}
}
