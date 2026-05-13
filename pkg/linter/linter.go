package linter

import (
	"bufio"
	"bytes"
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
	Message    string
}

type k8sResource struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   metadata `yaml:"metadata"`
}

type metadata struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations"`
}

type Options struct {
	Paths   []string
	Exclude []string
	Strict  bool
}

func Run(opts Options) ([]Violation, error) {
	var allViolations []Violation

	for _, p := range opts.Paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", p, err)
		}

		if info.IsDir() {
			violations, err := lintDirectory(p, opts.Exclude)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
		} else {
			violations, err := lintFile(p)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
		}
	}

	return allViolations, nil
}

func lintDirectory(dir string, exclude []string) ([]Violation, error) {
	var violations []Violation

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			for _, ex := range exclude {
				if matched, _ := filepath.Match(ex, info.Name()); matched {
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

		fileViolations, err := lintFile(path)
		if err != nil {
			return err
		}
		violations = append(violations, fileViolations...)
		return nil
	})

	return violations, err
}

func lintFile(path string) ([]Violation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	annotationLines := buildAnnotationLineMap(data)

	var violations []Violation
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		var resource k8sResource
		err := decoder.Decode(&resource)
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		if resource.APIVersion == "" || resource.Kind == "" {
			continue
		}

		for key, value := range resource.Metadata.Annotations {
			if !rules.IsOLMAnnotation(key) {
				continue
			}

			line := annotationLines[key]

			v := validateAnnotation(path, line, key, value, resource.Kind)
			violations = append(violations, v...)
		}
	}

	return violations, nil
}

func validateAnnotation(file string, line int, key, value, kind string) []Violation {
	var violations []Violation

	rule, found := rules.FindRule(key)
	if !found {
		caseRule, caseFound := rules.FindRuleCaseInsensitive(key)
		if caseFound {
			violations = append(violations, Violation{
				File:       file,
				Line:       line,
				Annotation: key,
				Kind:       kind,
				Severity:   rules.SeverityError,
				Message:    fmt.Sprintf("annotation has wrong casing, use %q", caseRule.Key),
			})
			return violations
		}

		violations = append(violations, Violation{
			File:       file,
			Line:       line,
			Annotation: key,
			Kind:       kind,
			Severity:   rules.SeverityError,
			Message:    "unknown OLM annotation",
		})
		return violations
	}

	if !rule.UserSettable {
		violations = append(violations, Violation{
			File:       file,
			Line:       line,
			Annotation: key,
			Kind:       kind,
			Severity:   rules.SeverityWarning,
			Message:    "annotation is controller-managed and should not be set by users",
		})
	}

	if !rules.IsValidResourceKind(rule, kind) {
		violations = append(violations, Violation{
			File:       file,
			Line:       line,
			Annotation: key,
			Kind:       kind,
			Severity:   rules.SeverityError,
			Message:    fmt.Sprintf("annotation is not valid on %s, expected one of: %s", kind, strings.Join(rule.ResourceKinds, ", ")),
		})
	}

	if rule.Format == rules.FormatDuration {
		if !rules.ValidateDuration(value) {
			violations = append(violations, Violation{
				File:       file,
				Line:       line,
				Annotation: key,
				Kind:       kind,
				Severity:   rules.SeverityError,
				Message:    fmt.Sprintf("invalid duration value %q, expected format like 10m, 1h30m, 5s", value),
			})
		}
	}

	return violations
}

func buildAnnotationLineMap(data []byte) map[string]int {
	lines := map[string]int{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			key := strings.SplitN(line, ":", 2)[0]
			key = strings.TrimSpace(key)
			if rules.IsOLMAnnotation(key) {
				lines[key] = lineNum
			}
		}
	}
	return lines
}
