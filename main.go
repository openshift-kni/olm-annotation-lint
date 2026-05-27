package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/reporter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
)

var version = "dev"

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func main() {
	var (
		path        string
		exclude     string
		allow       string
		strict      bool
		format      string
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

	paths := splitAndTrim(path)
	var excludePaths []string
	if exclude != "" {
		excludePaths = splitAndTrim(exclude)
	}
	var allowedAnnotations []string
	if allow != "" {
		allowedAnnotations = splitAndTrim(allow)
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
