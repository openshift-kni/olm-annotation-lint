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
	reporter.Report(&buf, testViolations, reporter.FormatText)

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
	reporter.Report(&buf, testViolations, reporter.FormatJSON)

	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 violations in JSON, got %d", len(result))
	}
}

func TestReportGitHub(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, testViolations, reporter.FormatGitHub)

	output := buf.String()
	if !strings.Contains(output, "::error file=test.yaml,line=5::") {
		t.Error("expected GitHub annotation format for error")
	}
	if !strings.Contains(output, "::warning file=test2.yaml,line=10::") {
		t.Error("expected GitHub annotation format for warning")
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
