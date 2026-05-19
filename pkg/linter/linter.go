package linter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
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
	Message    string
}

type k8sResource struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   metadata `yaml:"metadata"`
}

type metadata struct {
	Annotations map[string]string `yaml:"annotations"`
}

type Options struct {
	Paths              []string
	Exclude            []string
	AllowedAnnotations []string
}

func Run(opts Options) ([]Violation, error) {
	var allViolations []Violation

	for _, p := range opts.Paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", p, err)
		}

		if info.IsDir() {
			violations, err := lintDirectory(p, opts.Exclude, opts.AllowedAnnotations)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
		} else {
			violations, err := lintFile(p, opts.AllowedAnnotations)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
		}
	}

	return allViolations, nil
}

func lintDirectory(dir string, exclude []string, allowedAnnotations []string) ([]Violation, error) {
	var violations []Violation

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			for _, ex := range exclude {
				if matched, _ := filepath.Match(ex, d.Name()); matched {
					return filepath.SkipDir
				}
				if strings.Contains(path, ex) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		fileViolations, err := lintFile(path, allowedAnnotations)
		if err != nil {
			return err
		}
		violations = append(violations, fileViolations...)
		return nil
	})

	return violations, err
}

func lintFile(path string, allowedAnnotations []string) ([]Violation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var violations []Violation
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			violations = append(violations, Violation{
				File:     path,
				Severity: rules.SeverityWarning,
				Message:  fmt.Sprintf("YAML parse error: %v", err),
			})
			break
		}

		var resource k8sResource
		if err := node.Decode(&resource); err != nil {
			violations = append(violations, Violation{
				File:     path,
				Line:     node.Line,
				Severity: rules.SeverityWarning,
				Message:  fmt.Sprintf("cannot decode as Kubernetes resource: %v", err),
			})
			continue
		}

		if resource.APIVersion == "" || resource.Kind == "" {
			continue
		}

		annotationLines := extractAnnotationLines(&node)

		for key, value := range resource.Metadata.Annotations {
			if !rules.IsOLMAnnotation(key) {
				continue
			}

			line := annotationLines[key]

			v := validateAnnotation(path, line, key, value, resource.Kind, allowedAnnotations)
			violations = append(violations, v...)
		}
	}

	return violations, nil
}

func validateAnnotation(file string, line int, key, value, kind string, allowedAnnotations []string) []Violation {
	var violations []Violation

	newViolation := func(sev rules.Severity, msg string) Violation {
		return Violation{
			File: file, Line: line, Annotation: key, Kind: kind,
			Severity: sev, Message: msg,
		}
	}

	rule, found := rules.FindRule(key)
	if !found {
		caseRule, caseFound := rules.FindRuleCaseInsensitive(key)
		if caseFound {
			violations = append(violations, newViolation(rules.SeverityError,
				fmt.Sprintf("annotation has wrong casing, use %q", caseRule.Key)))
			return violations
		}

		if slices.Contains(allowedAnnotations, key) {
			violations = append(violations, newViolation(rules.SeverityInfo,
				fmt.Sprintf("annotation %q allowed via user override", key)))
			return violations
		}

		violations = append(violations, newViolation(rules.SeverityError, "unknown OLM annotation"))
		return violations
	}

	if !rule.UserSettable {
		violations = append(violations, newViolation(rules.SeverityWarning,
			"annotation is controller-managed and should not be set by users"))
	}

	if !rules.IsValidResourceKind(rule, kind) {
		violations = append(violations, newViolation(rules.SeverityError,
			fmt.Sprintf("annotation is not valid on %s, expected one of: %s", kind, strings.Join(rule.ResourceKinds, ", "))))
	}

	if rule.Format == rules.FormatDuration && !rules.ValidateDuration(value) {
		violations = append(violations, newViolation(rules.SeverityError,
			fmt.Sprintf("invalid duration value %q, expected format like 10m, 1h30m, 5s", value)))
	}

	return violations
}

func extractAnnotationLines(root *yaml.Node) map[string]int {
	lines := map[string]int{}
	metadataNode := findMappingValue(root, "metadata")
	if metadataNode == nil {
		return lines
	}
	annotationsNode := findMappingValue(metadataNode, "annotations")
	if annotationsNode == nil || annotationsNode.Kind != yaml.MappingNode {
		return lines
	}
	for i := 0; i+1 < len(annotationsNode.Content); i += 2 {
		lines[annotationsNode.Content[i].Value] = annotationsNode.Content[i].Line
	}
	return lines
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
