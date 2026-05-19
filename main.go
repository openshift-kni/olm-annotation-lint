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
	flag.StringVar(&exclude, "exclude", "", "Comma-separated paths to exclude")
	flag.StringVar(&allow, "allow", "", "Comma-separated annotation keys to allow (bypass unknown annotation errors)")
	flag.BoolVar(&strict, "strict", false, "Treat warnings as errors")
	flag.StringVar(&format, "format", "text", "Output format: text, json, github")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&listRules, "list-rules", false, "List all known OLM annotations and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	if listRules {
		rules.PrintRules(os.Stdout)
		return
	}

	paths := strings.Split(path, ",")
	var excludePaths []string
	if exclude != "" {
		excludePaths = strings.Split(exclude, ",")
	}
	var allowedAnnotations []string
	if allow != "" {
		allowedAnnotations = strings.Split(allow, ",")
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
