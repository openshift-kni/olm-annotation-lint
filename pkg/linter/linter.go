package linter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift-kni/olm-annotation-lint/pkg/rules"
	"gopkg.in/yaml.v3"
)

type Violation struct {
	File       string
	Line       int
	Annotation string
	Kind       string
	Severity   rules.Severity
	Rule       string
	Message    string
	Suggestion string
}

type k8sResource struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   metadata `yaml:"metadata"`
}

type metadata struct {
	Annotations map[string]string `yaml:"annotations"`
}

type bundleAnnotationsFile struct {
	Annotations map[string]string `yaml:"annotations"`
}

type Options struct {
	Paths              []string
	Exclude            []string
	AllowedAnnotations []string
	Rules              map[string]RuleConfig
}

// RuleConfig overrides whether a rule or annotation is enabled and its severity.
// Map keys may be an annotation key (e.g. "olm.skipRange") or a rule ID
// (e.g. "unknown-annotation").
type RuleConfig struct {
	Enabled  *bool
	Severity *rules.Severity
}

func Run(ctx context.Context, opts Options) ([]Violation, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var allViolations []Violation
	stdinConsumed := false

	for _, p := range opts.Paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if p == "-" {
			if stdinConsumed {
				return nil, fmt.Errorf("stdin (-) can only be specified once")
			}
			stdinConsumed = true
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("reading stdin: %w", err)
			}
			violations, err := LintData(ctx, data, "<stdin>", opts.AllowedAnnotations)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
			continue
		}

		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", p, err)
		}

		if info.IsDir() {
			violations, err := lintDirectory(ctx, p, opts.Exclude, opts.AllowedAnnotations)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
		} else {
			violations, err := lintFile(ctx, p, opts.AllowedAnnotations)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
		}
	}

	return applyRuleConfig(allViolations, opts.Rules), nil
}

func lintDirectory(ctx context.Context, dir string, exclude []string, allowedAnnotations []string) ([]Violation, error) {
	var violations []Violation

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		skip, err := matchesExclude(d.Name(), exclude)
		if err != nil {
			return err
		}
		if skip {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		fileViolations, err := lintFile(ctx, path, allowedAnnotations)
		if err != nil {
			return err
		}
		violations = append(violations, fileViolations...)
		return nil
	})

	return violations, err
}

