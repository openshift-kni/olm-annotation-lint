package rules_test

import (
	"bytes"
	"strings"
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
		{"operators.operatorframework.io.bundle.mediatype.v1", true},
		{"operators.operatorframework.io.bundle.channels.v1", true},
		{"operators.operatorframework.io.metrics.builder", true},
		{"operators.operatorframework.io.test.config.v1", true},
		{"Operators.Operatorframework.IO.Bundle.Mediatype.V1", true},
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
		{"operators.operatorframework.io.bundle.mediatype.v1", true},
		{"operators.operatorframework.io.bundle.channels.v1", true},
		{"operators.operatorframework.io.metrics.builder", true},
		{"operators.operatorframework.io.test.config.v1", true},
		{"olm.operatorframework.io/bundle-name", true},
		{"olm.operatorframework.io/bundle-version", true},
		{"olm.operatorframework.io/bundle-install-timeout", false},
		{"operatorframework.io/made-up", false},
		{"operators.operatorframework.io.bundle.nonexistent.v1", false},
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

func TestValidateJSON(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid object", `{"metadata":{"labels":{"key":"value"}}}`, true},
		{"valid array", `["a","b"]`, true},
		{"valid string", `"just a string"`, true},
		{"valid number", `42`, true},
		{"bare string", "not-json", false},
		{"truncated object", "{bad json", false},
		{"empty", "", false},
		{"unclosed brace", `{"unclosed": "brace"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rules.ValidateJSON(tt.value)
			if got != tt.valid {
				t.Errorf("ValidateJSON(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"with vars", "quay.io/example/catalog:v{kube_major_version}.{kube_minor_version}", true},
		{"plain string", "quay.io/example/catalog:latest", true},
		{"single known var", "image:{kube_patch_version}", true},
		{"empty", "", true},
		{"unknown var", "image:{tag}", false},
		{"typo var", "quay.io/example:{typo_variable}", false},
		{"empty braces", "image:{}", false},
		{"unclosed brace", "quay.io/example:{unclosed", false},
		{"extra close", "quay.io/example:closed}", false},
		{"nested braces", "quay.io/example:{{nested}}", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rules.ValidateTemplate(tt.value)
			if got != tt.valid {
				t.Errorf("ValidateTemplate(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestValidateSemverRange(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"range", ">=1.0.0 <2.0.0", true},
		{"single constraint", ">=1.2.3", true},
		{"exact version", "1.0.0", true},
		{"with pre-release", ">=1.0.0-rc1", true},
		{"two-part version", ">=1.0", true},
		{"with v prefix", ">=v1.0.0", true},
		{"bare string", "not-a-range", false},
		{"empty", "", false},
		{"only operator", ">=", false},
		{"single number", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rules.ValidateSemverRange(tt.value)
			if got != tt.valid {
				t.Errorf("ValidateSemverRange(%q) = %v, want %v", tt.value, got, tt.valid)
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

func TestValidateBundleMediatype(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"registry+v1", true},
		{"plain+v0", true},
		{"helm+v0", true},
		{"invalid-format", false},
		{"registry+v2", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := rules.ValidateBundleMediatype(tt.value)
			if got != tt.valid {
				t.Errorf("ValidateBundleMediatype(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestValidateCommaSeparated(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"single value", "alpha", true},
		{"multiple values", "alpha,beta,stable", true},
		{"with spaces", "alpha, beta", true},
		{"empty", "", false},
		{"only spaces", "   ", false},
		{"trailing comma", "alpha,", false},
		{"leading comma", ",alpha", false},
		{"consecutive commas", "alpha,,beta", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rules.ValidateCommaSeparated(tt.value)
			if got != tt.valid {
				t.Errorf("ValidateCommaSeparated(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestFindRuleBundleAnnotations(t *testing.T) {
	tests := []struct {
		key  string
		kind string
	}{
		{"operators.operatorframework.io.bundle.mediatype.v1", "BundleAnnotations"},
		{"operators.operatorframework.io.bundle.channels.v1", "BundleAnnotations"},
		{"operators.operatorframework.io.metrics.builder", "BundleAnnotations"},
		{"operators.operatorframework.io.test.config.v1", "BundleAnnotations"},
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

func TestFindRuleV1ControllerManaged(t *testing.T) {
	tests := []struct {
		key  string
		kind string
	}{
		{"olm.operatorframework.io/bundle-name", "ClusterObjectSet"},
		{"olm.operatorframework.io/bundle-version", "ClusterObjectSet"},
		{"olm.operatorframework.io/service-account-name", "ClusterObjectSet"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			rule, found := rules.FindRule(tt.key)
			if !found {
				t.Fatalf("FindRule(%q) not found", tt.key)
			}
			if rule.UserSettable {
				t.Errorf("expected %q to be controller-managed", tt.key)
			}
			if !rules.IsValidResourceKind(rule, tt.kind) {
				t.Errorf("expected %q to be valid on %s", tt.key, tt.kind)
			}
		})
	}
}

func TestPrintRules(t *testing.T) {
	var buf bytes.Buffer
	rules.PrintRules(&buf)
	output := buf.String()

	if !strings.Contains(output, "User-settable annotations:") {
		t.Error("expected user-settable header in output")
	}
	if !strings.Contains(output, "Controller-managed annotations") {
		t.Error("expected controller-managed header in output")
	}
	if !strings.Contains(output, "Bundle annotations") {
		t.Error("expected bundle annotations header in output")
	}
	if !strings.Contains(output, "operatorframework.io/bundle-unpack-timeout") {
		t.Error("expected bundle-unpack-timeout in output")
	}
	if !strings.Contains(output, "olm.operatorGroup") {
		t.Error("expected olm.operatorGroup in output")
	}
	if !strings.Contains(output, "operators.operatorframework.io.bundle.mediatype.v1") {
		t.Error("expected bundle mediatype in output")
	}
	if !strings.Contains(output, "olm.operatorframework.io/bundle-name") {
		t.Error("expected v1 controller-managed annotation in output")
	}
	if !strings.Contains(output, "(duration)") {
		t.Error("expected format type in output")
	}
	if !strings.Contains(output, "(bundle mediatype)") {
		t.Error("expected bundle mediatype format in output")
	}
	if !strings.Contains(output, "Override the default bundle unpack job deadline") {
		t.Error("expected user-settable description in output")
	}
	if !strings.Contains(output, "Identifies which OperatorGroup owns this CSV") {
		t.Error("expected controller-managed description in output")
	}
	if !strings.Contains(output, "Bundle format type") {
		t.Error("expected bundle annotation description in output")
	}
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		in      string
		want    rules.Severity
		wantErr bool
	}{
		{"error", rules.SeverityError, false},
		{"WARNING", rules.SeverityWarning, false},
		{"warn", rules.SeverityWarning, false},
		{"info", rules.SeverityInfo, false},
		{"notice", rules.SeverityInfo, false},
		{"banana", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := rules.ParseSeverity(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
