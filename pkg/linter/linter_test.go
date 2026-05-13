package linter_test

import (
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

	var found bool
	for _, v := range violations {
		if v.Annotation == "olm.operatorGroup" && v.Severity == rules.SeverityWarning {
			found = true
			break
		}
	}
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

func findViolation(violations []linter.Violation, annotation, messageContains string) bool {
	for _, v := range violations {
		if v.Annotation == annotation {
			if messageContains == "" {
				return true
			}
			if contains(v.Message, messageContains) {
				return true
			}
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
