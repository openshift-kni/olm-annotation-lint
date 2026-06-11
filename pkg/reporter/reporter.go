package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
)

type Format int

const (
	FormatText Format = iota
	FormatJSON
	FormatGitHub
)

func Report(w io.Writer, violations []linter.Violation, format Format, version string) {
	switch format {
	case FormatJSON:
		reportJSON(w, violations, version)
	case FormatGitHub:
		reportGitHub(w, violations)
	default:
		reportText(w, violations)
	}
}

func reportText(w io.Writer, violations []linter.Violation) {
	for _, v := range violations {
		severity := strings.ToUpper(v.Severity.String())
		if v.Line > 0 {
			_, _ = fmt.Fprintf(w, "%s:%d: [%s] %s: %s (on %s)\n", v.File, v.Line, severity, v.Annotation, v.Message, v.Kind)
		} else {
			_, _ = fmt.Fprintf(w, "%s: [%s] %s: %s (on %s)\n", v.File, severity, v.Annotation, v.Message, v.Kind)
		}
	}
}

type jsonViolation struct {
	File       string `json:"file"`
	Line       int    `json:"line,omitempty"`
	Annotation string `json:"annotation"`
	Kind       string `json:"kind"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
}

type jsonSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Total    int `json:"total"`
}

type jsonReport struct {
	Version    string          `json:"version"`
	Summary    jsonSummary     `json:"summary"`
	Violations []jsonViolation `json:"violations"`
}

func reportJSON(w io.Writer, violations []linter.Violation, ver string) {
	jvs := make([]jsonViolation, 0, len(violations))
	var errors, warnings int
	for _, v := range violations {
		jvs = append(jvs, jsonViolation{
			File:       v.File,
			Line:       v.Line,
			Annotation: v.Annotation,
			Kind:       v.Kind,
			Severity:   v.Severity.String(),
			Message:    v.Message,
		})
		switch v.Severity {
		case rules.SeverityError:
			errors++
		case rules.SeverityWarning:
			warnings++
		}
	}
	report := jsonReport{
		Version: ver,
		Summary: jsonSummary{
			Errors:   errors,
			Warnings: warnings,
			Total:    len(violations),
		},
		Violations: jvs,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		_, _ = fmt.Fprintf(w, "error encoding JSON: %v\n", err)
	}
}

func reportGitHub(w io.Writer, violations []linter.Violation) {
	for _, v := range violations {
		msg := strings.ReplaceAll(v.Message, "\n", "%0A")
		if v.Line > 0 {
			_, _ = fmt.Fprintf(w, "::%s file=%s,line=%d::%s: %s\n", v.Severity, v.File, v.Line, v.Annotation, msg)
		} else {
			_, _ = fmt.Fprintf(w, "::%s file=%s::%s: %s\n", v.Severity, v.File, v.Annotation, msg)
		}
	}
}

func HasErrors(violations []linter.Violation, strict bool) bool {
	for _, v := range violations {
		if v.Severity == rules.SeverityError {
			return true
		}
		if strict && v.Severity == rules.SeverityWarning {
			return true
		}
	}
	return false
}