func matchesExclude(name string, exclude []string) (bool, error) {
	for _, ex := range exclude {
		matched, err := filepath.Match(ex, name)
		if err != nil {
			return false, fmt.Errorf("invalid exclude pattern %q: %w", ex, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func lintFile(ctx context.Context, path string, allowedAnnotations []string) ([]Violation, error) {
	data, err := os.ReadFile(path) //nolint:gosec // lint target path is user-specified CLI input
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return LintData(ctx, data, path, allowedAnnotations)
}

func LintData(ctx context.Context, data []byte, source string, allowedAnnotations []string) ([]Violation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	allowSet := make(map[string]bool, len(allowedAnnotations))
	for _, a := range allowedAnnotations {
		allowSet[a] = true
	}

	var violations []Violation
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var node yaml.Node
		err := decoder.Decode(&node)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			violations = append(violations, Violation{
				File:     source,
				Severity: rules.SeverityWarning,
				Message:  fmt.Sprintf("YAML parse error: %v", err),
			})
			break
		}

		var resource k8sResource
		if err := node.Decode(&resource); err != nil {
			violations = append(violations, Violation{
				File:     source,
				Line:     node.Line,
				Severity: rules.SeverityWarning,
				Message:  fmt.Sprintf("cannot decode as Kubernetes resource: %v", err),
			})
			continue
		}

		if resource.APIVersion == "" || resource.Kind == "" {
			bundleViolations := lintBundleAnnotations(&node, source, allowSet)
			if bundleViolations != nil {
				violations = append(violations, bundleViolations...)
			}
			continue
		}

		annotationLines := extractAnnotationLines(&node)
		ignores := extractIgnoreDirectives(findMappingValue(findMappingValue(&node, "metadata"), "annotations"))

		for key, value := range resource.Metadata.Annotations {
			if !rules.IsOLMAnnotation(key) {
				continue
			}
			if ignores.skipAll(key) {
				continue
			}

			line := annotationLines[key]

			v := validateAnnotation(source, line, key, value, resource.Kind, allowSet)
			violations = append(violations, ignores.filter(key, v)...)
		}
	}

	return violations, nil
}

func validateAnnotation(file string, line int, key, value, kind string, allowedAnnotations map[string]bool) []Violation {
	var violations []Violation

	newViolation := func(sev rules.Severity, ruleID, msg string) Violation {
		return Violation{
			File: file, Line: line, Annotation: key, Kind: kind,
			Severity: sev, Rule: ruleID, Message: msg,
		}
	}

	rule, found := rules.FindRule(key)
	if !found {
		caseRule, caseFound := rules.FindRuleCaseInsensitive(key)
		if caseFound {
			violations = append(violations, Violation{
				File: file, Line: line, Annotation: key, Kind: kind,
				Severity:   rules.SeverityError,
				Rule:       rules.RuleCaseMismatch,
				Message:    fmt.Sprintf("annotation case mismatch: use %q instead of %q", caseRule.Key, key),
				Suggestion: caseRule.Key,
			})
			return violations
		}

		if allowedAnnotations[key] {
			violations = append(violations, newViolation(rules.SeverityInfo, rules.RuleAllowedOverride,
				fmt.Sprintf("annotation %q allowed via user override", key)))
			return violations
		}

		violations = append(violations, newViolation(rules.SeverityError, rules.RuleUnknownAnnotation,
			fmt.Sprintf("unknown OLM annotation (use --allow %s to bypass this error)", key)))
		return violations
	}

	if !rule.UserSettable {
		violations = append(violations, newViolation(rules.SeverityWarning, rules.RuleControllerManaged,
			"annotation is controller-managed and should not be set by users"))
	}

	if !rules.IsValidResourceKind(rule, kind) {
		violations = append(violations, newViolation(rules.SeverityError, rules.RuleWrongResourceKind,
			fmt.Sprintf("annotation is not valid on %s, expected one of: %s", kind, strings.Join(rule.ResourceKinds, ", "))))
	}

	switch rule.Format {
	case rules.FormatDuration:
		if !rules.ValidateDuration(value) {
			violations = append(violations, newViolation(rules.SeverityError, rules.RuleInvalidValue,
				fmt.Sprintf("invalid duration value %q, expected format like 10m, 1h30m, 5s", value)))
		}
	case rules.FormatJSON:
		if !rules.ValidateJSON(value) {
			violations = append(violations, newViolation(rules.SeverityError, rules.RuleInvalidValue,
				fmt.Sprintf("invalid JSON value %q", value)))
		}
	case rules.FormatTemplate:
		if !rules.ValidateTemplate(value) {
			violations = append(violations, newViolation(rules.SeverityError, rules.RuleInvalidValue,
				fmt.Sprintf("invalid template value %q, unbalanced curly braces", value)))
		}
	case rules.FormatSemverRange:
		if !rules.ValidateSemverRange(value) {
			violations = append(violations, newViolation(rules.SeverityError, rules.RuleInvalidValue,
				fmt.Sprintf("invalid semver range %q, expected format like >=1.0.0 <2.0.0", value)))
		}
	case rules.FormatBundleMediatype:
		if !rules.ValidateBundleMediatype(value) {
			violations = append(violations, newViolation(rules.SeverityError, rules.RuleInvalidValue,
				fmt.Sprintf("invalid bundle mediatype %q, expected one of: registry+v1, plain+v0, helm+v0", value)))
		}
	case rules.FormatCommaSeparated:
		if !rules.ValidateCommaSeparated(value) {
			violations = append(violations, newViolation(rules.SeverityError, rules.RuleInvalidValue,
				fmt.Sprintf("invalid comma-separated list %q, expected non-empty comma-separated values", value)))
		}
	}

	return violations
}

func extractAnnotationLines(root *yaml.Node) map[string]int {
	metadataNode := findMappingValue(root, "metadata")
	return annotationLinesFromNode(findMappingValue(metadataNode, "annotations"))
}

func annotationLinesFromNode(node *yaml.Node) map[string]int {
	lines := map[string]int{}
	if node == nil || node.Kind != yaml.MappingNode {
		return lines
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		lines[node.Content[i].Value] = node.Content[i].Line
	}
	return lines
}

const ignoreDirectivePrefix = "olm-annotation-lint: ignore"

type ignoreSpec struct {
	all   bool
	rules map[string]bool
}

type ignoreSet map[string]ignoreSpec

func (s ignoreSet) skipAll(key string) bool {
	spec, ok := s[key]
	return ok && spec.all
}

func (s ignoreSet) filter(key string, vs []Violation) []Violation {
	spec, ok := s[key]
	if !ok || spec.all {
		return vs
	}
	var filtered []Violation
	for _, v := range vs {
		if spec.rules[v.Rule] {
			continue
		}
		filtered = append(filtered, v)
	}
	return filtered
}

func extractIgnoreDirectives(node *yaml.Node) ignoreSet {
	ignores := ignoreSet{}
	if node == nil || node.Kind != yaml.MappingNode {
		return ignores
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode, valNode := node.Content[i], node.Content[i+1]
		spec, ok := parseIgnoreComments(keyNode.HeadComment, keyNode.LineComment, keyNode.FootComment, valNode.HeadComment, valNode.LineComment, valNode.FootComment)
		if ok {
			ignores[keyNode.Value] = spec
		}
	}
	return ignores
}

func parseIgnoreComments(comments ...string) (ignoreSpec, bool) {
	var spec ignoreSpec
	found := false
	for _, comment := range comments {
		for _, line := range strings.Split(comment, "\n") {
			line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
			if !strings.HasPrefix(line, ignoreDirectivePrefix) {
				continue
			}
			found = true
			rest := strings.TrimSpace(strings.TrimPrefix(line, ignoreDirectivePrefix))
			if rest == "" {
				spec.all = true
				continue
			}
			if spec.rules == nil {
				spec.rules = map[string]bool{}
			}
			for _, ruleID := range strings.Fields(rest) {
				spec.rules[strings.Trim(ruleID, ",")] = true
			}
		}
	}
	if spec.all {
		spec.rules = nil
	}
	return spec, found
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return findMappingValue(node.Content[0], key)
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

const bundleAnnotationPrefix = "operators.operatorframework.io.bundle."

func lintBundleAnnotations(node *yaml.Node, source string, allowSet map[string]bool) []Violation {
	var bundle bundleAnnotationsFile
	if err := node.Decode(&bundle); err != nil || len(bundle.Annotations) == 0 {
		return nil
	}

	hasBundleKey := false
	for key := range bundle.Annotations {
		if strings.HasPrefix(key, bundleAnnotationPrefix) {
			hasBundleKey = true
			break
		}
	}
	if !hasBundleKey {
		return nil
	}

	annNode := findMappingValue(node, "annotations")
	annotationLines := annotationLinesFromNode(annNode)
	ignores := extractIgnoreDirectives(annNode)
	var violations []Violation

	for key, value := range bundle.Annotations {
		if !rules.IsOLMAnnotation(key) {
			continue
		}
		if ignores.skipAll(key) {
			continue
		}
		line := annotationLines[key]
		v := validateAnnotation(source, line, key, value, rules.KindBundleAnnotations, allowSet)
		violations = append(violations, ignores.filter(key, v)...)
	}

	for _, req := range rules.RequiredBundleAnnotations {
		if _, exists := bundle.Annotations[req]; !exists {
			violations = append(violations, Violation{
				File:       source,
				Annotation: req,
				Kind:       rules.KindBundleAnnotations,
				Severity:   rules.SeverityWarning,
				Rule:       rules.RuleMissingAnnotation,
				Message:    fmt.Sprintf("required bundle annotation %q is missing", req),
			})
		}
	}

	return violations
}

func applyRuleConfig(violations []Violation, cfg map[string]RuleConfig) []Violation {
	if len(cfg) == 0 {
		return violations
	}
	var filtered []Violation
	for _, v := range violations {
		rc, ok := lookupRuleConfig(cfg, v)
		if !ok {
			filtered = append(filtered, v)
			continue
		}
		if rc.Enabled != nil && !*rc.Enabled {
			continue
		}
		if rc.Severity != nil {
			v.Severity = *rc.Severity
		}
		filtered = append(filtered, v)
	}
	return filtered
}

func lookupRuleConfig(cfg map[string]RuleConfig, v Violation) (RuleConfig, bool) {
	if rc, ok := cfg[v.Annotation]; ok {
		return rc, true
	}
	if rc, ok := cfg[v.Rule]; ok {
		return rc, true
	}
	return RuleConfig{}, false
}
