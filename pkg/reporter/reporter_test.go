package reporter_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/reporter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
)

var testViolations = []linter.Violation{
	{
		File:       "test.yaml",
		Line:       5,
		Annotation: "olm.fake",
		Kind:       "OperatorGroup",
		Severity:   rules.SeverityError,
		Message:    "unknown OLM annotation",
	},
	{
		File:       "test2.yaml",
		Line:       10,
		Annotation: "olm.operatorGroup",
		Kind:       "ClusterServiceVersion",
		Severity:   rules.SeverityWarning,
		Message:    "controller-managed annotation",
	},
}

func TestReportText(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, testViolations, reporter.FormatText, "1.0.0")

	output := buf.String()
	if !strings.Contains(output, "test.yaml:5") {
		t.Error("expected file:line in text output")
	}
	if !strings.Contains(output, "[ERROR]") {
		t.Error("expected ERROR severity in text output")
	}
	if !strings.Contains(output, "[WARNING]") {
		t.Error("expected WARNING severity in text output")
	}
}

func TestReportJSON(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, testViolations, reporter.FormatJSON, "1.0.0")

	var result struct {
		Version string `json:"version"`
		Summary struct {
			Errors   int `json:"errors"`
			Warnings int `json:"warnings"`
			Total    int `json:"total"`
		} `json:"summary"`
		Violations []map[string]interface{} `json:"violations"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result.Version)
	}
	if result.Summary.Errors != 1 {
		t.Errorf("expected 1 error in summary, got %d", result.Summary.Errors)
	}
	if result.Summary.Warnings != 1 {
		t.Errorf("expected 1 warning in summary, got %d", result.Summary.Warnings)
	}
	if result.Summary.Total != 2 {
		t.Errorf("expected 2 total in summary, got %d", result.Summary.Total)
	}
	if len(result.Violations) != 2 {
		t.Errorf("expected 2 violations in JSON, got %d", len(result.Violations))
	}
}

func TestReportGitHub(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, testViolations, reporter.FormatGitHub, "1.0.0")

	output := buf.String()
	if !strings.Contains(output, "::error file=test.yaml,line=5::") {
		t.Error("expected GitHub annotation format for error")
	}
	if !strings.Contains(output, "::warning file=test2.yaml,line=10::") {
		t.Error("expected GitHub annotation format for warning")
	}
}

func TestReportGitHubNotice(t *testing.T) {
	infoViolations := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       5,
			Annotation: "olm.custom",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityInfo,
			Message:    `annotation "olm.custom" allowed via user override`,
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, infoViolations, reporter.FormatGitHub, "1.0.0")
	output := buf.String()
	if !strings.Contains(output, "::notice file=test.yaml,line=5::") {
		t.Errorf("expected GitHub notice annotation, got: %s", output)
	}
}

func TestHasErrorsIgnoresInfo(t *testing.T) {
	infoOnly := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       5,
			Annotation: "olm.custom",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityInfo,
			Message:    "allowed via user override",
		},
	}
	if reporter.HasErrors(infoOnly, false) {
		t.Error("expected HasErrors to return false with only info violations")
	}
	if reporter.HasErrors(infoOnly, true) {
		t.Error("expected HasErrors to return false with info violations even in strict mode")
	}
}

func TestHasErrors(t *testing.T) {
	if !reporter.HasErrors(testViolations, false) {
		t.Error("expected HasErrors to return true with error violations")
	}

	warningOnly := []linter.Violation{testViolations[1]}
	if reporter.HasErrors(warningOnly, false) {
		t.Error("expected HasErrors to return false with only warnings")
	}
	if !reporter.HasErrors(warningOnly, true) {
		t.Error("expected HasErrors to return true with warnings in strict mode")
	}
}
