package rules_test

import (
	"testing"

	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
)

func BenchmarkFindRule(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = rules.FindRule("olm.skipRange")
	}
}

func BenchmarkFindRuleMissing(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = rules.FindRule("olm.does-not-exist")
	}
}

func BenchmarkFindRuleCaseInsensitive(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = rules.FindRuleCaseInsensitive("OLM.SKIPRANGE")
	}
}
