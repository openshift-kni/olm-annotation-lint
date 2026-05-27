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
