package rules_test

import (
	"testing"

	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
)

func FuzzValidateSemverRange(f *testing.F) {
	f.Add(">=1.0.0 <2.0.0")
	f.Add("1.0")
	f.Add("")
	f.Add(">=v1.2.3-rc1")
	f.Add("not-a-range")

	f.Fuzz(func(_ *testing.T, value string) {
		_ = rules.ValidateSemverRange(value)
	})
}

func FuzzValidateTemplate(f *testing.F) {
	f.Add("{kube_major_version}")
	f.Add("{{nested}}")
	f.Add("")
	f.Add("quay.io/example/catalog:v{kube_major_version}.{kube_minor_version}")

	f.Fuzz(func(_ *testing.T, value string) {
		_ = rules.ValidateTemplate(value)
	})
}

func FuzzValidateDuration(f *testing.F) {
	f.Add("10m")
	f.Add("1h30m")
	f.Add("")
	f.Add("not-a-duration")

	f.Fuzz(func(_ *testing.T, value string) {
		_ = rules.ValidateDuration(value)
	})
}

func FuzzValidateJSON(f *testing.F) {
	f.Add(`{"foo":"bar"}`)
	f.Add("{")
	f.Add("")

	f.Fuzz(func(_ *testing.T, value string) {
		_ = rules.ValidateJSON(value)
	})
}
