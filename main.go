package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/reporter"
)

func main() {
	var (
		path    string
		exclude string
		strict  bool
		format  string
	)

	flag.StringVar(&path, "path", ".", "Path or comma-separated paths to scan")
	flag.StringVar(&exclude, "exclude", "", "Comma-separated paths to exclude")
	flag.BoolVar(&strict, "strict", false, "Treat warnings as errors")
	flag.StringVar(&format, "format", "text", "Output format: text, json, github")
	flag.Parse()

	paths := strings.Split(path, ",")
	var excludePaths []string
	if exclude != "" {
		excludePaths = strings.Split(exclude, ",")
	}

	violations, err := linter.Run(linter.Options{
		Paths:   paths,
		Exclude: excludePaths,
		Strict:  strict,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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

	if reporter.HasErrors(violations, strict) {
		errorCount := 0
		warningCount := 0
		for _, v := range violations {
			if v.Severity == 0 {
				errorCount++
			} else {
				warningCount++
			}
		}
		fmt.Fprintf(os.Stderr, "\nFound %d error(s) and %d warning(s)\n", errorCount, warningCount)
		os.Exit(1)
	}

	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "\nFound %d warning(s)\n", len(violations))
	}
}
