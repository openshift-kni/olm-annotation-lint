package linter_test

import (
	"context"
	"testing"

	"github.com/openshift-kni/olm-annotation-lint/pkg/linter"
)

func FuzzLintData(f *testing.F) {
	f.Add([]byte(`apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  annotations:
    olm.skipRange: ">=1.0.0 <2.0.0"
`))
	f.Add([]byte("invalid yaml {{{"))
	f.Add([]byte(`apiVersion: v1
kind: Test
metadata:
  annotations:
    olm.foo: "{{imbalanced"
`))
	f.Add([]byte(""))
	f.Add([]byte("annotations:\n  operators.operatorframework.io.bundle.mediatype.v1: registry+v1\n"))

	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = linter.LintData(context.Background(), data, "fuzz.yaml", nil)
	})
}
