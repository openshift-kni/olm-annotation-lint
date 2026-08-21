package linter_test

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
)

func BenchmarkLintDataCSV(b *testing.B) {
	data, err := os.ReadFile("../../testdata/valid/csv_with_skip_range.yaml")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := linter.LintData(context.Background(), data, "bench.yaml", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLintDataBundle(b *testing.B) {
	data, err := os.ReadFile("../../testdata/valid/bundle_annotations.yaml")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := linter.LintData(context.Background(), data, "bench.yaml", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRunValidDir(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, err := linter.Run(context.Background(), linter.Options{Paths: []string{"../../testdata/valid"}})
		if err != nil {
			b.Fatal(err)
		}
	}
}
