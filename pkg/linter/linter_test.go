package linter_test

import (
	"strings"
	"testing"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
)

func TestValidFiles(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation in %s: %s: %s", v.File, v.Annotation, v.Message)
		}
	}
}

func TestUnknownAnnotation(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/unknown_olm_annotation.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "olm.operatorframework.io/bundle-install-timeout", "unknown OLM annotation")
	if !found {
		t.Error("expected violation for unknown annotation olm.operatorframework.io/bundle-install-timeout")
	}
}

func TestWrongResourceType(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/wrong_resource_type.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "operatorframework.io/bundle-unpack-timeout", "is not valid on Subscription")
	if !found {
		t.Error("expected violation for wrong resource type")
	}
}

func TestBadDurationValue(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/bad_duration_value.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "operatorframework.io/bundle-unpack-timeout", "invalid duration value")
	if !found {
		t.Error("expected violation for bad duration value")
	}
}

func TestBadJSONValue(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/bad_json_value.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "operatorframework.io/suggested-namespace-template", "invalid JSON value")
	if !found {
		t.Error("expected violation for bad JSON value")
	}
}

func TestBadTemplateValue(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/bad_template_value.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "olm.catalogImageTemplate", "invalid template value")
	if !found {
		t.Error("expected violation for bad template value")
	}
}

func TestBadSemverRange(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/bad_semver_range.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "olm.skipRange", "invalid semver range")
	if !found {
		t.Error("expected violation for bad semver range")
	}
}

func TestWrongPrefix(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/wrong_prefix.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "olm.operatorframework.io/bundle-unpack-timeout", "unknown OLM annotation")
	if !found {
		t.Error("expected violation for wrong prefix")
	}
}

func TestCaseMismatch(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/case_mismatch.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "OLM.providedAPIs", "wrong casing")
	if !found {
		t.Error("expected violation for case mismatch")
	}
}

func TestControllerManagedAnnotation(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/controller_managed_annotation.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "olm.operatorGroup", "controller-managed")
	if !found {
		t.Error("expected warning for controller-managed annotation")
	}
}

func TestExcludePaths(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths:   []string{"../../testdata"},
		Exclude: []string{"invalid"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations when excluding invalid dir, got %d", len(violations))
	}
}

func TestAllowedAnnotationOverride(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths:              []string{"../../testdata/invalid/unknown_olm_annotation.yaml"},
		AllowedAnnotations: []string{"olm.operatorframework.io/bundle-install-timeout"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, v := range violations {
		if v.Annotation == "olm.operatorframework.io/bundle-install-timeout" {
			if v.Severity == rules.SeverityError {
				t.Error("expected allowed annotation to not produce an error")
			}
			if v.Severity != rules.SeverityInfo {
				t.Errorf("expected info severity for allowed annotation, got %s", v.Severity)
			}
			if !strings.Contains(v.Message, "allowed via user override") {
				t.Errorf("expected override message, got %q", v.Message)
			}
			return
		}
	}
	t.Error("expected a violation (info notice) for the allowed annotation")
}

func TestAllowedAnnotationDoesNotAffectKnownRules(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths:              []string{"../../testdata/valid"},
		AllowedAnnotations: []string{"olm.operatorframework.io/bundle-install-timeout"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range violations {
		if v.Severity == rules.SeverityError {
			t.Errorf("unexpected error on valid file with allow list set: %s: %s", v.Annotation, v.Message)
		}
	}
}

func TestMultiDocumentValid(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/multi_document.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range violations {
		if v.Severity == rules.SeverityError {
			t.Errorf("unexpected error in multi-document file: %s: %s", v.Annotation, v.Message)
		}
	}
}

func TestMultiDocumentMixed(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/multi_document_mixed.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := findViolation(violations, "olm.operatorframework.io/bundle-install-timeout", "unknown OLM annotation")
	if !found {
		t.Error("expected violation for unknown annotation in second document")
	}

	for _, v := range violations {
		if v.Annotation == "operatorframework.io/bundle-unpack-timeout" && v.Severity == rules.SeverityError {
			t.Error("valid annotation in first document should not produce an error")
		}
	}
}

func TestEmptyFile(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/empty_file.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for empty file, got %d", len(violations))
	}
}

func TestCommentsOnlyFile(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/comments_only.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for comments-only file, got %d", len(violations))
	}
}

func TestNoMetadataResource(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/no_metadata.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for resource without annotations, got %d", len(violations))
	}
}

func TestLintDataValid(t *testing.T) {
	data := []byte(`---
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  annotations:
    olm.skipRange: ">=1.0.0 <2.0.0"
  name: test-operator.v2.0.0
  namespace: test-namespace
spec:
  displayName: Test Operator
`)
	violations, err := linter.LintData(data, "<stdin>", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations, got %d: %v", len(violations), violations)
	}
}

func TestLintDataInvalid(t *testing.T) {
	data := []byte(`---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  annotations:
    operatorframework.io/bundle-unpack-timeout: "not-a-duration"
  name: test
  namespace: test
spec:
  upgradeStrategy: Default
`)
	violations, err := linter.LintData(data, "<stdin>", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected at least one violation")
	}
	found := findViolation(violations, "operatorframework.io/bundle-unpack-timeout", "invalid duration")
	if !found {
		t.Error("expected violation for bad duration value from LintData")
	}
	if violations[0].File != "<stdin>" {
		t.Errorf("expected file to be <stdin>, got %q", violations[0].File)
	}
}

