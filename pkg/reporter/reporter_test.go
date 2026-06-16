package reporter_test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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

func TestReportTextNoLineNumber(t *testing.T) {
	violations := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       0,
			Annotation: "olm.fake",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityError,
			Message:    "unknown OLM annotation",
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, violations, reporter.FormatText, "1.0.0")
	output := buf.String()
	if strings.Contains(output, "test.yaml:0") {
		t.Error("line 0 should not appear as :0 in text output")
	}
	if !strings.Contains(output, "test.yaml: [ERROR]") {
		t.Errorf("expected 'test.yaml: [ERROR]' format without line number, got: %s", output)
	}
}

func TestReportGitHubNoLineNumber(t *testing.T) {
	violations := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       0,
			Annotation: "olm.fake",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityError,
			Message:    "unknown OLM annotation",
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, violations, reporter.FormatGitHub, "1.0.0")
	output := buf.String()
	if strings.Contains(output, "line=0") {
		t.Error("line 0 should not appear in GitHub output")
	}
	if !strings.Contains(output, "::error file=test.yaml::") {
		t.Errorf("expected GitHub format without line number, got: %s", output)
	}
}

func TestReportJSONEmptyViolations(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, []linter.Violation{}, reporter.FormatJSON, "1.0.0")
	var result struct {
		Version    string              `json:"version"`
		Summary    struct{ Total int } `json:"summary"`
		Violations []interface{}       `json:"violations"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON for empty violations: %v", err)
	}
	if result.Summary.Total != 0 {
		t.Errorf("expected 0 total, got %d", result.Summary.Total)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected empty violations array, got %d", len(result.Violations))
	}
}

func TestReportGitHubNewlineEscaping(t *testing.T) {
	violations := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       5,
			Annotation: "olm.fake",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityError,
			Message:    "line one\nline two",
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, violations, reporter.FormatGitHub, "1.0.0")
	output := buf.String()
	if strings.Contains(output, "line one\nline two") {
		t.Error("newlines should be escaped in GitHub format")
	}
	if !strings.Contains(output, "line one%0Aline two") {
		t.Errorf("expected %%0A escaping, got: %s", output)
	}
}

func TestReportJUnit(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, testViolations, reporter.FormatJUnit, "1.0.0")

	output := buf.String()
	if !strings.HasPrefix(output, "<?xml") {
		t.Error("expected XML declaration")
	}

	var result struct {
		XMLName    xml.Name `xml:"testsuites"`
		TestSuites []struct {
			Name     string `xml:"name,attr"`
			Tests    int    `xml:"tests,attr"`
			Failures int    `xml:"failures,attr"`
			Cases    []struct {
				Name      string `xml:"name,attr"`
				Classname string `xml:"classname,attr"`
				Failure   *struct {
					Message string `xml:"message,attr"`
					Type    string `xml:"type,attr"`
				} `xml:"failure"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JUnit XML: %v", err)
	}
	if len(result.TestSuites) != 1 {
		t.Fatalf("expected 1 testsuite, got %d", len(result.TestSuites))
	}
	suite := result.TestSuites[0]
	if suite.Name != "olm-annotation-lint" {
		t.Errorf("expected suite name olm-annotation-lint, got %s", suite.Name)
	}
	if suite.Tests != 2 {
		t.Errorf("expected 2 tests, got %d", suite.Tests)
	}
	if suite.Failures != 2 {
		t.Errorf("expected 2 failures, got %d", suite.Failures)
	}
	if len(suite.Cases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(suite.Cases))
	}
	if suite.Cases[0].Classname != "test.yaml" {
		t.Errorf("expected classname test.yaml, got %s", suite.Cases[0].Classname)
	}
	if suite.Cases[0].Failure == nil {
		t.Error("expected failure element on error violation")
	}
	if suite.Cases[0].Failure.Type != "error" {
		t.Errorf("expected failure type 'error', got %s", suite.Cases[0].Failure.Type)
	}
}

func TestReportJUnitEmpty(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, []linter.Violation{}, reporter.FormatJUnit, "1.0.0")

	var result struct {
		XMLName    xml.Name `xml:"testsuites"`
		TestSuites []struct {
			Tests    int `xml:"tests,attr"`
			Failures int `xml:"failures,attr"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JUnit XML for empty violations: %v", err)
	}
	if result.TestSuites[0].Tests != 0 {
		t.Errorf("expected 0 tests, got %d", result.TestSuites[0].Tests)
	}
	if result.TestSuites[0].Failures != 0 {
		t.Errorf("expected 0 failures, got %d", result.TestSuites[0].Failures)
	}
}

func TestReportJUnitInfoNotFailure(t *testing.T) {
	infoViolations := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       5,
			Annotation: "olm.custom",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityInfo,
			Message:    "allowed via user override",
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, infoViolations, reporter.FormatJUnit, "1.0.0")

	var result struct {
		XMLName    xml.Name `xml:"testsuites"`
		TestSuites []struct {
			Tests    int `xml:"tests,attr"`
			Failures int `xml:"failures,attr"`
			Cases    []struct {
				Failure *struct{} `xml:"failure"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JUnit XML: %v", err)
	}
	suite := result.TestSuites[0]
	if suite.Failures != 0 {
		t.Errorf("expected 0 failures for info-only violations, got %d", suite.Failures)
	}
	if suite.Cases[0].Failure != nil {
		t.Error("expected no failure element for info-severity violation")
	}
}

func TestReportJUnitXMLEscaping(t *testing.T) {
	violations := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       5,
			Annotation: "olm.fake",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityError,
			Message:    `value "bad" contains <special> & chars`,
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, violations, reporter.FormatJUnit, "1.0.0")

	output := buf.String()
	if strings.Contains(output, `<special>`) {
		t.Error("expected XML escaping of angle brackets in message")
	}
	if !strings.Contains(output, "&lt;special&gt;") {
		t.Errorf("expected escaped angle brackets, got: %s", output)
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
