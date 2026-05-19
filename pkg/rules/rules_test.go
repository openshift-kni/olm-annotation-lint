package rules_test

import (
	"testing"

	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
)

func TestIsOLMAnnotation(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"olm.providedAPIs", true},
		{"olm.skipRange", true},
		{"olm.operatorGroup", true},
		{"olm.operatorframework.io/bundle-install-timeout", true},
		{"operatorframework.io/bundle-unpack-timeout", true},
		{"operatorframework.io/bundle-unpack-min-retry-interval", true},
		{"OLM.providedAPIs", true},
		{"argocd.argoproj.io/sync-wave", false},
		{"ran.openshift.io/ztp-deploy-wave", false},
		{"kubectl.kubernetes.io/last-applied-configuration", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := rules.IsOLMAnnotation(tt.key)
			if got != tt.want {
				t.Errorf("IsOLMAnnotation(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestFindRule(t *testing.T) {
	tests := []struct {
		key   string
		found bool
	}{
		{"operatorframework.io/bundle-unpack-timeout", true},
		{"operatorframework.io/bundle-unpack-min-retry-interval", true},
		{"olm.skipRange", true},
		{"olm.providedAPIs", true},
		{"olm.operatorframework.io/bundle-install-timeout", false},
		{"operatorframework.io/made-up", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			_, found := rules.FindRule(tt.key)
			if found != tt.found {
				t.Errorf("FindRule(%q) found = %v, want %v", tt.key, found, tt.found)
			}
		})
	}
}

func TestFindRuleConsoleAnnotations(t *testing.T) {
	tests := []struct {
		key  string
		kind string
	}{
		{"operatorframework.io/suggested-namespace", "ClusterServiceVersion"},
		{"operatorframework.io/suggested-namespace-template", "ClusterServiceVersion"},
		{"operatorframework.io/cluster-monitoring", "ClusterServiceVersion"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			rule, found := rules.FindRule(tt.key)
			if !found {
				t.Fatalf("FindRule(%q) not found", tt.key)
			}
			if !rule.UserSettable {
				t.Errorf("expected %q to be user-settable", tt.key)
			}
			if !rules.IsValidResourceKind(rule, tt.kind) {
				t.Errorf("expected %q to be valid on %s", tt.key, tt.kind)
			}
		})
	}
}

func TestFindRuleCaseInsensitive(t *testing.T) {
	rule, found := rules.FindRuleCaseInsensitive("OLM.providedAPIs")
	if !found {
		t.Fatal("expected to find rule case-insensitively")
	}
	if rule.Key != "olm.providedAPIs" {
		t.Errorf("expected key olm.providedAPIs, got %s", rule.Key)
	}
}

func TestIsValidResourceKind(t *testing.T) {
	rule, _ := rules.FindRule("operatorframework.io/bundle-unpack-timeout")

	if !rules.IsValidResourceKind(rule, "OperatorGroup") {
		t.Error("expected OperatorGroup to be valid")
	}
	if rules.IsValidResourceKind(rule, "Subscription") {
		t.Error("expected Subscription to be invalid")
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"10m", true},
		{"1h30m", true},
		{"5s", true},
		{"0", true},
		{"not-a-duration", false},
		{"10", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := rules.ValidateDuration(tt.value)
			if got != tt.valid {
				t.Errorf("ValidateDuration(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestSeverityString(t *testing.T) {
	if rules.SeverityError.String() != "error" {
		t.Errorf("expected 'error', got %q", rules.SeverityError.String())
	}
	if rules.SeverityWarning.String() != "warning" {
		t.Errorf("expected 'warning', got %q", rules.SeverityWarning.String())
	}
	if rules.SeverityInfo.String() != "notice" {
		t.Errorf("expected 'notice', got %q", rules.SeverityInfo.String())
	}
}
