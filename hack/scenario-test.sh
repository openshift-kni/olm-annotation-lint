#!/usr/bin/env bash
# CLI scenario tests. Feature-specific smokes run only when that feature
# (flag, fixture, or source) is present so this script is safe on main and
# on feature branches.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BINARY="${BINARY:-./olm-annotation-lint}"

fail() {
	echo "FAIL: $*"
	exit 1
}

pass() {
	echo "PASS: $*"
}

skip() {
	echo "SKIP: $*"
}

if [[ ! -x "$BINARY" ]]; then
	fail "binary not found or not executable: $BINARY"
fi

source_has() {
	local pattern="$1"
	local file="$2"
	[[ -f "$file" ]] && grep -qE "$pattern" "$file"
}

HELP="$("$BINARY" --help 2>&1 || true)"

has_flag() {
	# Go's flag package prints long names with a single dash (-timeout).
	printf '%s\n' "$HELP" | grep -qE -- "(^|[[:space:]])${1}([[:space:]]|$)"
}

echo "=== Testing valid YAMLs ==="
if ! "$BINARY" --path testdata/valid; then
	fail "valid YAMLs rejected"
fi
pass "all valid YAMLs accepted"

echo ""
echo "=== Testing invalid YAMLs (strict mode) ==="
shopt -s nullglob
invalid_files=(testdata/invalid/*.yaml)
if [[ ${#invalid_files[@]} -eq 0 ]]; then
	fail "no testdata/invalid/*.yaml fixtures found"
fi
for f in "${invalid_files[@]}"; do
	if "$BINARY" --path "$f" --strict >/dev/null 2>&1; then
		fail "$f should have been rejected"
	fi
	pass "$f correctly rejected"
done

echo ""
echo "=== Testing --version flag ==="
VERSION_OUT="$("$BINARY" --version)"
if [[ -z "$VERSION_OUT" ]]; then
	fail "--version produced no output"
fi
pass "--version prints '$VERSION_OUT'"

echo ""
echo "=== Testing --list-rules flag ==="
RULES_OUT="$("$BINARY" --list-rules)"
if ! printf '%s\n' "$RULES_OUT" | grep -q "User-settable annotations"; then
	fail "--list-rules missing expected output"
fi
pass "--list-rules prints annotation list"

if source_has 'r\.Description' pkg/rules/rules.go; then
	if ! printf '%s\n' "$RULES_OUT" | grep -q "Override the default bundle unpack job deadline"; then
		fail "--list-rules missing user-settable description"
	fi
	pass "--list-rules prints annotation descriptions"
fi

if source_has 'exampleValues' pkg/rules/rules.go; then
	if ! printf '%s\n' "$RULES_OUT" | grep -q "Examples: 10m"; then
		fail "--list-rules missing duration examples"
	fi
	if ! printf '%s\n' "$RULES_OUT" | grep -q "Since: v1.0.0"; then
		fail "--list-rules missing Since version"
	fi
	pass "--list-rules prints rule metadata"
fi

echo ""
echo "=== Nested bundle directories ==="
if [[ -d testdata/valid/bundles/matching ]]; then
	if ! "$BINARY" --path testdata/valid/bundles/matching; then
		fail "matching bundle directory should pass"
	fi
	pass "testdata/valid/bundles/matching accepted"
else
	skip "testdata/valid/bundles/matching not present"
fi

if [[ -d testdata/invalid/bundles/mismatch ]]; then
	if "$BINARY" --path testdata/invalid/bundles/mismatch --strict >/dev/null 2>&1; then
		fail "mismatched bundle directory should be rejected in strict mode"
	fi
	pass "testdata/invalid/bundles/mismatch rejected in strict mode"
else
	skip "testdata/invalid/bundles/mismatch not present"
fi

echo ""
echo "=== Inline ignore directives ==="
if [[ -f testdata/invalid/ignore_directive_mixed.yaml ]]; then
	code=0
	ignore_out="$("$BINARY" --path testdata/invalid/ignore_directive_mixed.yaml --strict 2>&1)" || code=$?
	if [[ "$code" -eq 0 ]]; then
		fail "ignore_directive_mixed.yaml should fail in strict mode"
	fi
	if printf '%s\n' "$ignore_out" | grep -q "olm.operatorframework.io/bundle-install-timeout"; then
		fail "ignored annotation was still reported"
	fi
	if ! printf '%s\n' "$ignore_out" | grep -q "olm.operatorframework.io/not-ignored"; then
		fail "non-ignored unknown annotation was not reported"
	fi
	pass "ignore directives suppress only the annotated keys"
else
	skip "testdata/invalid/ignore_directive_mixed.yaml not present"
fi

echo ""
echo "=== --timeout flag ==="
if has_flag -timeout; then
	if ! "$BINARY" --timeout 1h --path testdata/valid >/dev/null; then
		fail "--timeout 1h on testdata/valid should succeed"
	fi
	code=0
	"$BINARY" --timeout not-a-duration >/dev/null 2>&1 || code=$?
	if [[ "$code" -ne 2 ]]; then
		fail "--timeout not-a-duration should exit 2, got $code"
	fi
	pass "--timeout accepts durations and rejects invalid values"
else
	skip "--timeout not in --help"
fi

echo ""
echo "=== Case-mismatch suggestions ==="
if source_has 'Suggestion:' pkg/reporter/reporter.go; then
	code=0
	suggest_out="$("$BINARY" --path testdata/invalid/case_mismatch.yaml 2>&1)" || code=$?
	if [[ "$code" -eq 0 ]]; then
		fail "case_mismatch.yaml should produce errors"
	fi
	if ! printf '%s\n' "$suggest_out" | grep -q "Suggestion: olm.providedAPIs"; then
		fail "expected Suggestion: olm.providedAPIs in text output"
	fi
	pass "case mismatch prints Suggestion: olm.providedAPIs"
else
	skip "Suggestion reporter not present"
fi

echo ""
echo "=== Per-rule configuration ==="
if source_has 'type ruleConfig struct' main.go; then
	tmp="$(mktemp -d)"

	cat >"$tmp/bad-severity.yaml" <<'EOF'
path: testdata/valid
rules:
  olm.skipRange:
    severity: banana
EOF
	code=0
	"$BINARY" --config "$tmp/bad-severity.yaml" >/dev/null 2>&1 || code=$?
	if [[ "$code" -ne 2 ]]; then
		fail "invalid rule severity should exit 2, got $code"
	fi

	cat >"$tmp/disable.yaml" <<'EOF'
path: testdata/invalid/controller_managed_annotation.yaml
rules:
  olm.operatorGroup:
    enabled: false
EOF
	code=0
	disable_out="$("$BINARY" --config "$tmp/disable.yaml" 2>&1)" || code=$?
	if [[ "$code" -ne 0 ]]; then
		fail "disabling olm.operatorGroup should exit 0, got $code: $disable_out"
	fi
	if printf '%s\n' "$disable_out" | grep -q "olm.operatorGroup"; then
		fail "disabled olm.operatorGroup still reported"
	fi
	pass "per-rule config disables rules and rejects bad severity"
	rm -rf "$tmp"
else
	skip "per-rule config not present"
fi

echo ""
echo "=== Exclude file globs ==="
if [[ -f testdata/invalid/custom.generated.yaml ]]; then
	code=0
	exclude_out="$("$BINARY" --path testdata/invalid --exclude '*.generated.yaml' --strict 2>&1)" || code=$?
	if [[ "$code" -eq 0 ]]; then
		fail "excluding *.generated.yaml should still fail on other invalid files"
	fi
	if printf '%s\n' "$exclude_out" | grep -q "custom.generated.yaml"; then
		fail "excluded custom.generated.yaml was still reported"
	fi
	pass "exclude glob skips *.generated.yaml"
else
	skip "testdata/invalid/custom.generated.yaml not present"
fi

echo ""
echo "=== JUnit severity mapping ==="
if source_has 'system-err' pkg/reporter/reporter.go; then
	junit_warn="$("$BINARY" --format junit --path testdata/invalid/controller_managed_annotation.yaml 2>/dev/null || true)"
	if ! printf '%s\n' "$junit_warn" | grep -q "<system-err>"; then
		fail "warning-only JUnit output missing <system-err>"
	fi
	if printf '%s\n' "$junit_warn" | grep -q "<failure"; then
		fail "warning-only JUnit output should not include <failure>"
	fi
	if ! printf '%s\n' "$junit_warn" | grep -q 'failures="0"'; then
		fail "warning-only JUnit output should have failures=\"0\""
	fi

	code=0
	junit_mixed="$("$BINARY" --format junit --path testdata/invalid/unknown_olm_annotation.yaml,testdata/invalid/controller_managed_annotation.yaml 2>/dev/null)" || code=$?
	if [[ "$code" -eq 0 ]]; then
		fail "mixed error/warning JUnit run should exit non-zero"
	fi
	if ! printf '%s\n' "$junit_mixed" | grep -q 'failures="1"'; then
		fail "mixed JUnit output should have failures=\"1\""
	fi
	if ! printf '%s\n' "$junit_mixed" | grep -q "<system-err>"; then
		fail "mixed JUnit output missing <system-err> for the warning"
	fi
	pass "JUnit maps errors to failures and warnings to system-err"
else
	skip "JUnit system-err mapping not present"
fi

echo ""
echo "=== Resource metadata.name in output ==="
if source_has 'json:"name,omitempty"' pkg/reporter/reporter.go; then
	code=0
	name_out="$("$BINARY" --path testdata/invalid/unknown_olm_annotation.yaml 2>&1)" || code=$?
	if [[ "$code" -eq 0 ]]; then
		fail "unknown_olm_annotation.yaml should produce errors"
	fi
	if ! printf '%s\n' "$name_out" | grep -q "cert-manager-operator"; then
		fail "expected resource name cert-manager-operator in text output"
	fi
	pass "violation output includes metadata.name"
else
	skip "resource name reporter field not present"
fi

echo ""
echo "=== --output / -o flag ==="
if has_flag -output; then
	tmp="$(mktemp -d)"
	out_file="$tmp/results.json"
	code=0
	stdout="$("$BINARY" --path testdata/invalid/unknown_olm_annotation.yaml --format json --output "$out_file" 2>/dev/null)" || code=$?
	if [[ "$code" -ne 1 ]]; then
		fail "--output json on invalid testdata should exit 1, got $code"
	fi
	if printf '%s\n' "$stdout" | grep -q '"violations"'; then
		fail "JSON should be written to the file, not stdout"
	fi
	if [[ ! -f "$out_file" ]] || ! grep -q '"violations"' "$out_file"; then
		fail "expected JSON report in $out_file"
	fi
	pass "--output writes JSON to a file"
	rm -rf "$tmp"
else
	skip "--output not in --help"
fi

echo ""
echo "=== --allow wildcards ==="
if source_has 'only a trailing \* wildcard' pkg/linter/linter.go; then
	code=0
	"$BINARY" --allow 'olm.operatorframework.io/*' --path testdata/invalid/unknown_olm_annotation.yaml >/dev/null 2>&1 || code=$?
	if [[ "$code" -ne 0 ]]; then
		fail "--allow prefix wildcard should suppress unknown-annotation errors, got exit $code"
	fi
	code=0
	"$BINARY" --allow 'olm.*.foo' --path testdata/valid >/dev/null 2>&1 || code=$?
	if [[ "$code" -ne 2 ]]; then
		fail "--allow olm.*.foo should exit 2, got $code"
	fi
	pass "--allow wildcards match prefixes and reject invalid patterns"
else
	skip "allow wildcard matching not present"
fi

echo ""
echo "All scenario tests passed."
