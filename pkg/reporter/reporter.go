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

func Report(w io.Writer, violations []linter.Violation, format Format) {
	switch format {
	case FormatJSON:
		reportJSON(w, violations)
	case FormatGitHub:
		reportGitHub(w, violations)
	default:
		reportText(w, violations)
	}
}

func reportText(w io.Writer, violations []linter.Violation) {
	for _, v := range violations {
		severity := "ERROR"
		if v.Severity == rules.SeverityWarning {
			severity = "WARNING"
		}
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

func reportJSON(w io.Writer, violations []linter.Violation) {
	jvs := make([]jsonViolation, 0, len(violations))
	for _, v := range violations {
		severity := "error"
		if v.Severity == rules.SeverityWarning {
			severity = "warning"
		}
		jvs = append(jvs, jsonViolation{
			File:       v.File,
			Line:       v.Line,
			Annotation: v.Annotation,
			Kind:       v.Kind,
			Severity:   severity,
			Message:    v.Message,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(jvs) //nolint:errcheck
}

func reportGitHub(w io.Writer, violations []linter.Violation) {
	for _, v := range violations {
		level := "error"
		if v.Severity == rules.SeverityWarning {
			level = "warning"
		}
		msg := strings.ReplaceAll(v.Message, "\n", "%0A")
		if v.Line > 0 {
			_, _ = fmt.Fprintf(w, "::%s file=%s,line=%d::%s: %s\n", level, v.File, v.Line, v.Annotation, msg)
		} else {
			_, _ = fmt.Fprintf(w, "::%s file=%s::%s: %s\n", level, v.File, v.Annotation, msg)
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
