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
		Rule:       rules.RuleUnknownAnnotation,
		Message:    "unknown OLM annotation",
	},
	{
		File:       "test2.yaml",
		Line:       10,
		Annotation: "olm.operatorGroup",
		Kind:       "ClusterServiceVersion",
		Severity:   rules.SeverityWarning,
		Rule:       rules.RuleControllerManaged,
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

func TestReportTextSuggestion(t *testing.T) {
	violations := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       5,
			Annotation: "OLM.providedAPIs",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityError,
			Rule:       rules.RuleCaseMismatch,
			Message:    `annotation case mismatch: use "olm.providedAPIs" instead of "OLM.providedAPIs"`,
			Suggestion: "olm.providedAPIs",
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, violations, reporter.FormatText, "1.0.0")
	output := buf.String()
	if !strings.Contains(output, "Suggestion: olm.providedAPIs") {
		t.Errorf("expected suggestion line in text output, got: %s", output)
	}

	buf.Reset()
	reporter.Report(&buf, violations, reporter.FormatJSON, "1.0.0")
	if !strings.Contains(buf.String(), `"suggestion": "olm.providedAPIs"`) {
		t.Errorf("expected suggestion field in JSON output, got: %s", buf.String())
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
	if rule, ok := result.Violations[0]["rule"].(string); !ok || rule != rules.RuleUnknownAnnotation {
		t.Errorf("expected rule 'unknown-annotation' in first violation, got %v", result.Violations[0]["rule"])
	}
	if rule, ok := result.Violations[1]["rule"].(string); !ok || rule != rules.RuleControllerManaged {
		t.Errorf("expected rule 'controller-managed' in second violation, got %v", result.Violations[1]["rule"])
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
	if suite.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", suite.Failures)
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
	if suite.Cases[0].Failure.Type != rules.RuleUnknownAnnotation {
		t.Errorf("expected failure type %q, got %s", rules.RuleUnknownAnnotation, suite.Cases[0].Failure.Type)
	}
	if suite.Cases[1].Failure != nil {
		t.Error("warning violation should not have a failure element")
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

func TestReportJUnitWarningNotFailure(t *testing.T) {
	warnings := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       10,
			Annotation: "olm.operatorGroup",
			Kind:       "ClusterServiceVersion",
			Severity:   rules.SeverityWarning,
			Rule:       rules.RuleControllerManaged,
			Message:    "controller-managed annotation",
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, warnings, reporter.FormatJUnit, "1.0.0")

	if !strings.Contains(buf.String(), "<system-err>") {
		t.Errorf("expected system-err for warning, got: %s", buf.String())
	}
	if strings.Contains(buf.String(), "<failure") {
		t.Errorf("warning should not include a failure element, got: %s", buf.String())
	}

	var result struct {
		TestSuites []struct {
			Tests    int `xml:"tests,attr"`
			Failures int `xml:"failures,attr"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JUnit XML: %v", err)
	}
	if result.TestSuites[0].Tests != 1 {
		t.Errorf("expected 1 test, got %d", result.TestSuites[0].Tests)
	}
	if result.TestSuites[0].Failures != 0 {
		t.Errorf("expected 0 failures for warnings, got %d", result.TestSuites[0].Failures)
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

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    reporter.Format
		wantErr bool
	}{
		{"text", reporter.FormatText, false},
		{"json", reporter.FormatJSON, false},
		{"github", reporter.FormatGitHub, false},
		{"junit", reporter.FormatJUnit, false},
		{"sarif", reporter.FormatSARIF, false},
		{"xml", 0, true},
		{"", 0, true},
		{"TEXT", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := reporter.ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if err != nil && !strings.Contains(err.Error(), "unknown format") {
				t.Errorf("expected 'unknown format' in error, got: %v", err)
			}
		})
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

func TestReportSARIF(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, testViolations, reporter.FormatSARIF, "1.0.0")

	var result struct {
		Schema  string `json:"$schema"`
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name    string `json:"name"`
					Version string `json:"version"`
					Rules   []struct {
						ID               string `json:"id"`
						ShortDescription struct {
							Text string `json:"text"`
						} `json:"shortDescription"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID  string `json:"ruleId"`
				Level   string `json:"level"`
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region *struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}
	if result.Version != "2.1.0" {
		t.Errorf("expected SARIF version 2.1.0, got %s", result.Version)
	}
	if !strings.Contains(result.Schema, "sarif-schema-2.1.0") {
		t.Errorf("expected SARIF schema URL, got %s", result.Schema)
	}
	if len(result.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(result.Runs))
	}
	run := result.Runs[0]
	if run.Tool.Driver.Name != "olm-annotation-lint" {
		t.Errorf("expected tool name olm-annotation-lint, got %s", run.Tool.Driver.Name)
	}
	if run.Tool.Driver.Version != "1.0.0" {
		t.Errorf("expected tool version 1.0.0, got %s", run.Tool.Driver.Version)
	}
	if len(run.Tool.Driver.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(run.Tool.Driver.Rules))
	}
	if len(run.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(run.Results))
	}
	if run.Results[0].RuleID != rules.RuleUnknownAnnotation {
		t.Errorf("expected ruleId %q, got %q", rules.RuleUnknownAnnotation, run.Results[0].RuleID)
	}
	if run.Results[0].Level != "error" {
		t.Errorf("expected level error, got %s", run.Results[0].Level)
	}
	if run.Results[1].Level != "warning" {
		t.Errorf("expected level warning, got %s", run.Results[1].Level)
	}
	if run.Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI != "test.yaml" {
		t.Errorf("expected URI test.yaml, got %s", run.Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI)
	}
	if run.Results[0].Locations[0].PhysicalLocation.Region == nil || run.Results[0].Locations[0].PhysicalLocation.Region.StartLine != 5 {
		t.Error("expected region with startLine 5")
	}
}

func TestReportSARIFEmpty(t *testing.T) {
	var buf bytes.Buffer
	reporter.Report(&buf, []linter.Violation{}, reporter.FormatSARIF, "1.0.0")

	var result struct {
		Runs []struct {
			Results []interface{} `json:"results"`
			Tool    struct {
				Driver struct {
					Rules []interface{} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid SARIF JSON for empty violations: %v", err)
	}
	if len(result.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Runs[0].Results))
	}
}

func TestReportSARIFNoLine(t *testing.T) {
	violations := []linter.Violation{
		{
			File:       "test.yaml",
			Line:       0,
			Annotation: "olm.fake",
			Kind:       "OperatorGroup",
			Severity:   rules.SeverityError,
			Rule:       rules.RuleUnknownAnnotation,
			Message:    "unknown OLM annotation",
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, violations, reporter.FormatSARIF, "1.0.0")

	output := buf.String()
	if strings.Contains(output, "startLine") {
		t.Error("expected no region/startLine for line 0 violations")
	}
}

func TestReportSARIFDedupAndInfo(t *testing.T) {
	violations := []linter.Violation{
		{
			File: "a.yaml", Line: 1, Annotation: "olm.fake1",
			Kind: "OperatorGroup", Severity: rules.SeverityError,
			Rule: rules.RuleUnknownAnnotation, Message: "unknown 1",
		},
		{
			File: "b.yaml", Line: 2, Annotation: "olm.fake2",
			Kind: "OperatorGroup", Severity: rules.SeverityError,
			Rule: rules.RuleUnknownAnnotation, Message: "unknown 2",
		},
		{
			File: "c.yaml", Line: 3, Annotation: "olm.custom",
			Kind: "OperatorGroup", Severity: rules.SeverityInfo,
			Rule: rules.RuleAllowedOverride, Message: "allowed",
		},
	}
	var buf bytes.Buffer
	reporter.Report(&buf, violations, reporter.FormatSARIF, "1.0.0")

	var result struct {
		Runs []struct {
			Tool struct {
				Driver struct {
					InformationURI string `json:"informationUri"`
					Rules          []struct {
						ID            string `json:"id"`
						DefaultConfig struct {
							Level string `json:"level"`
						} `json:"defaultConfiguration"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleIndex *int   `json:"ruleIndex"`
				Level     string `json:"level"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}
	run := result.Runs[0]
	if run.Tool.Driver.InformationURI != "https://github.com/openshift-kni/olm-annotation-lint" {
		t.Errorf("expected informationUri, got %q", run.Tool.Driver.InformationURI)
	}
	if len(run.Tool.Driver.Rules) != 2 {
		t.Fatalf("expected 2 deduplicated rules, got %d", len(run.Tool.Driver.Rules))
	}
	if run.Results[0].RuleIndex == nil || *run.Results[0].RuleIndex != 0 ||
		run.Results[1].RuleIndex == nil || *run.Results[1].RuleIndex != 0 {
		t.Error("expected both unknown-annotation results to have ruleIndex 0")
	}
	if run.Results[2].RuleIndex == nil || *run.Results[2].RuleIndex != 1 {
		t.Error("expected allowed-override result to have ruleIndex 1")
	}
	if run.Results[2].Level != "note" {
		t.Errorf("expected level 'note' for info severity, got %q", run.Results[2].Level)
	}
	if run.Tool.Driver.Rules[1].DefaultConfig.Level != "note" {
		t.Errorf("expected rule default level 'note' for info rule, got %q", run.Tool.Driver.Rules[1].DefaultConfig.Level)
	}
}
