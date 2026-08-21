package reporter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
)

type sarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string          `json:"id"`
	ShortDescription     sarifMessage    `json:"shortDescription"`
	DefaultConfiguration sarifRuleConfig `json:"defaultConfiguration"`
}

type sarifRuleConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId,omitempty"`
	RuleIndex *int            `json:"ruleIndex,omitempty"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

var ruleDescriptions = map[string]string{
	rules.RuleUnknownAnnotation: "Unknown OLM annotation",
	rules.RuleCaseMismatch:      "Annotation case mismatch",
	rules.RuleAllowedOverride:   "Annotation allowed via user override",
	rules.RuleControllerManaged: "Controller-managed annotation",
	rules.RuleWrongResourceKind: "Annotation on wrong resource kind",
	rules.RuleInvalidValue:      "Invalid annotation value",
	rules.RuleMissingAnnotation: "Required bundle annotation missing",
	rules.RuleBundlePackage:     "Bundle package name does not match CSV",
	rules.RuleDuplicateKey:      "Duplicate annotation key",
}

func sarifLevel(s rules.Severity) string {
	switch s {
	case rules.SeverityWarning:
		return "warning"
	case rules.SeverityInfo:
		return "note"
	default:
		return "error"
	}
}

func reportSARIF(w io.Writer, violations []linter.Violation, ver string) {
	ruleIndex := map[string]int{}
	sarifRules := make([]sarifRule, 0)

	for _, v := range violations {
		if v.Rule == "" {
			continue
		}
		if _, exists := ruleIndex[v.Rule]; !exists {
			ruleIndex[v.Rule] = len(sarifRules)
			desc := ruleDescriptions[v.Rule]
			if desc == "" {
				desc = v.Rule
			}
			sarifRules = append(sarifRules, sarifRule{
				ID:                   v.Rule,
				ShortDescription:     sarifMessage{Text: desc},
				DefaultConfiguration: sarifRuleConfig{Level: sarifLevel(v.Severity)},
			})
		}
	}

	results := make([]sarifResult, 0, len(violations))
	for _, v := range violations {
		r := sarifResult{
			RuleID:  v.Rule,
			Level:   sarifLevel(v.Severity),
			Message: sarifMessage{Text: v.Message},
		}
		if idx, ok := ruleIndex[v.Rule]; ok {
			r.RuleIndex = &idx
		}

		loc := sarifLocation{
			PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: v.File},
			},
		}
		if v.Line > 0 {
			loc.PhysicalLocation.Region = &sarifRegion{StartLine: v.Line}
		}
		r.Locations = []sarifLocation{loc}

		results = append(results, r)
	}

	report := sarifReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "olm-annotation-lint",
						Version:        ver,
						InformationURI: "https://github.com/openshift-kni/olm-annotation-lint",
						Rules:          sarifRules,
					},
				},
				Results: results,
			},
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		_, _ = fmt.Fprintf(w, "error encoding SARIF: %v\n", err)
	}
}