func TestLintDataMultiDocument(t *testing.T) {
	data := []byte(`---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  annotations:
    operatorframework.io/priorityclass: "system-cluster-critical"
  name: test
  namespace: test
spec:
  sourceType: grpc
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  annotations:
    operatorframework.io/bundle-unpack-timeout: "bad"
  name: test
  namespace: test
spec:
  upgradeStrategy: Default
`)
	violations, err := linter.LintData(data, "<stdin>", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := findViolation(violations, "operatorframework.io/bundle-unpack-timeout", "invalid duration")
	if !found {
		t.Error("expected violation from second document")
	}
}

func TestMalformedYAML(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/malformed_yaml.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected at least one violation for malformed YAML")
	}
	found := false
	for _, v := range violations {
		if v.Severity == rules.SeverityWarning && strings.Contains(v.Message, "YAML parse error") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected YAML parse error warning")
	}
}

func TestInvalidK8sResource(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/invalid_k8s_resource.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		for _, v := range violations {
			if v.Severity == rules.SeverityError {
				t.Errorf("invalid k8s resource (missing apiVersion) should be silently skipped, got error: %s", v.Message)
			}
		}
	}
}

func TestEmptyAnnotations(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/empty_annotations.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for empty annotations map, got %d", len(violations))
	}
}

func TestNonOLMAnnotationsOnly(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/non_olm_annotations_only.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for non-OLM annotations, got %d", len(violations))
	}
}

func TestMultiDocumentAllInvalid(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/multi_document_all_invalid.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) < 3 {
		t.Errorf("expected at least 3 violations (one per document), got %d", len(violations))
	}
	expectedViolations := []struct {
		annotation string
		message    string
	}{
		{"olm.unknown-annotation", "unknown OLM annotation"},
		{"operatorframework.io/bundle-unpack-timeout", "is not valid on Subscription"},
		{"olm.skipRange", "invalid semver range"},
	}
	for _, expected := range expectedViolations {
		if !findViolation(violations, expected.annotation, expected.message) {
			t.Errorf("expected violation for %s: %s", expected.annotation, expected.message)
		}
	}
}

func TestWhitespaceOnlyFile(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/whitespace_only.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for whitespace-only file, got %d", len(violations))
	}
}

func TestMultipleViolationsSingleResource(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/invalid/multiple_violations_single_resource.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) < 4 {
		t.Errorf("expected at least 4 violations, got %d", len(violations))
	}
	expectedViolations := []struct {
		annotation string
		message    string
		severity   rules.Severity
	}{
		{"olm.skipRange", "invalid semver range", rules.SeverityError},
		{"olm.unknown-annotation", "unknown OLM annotation", rules.SeverityError},
		{"OLM.providedAPIs", "wrong casing", rules.SeverityError},
		{"olm.operatorGroup", "controller-managed", rules.SeverityWarning},
	}
	for _, expected := range expectedViolations {
		found := false
		for _, v := range violations {
			if v.Annotation == expected.annotation && strings.Contains(v.Message, expected.message) {
				if v.Severity != expected.severity {
					t.Errorf("expected severity %s for %s, got %s", expected.severity, expected.annotation, v.Severity)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected violation for %s: %s", expected.annotation, expected.message)
		}
	}
}

func TestDeeplyNestedStructure(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../testdata/valid/deeply_nested_structure.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range violations {
		if v.Severity == rules.SeverityError {
			t.Errorf("unexpected error in deeply nested structure: %s: %s", v.Annotation, v.Message)
		}
	}
}

func TestNonExistentPath(t *testing.T) {
	_, err := linter.Run(linter.Options{
		Paths: []string{"/nonexistent/path"},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if !strings.Contains(err.Error(), "cannot access") {
		t.Errorf("expected 'cannot access' error, got: %v", err)
	}
}

func TestDirectoryWithNoYAMLFiles(t *testing.T) {
	violations, err := linter.Run(linter.Options{
		Paths: []string{"../../pkg"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for directory with no YAML files, got %d", len(violations))
	}
}

func TestStdinMultipleTimesError(t *testing.T) {
	_, err := linter.Run(linter.Options{
		Paths: []string{"-", "-"},
	})
	if err == nil {
		t.Fatal("expected error when stdin (-) specified multiple times")
	}
	if !strings.Contains(err.Error(), "stdin (-) can only be specified once") {
		t.Errorf("expected stdin-specific error, got: %v", err)
	}
}

func TestLintDataEmptyInput(t *testing.T) {
	violations, err := linter.LintData([]byte(""), "<test>", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for empty input, got %d", len(violations))
	}
}

func TestLintDataNilInput(t *testing.T) {
	violations, err := linter.LintData(nil, "<test>", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("expected no violations for nil input, got %d", len(violations))
	}
}

func findViolation(violations []linter.Violation, annotation, messageContains string) bool {
	for _, v := range violations {
		if v.Annotation == annotation {
			if messageContains == "" {
				return true
			}
			if strings.Contains(v.Message, messageContains) {
				return true
			}
		}
	}
	return false
}
