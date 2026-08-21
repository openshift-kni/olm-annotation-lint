package rules

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

const KindBundleAnnotations = "BundleAnnotations"

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "notice"
	default:
		return "error"
	}
}

func ParseSeverity(s string) (Severity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "error":
		return SeverityError, nil
	case "warning", "warn":
		return SeverityWarning, nil
	case "info", "notice":
		return SeverityInfo, nil
	default:
		return 0, fmt.Errorf("unknown severity %q (supported: error, warning, info)", s)
	}
}

const (
	RuleUnknownAnnotation = "unknown-annotation"
	RuleCaseMismatch      = "case-mismatch"
	RuleAllowedOverride   = "allowed-override"
	RuleControllerManaged = "controller-managed"
	RuleWrongResourceKind = "wrong-resource-kind"
	RuleInvalidValue      = "invalid-value"
	RuleMissingAnnotation = "missing-annotation"
	RuleDuplicateKey      = "duplicate-key"
)

var RequiredBundleAnnotations []string

func init() {
	for _, r := range bundleAnnotations {
		if r.Required {
			RequiredBundleAnnotations = append(RequiredBundleAnnotations, r.Key)
		}
	}
}

type ValueFormat int

const (
	FormatString ValueFormat = iota
	FormatDuration
	FormatJSON
	FormatTemplate
	FormatSemverRange
	FormatBundleMediatype
	FormatCommaSeparated
)

type AnnotationRule struct {
	Key           string
	ResourceKinds []string
	UserSettable  bool
	Required      bool
	Format        ValueFormat
	Description   string
}

var userSettable = []AnnotationRule{
	{
		Key:           "operatorframework.io/bundle-unpack-timeout",
		ResourceKinds: []string{"OperatorGroup"},
		UserSettable:  true,
		Format:        FormatDuration,
		Description:   "Override the default bundle unpack job deadline",
	},
	{
		Key:           "operatorframework.io/bundle-unpack-min-retry-interval",
		ResourceKinds: []string{"OperatorGroup"},
		UserSettable:  true,
		Format:        FormatDuration,
		Description:   "Minimum wait before retrying a failed unpack job",
	},
	{
		Key:           "operatorframework.io/priorityclass",
		ResourceKinds: []string{"CatalogSource"},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Priority class for catalog pods",
	},
	{
		Key:           "olm.catalogImageTemplate",
		ResourceKinds: []string{"CatalogSource"},
		UserSettable:  true,
		Format:        FormatTemplate,
		Description:   "Template for catalog image with kube version variables",
	},
	{
		Key:           "olm.skipRange",
		ResourceKinds: []string{"ClusterServiceVersion"},
		UserSettable:  true,
		Format:        FormatSemverRange,
		Description:   "Version range to skip during upgrades",
	},
	{
		Key:           "olm.operatorframework.io/exclude-global-namespace-resolution",
		ResourceKinds: []string{"Subscription"},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Exclude subscription from global namespace resolution",
	},
	{
		Key:           "operatorframework.io/suggested-namespace",
		ResourceKinds: []string{"ClusterServiceVersion"},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Suggested namespace for operator installation (consumed by OpenShift Console)",
	},
	{
		Key:           "operatorframework.io/suggested-namespace-template",
		ResourceKinds: []string{"ClusterServiceVersion"},
		UserSettable:  true,
		Format:        FormatJSON,
		Description:   "JSON template for suggested namespace with labels and annotations (consumed by OpenShift Console)",
	},
	{
		Key:           "operatorframework.io/cluster-monitoring",
		ResourceKinds: []string{"ClusterServiceVersion"},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Enable cluster monitoring for the operator namespace (consumed by OpenShift Console)",
	},
}

var controllerManaged = []AnnotationRule{
	{
		Key:           "olm.operatorGroup",
		ResourceKinds: []string{"ClusterServiceVersion"},
		Description:   "Identifies which OperatorGroup owns this CSV",
	},
	{
		Key:           "olm.operatorNamespace",
		ResourceKinds: []string{"ClusterServiceVersion"},
		Description:   "Records the namespace of the managing OperatorGroup",
	},
	{
		Key:           "olm.targetNamespaces",
		ResourceKinds: []string{"OperatorGroup"},
		Description:   "Computed target namespaces for the OperatorGroup",
	},
	{
		Key:           "olm.providedAPIs",
		ResourceKinds: []string{"OperatorGroup"},
		Description:   "APIs provided by operators in the group",
	},
	{
		Key:           "olm.operatorframework.io/bundle-name",
		ResourceKinds: []string{"ClusterObjectSet"},
		Description:   "Bundle name set by operator-controller",
	},
	{
		Key:           "olm.operatorframework.io/bundle-version",
		ResourceKinds: []string{"ClusterObjectSet"},
		Description:   "Bundle version set by operator-controller",
	},
	{
		Key:           "olm.operatorframework.io/bundle-release",
		ResourceKinds: []string{"ClusterObjectSet"},
		Description:   "Bundle release set by operator-controller",
	},
	{
		Key:           "olm.operatorframework.io/bundle-reference",
		ResourceKinds: []string{"ClusterObjectSet"},
		Description:   "Image or catalog reference set by operator-controller",
	},
	{
		Key:           "olm.operatorframework.io/service-account-name",
		ResourceKinds: []string{"ClusterObjectSet"},
		Description:   "ServiceAccount name set by operator-controller",
	},
	{
		Key:           "olm.operatorframework.io/service-account-namespace",
		ResourceKinds: []string{"ClusterObjectSet"},
		Description:   "ServiceAccount namespace set by operator-controller",
	},
}

var bundleAnnotations = []AnnotationRule{
	{
		Key:           "operators.operatorframework.io.bundle.mediatype.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Required:      true,
		Format:        FormatBundleMediatype,
		Description:   "Bundle format type",
	},
	{
		Key:           "operators.operatorframework.io.bundle.manifests.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Required:      true,
		Format:        FormatString,
		Description:   "Path to manifests directory in the bundle image",
	},
	{
		Key:           "operators.operatorframework.io.bundle.metadata.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Required:      true,
		Format:        FormatString,
		Description:   "Path to metadata directory in the bundle image",
	},
	{
		Key:           "operators.operatorframework.io.bundle.package.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Required:      true,
		Format:        FormatString,
		Description:   "Operator package name",
	},
	{
		Key:           "operators.operatorframework.io.bundle.channels.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Required:      true,
		Format:        FormatCommaSeparated,
		Description:   "Comma-separated list of channels this bundle belongs to",
	},
	{
		Key:           "operators.operatorframework.io.bundle.channel.default.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Default channel for the operator",
	},
	{
		Key:           "operators.operatorframework.io.metrics.builder",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Builder tool and version used to create the bundle",
	},
	{
		Key:           "operators.operatorframework.io.metrics.mediatype.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Metrics format type",
	},
	{
		Key:           "operators.operatorframework.io.metrics.project_layout",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Project layout type",
	},
	{
		Key:           "operators.operatorframework.io.test.config.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Path to test configuration directory",
	},
	{
		Key:           "operators.operatorframework.io.test.mediatype.v1",
		ResourceKinds: []string{KindBundleAnnotations},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Test format type",
	},
}

var olmPrefixes = []string{
	"olm.",
	"operatorframework.io/",
	"operators.operatorframework.io.",
}

func IsOLMAnnotation(key string) bool {
	lower := strings.ToLower(key)
	for _, prefix := range olmPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func findRuleWith(key string, match func(a, b string) bool) (AnnotationRule, bool) {
	for _, r := range userSettable {
		if match(r.Key, key) {
			return r, true
		}
	}
	for _, r := range controllerManaged {
		if match(r.Key, key) {
			return r, true
		}
	}
	for _, r := range bundleAnnotations {
		if match(r.Key, key) {
			return r, true
		}
	}
	return AnnotationRule{}, false
}

func FindRule(key string) (AnnotationRule, bool) {
	return findRuleWith(key, func(a, b string) bool { return a == b })
}

func FindRuleCaseInsensitive(key string) (AnnotationRule, bool) {
	return findRuleWith(key, strings.EqualFold)
}

func IsValidResourceKind(rule AnnotationRule, kind string) bool {
	return slices.Contains(rule.ResourceKinds, kind)
}

func ValidateDuration(value string) bool {
	_, err := time.ParseDuration(value)
	return err == nil
}

func ValidateJSON(value string) bool {
	return json.Valid([]byte(value))
}

var allowedTemplateVars = map[string]bool{
	"kube_major_version": true,
	"kube_minor_version": true,
	"kube_patch_version": true,
}

func ValidateTemplate(value string) bool {
	depth := 0
	varNameStart := -1
	for i, ch := range value {
		switch ch {
		case '{':
			depth++
			if depth > 1 {
				return false
			}
			varNameStart = i + 1
		case '}':
			depth--
			if depth < 0 {
				return false
			}
			if varNameStart < 0 || !allowedTemplateVars[value[varNameStart:i]] {
				return false
			}
			varNameStart = -1
		}
	}
	return depth == 0
}

func ValidateSemverRange(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	operators := []string{">=", "<=", "!=", ">", "<", "="}
	for _, token := range strings.Fields(value) {
		for _, op := range operators {
			if strings.HasPrefix(token, op) {
				token = token[len(op):]
				break
			}
		}
		if !isVersion(token) {
			return false
		}
	}
	return true
}

func isVersion(s string) bool {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, "-", 2)
	segments := strings.Split(parts[0], ".")
	if len(segments) < 2 {
		return false
	}
	for _, seg := range segments {
		if _, err := strconv.Atoi(seg); err != nil {
			return false
		}
	}
	return true
}

var validBundleMediatypes = []string{"registry+v1", "plain+v0", "helm+v0"}

func ValidateBundleMediatype(value string) bool {
	return slices.Contains(validBundleMediatypes, value)
}

func ValidateCommaSeparated(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) == "" {
			return false
		}
	}
	return true
}

func formatName(f ValueFormat) string {
	switch f {
	case FormatDuration:
		return "duration"
	case FormatJSON:
		return "JSON"
	case FormatTemplate:
		return "template"
	case FormatSemverRange:
		return "semver range"
	case FormatBundleMediatype:
		return "bundle mediatype"
	case FormatCommaSeparated:
		return "comma-separated list"
	default:
		return "string"
	}
}

func PrintRules(w io.Writer) {
	_, _ = fmt.Fprintln(w, "User-settable annotations:")
	for _, r := range userSettable {
		printRule(w, r, true)
	}
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Controller-managed annotations (should not be set by users):")
	for _, r := range controllerManaged {
		printRule(w, r, true)
	}
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Bundle annotations (in metadata/annotations.yaml):")
	for _, r := range bundleAnnotations {
		printRule(w, r, false)
	}
}

func printRule(w io.Writer, r AnnotationRule, includeKind bool) {
	if includeKind {
		_, _ = fmt.Fprintf(w, "  %-65s %s  (%s)\n", r.Key, strings.Join(r.ResourceKinds, ", "), formatName(r.Format))
	} else {
		_, _ = fmt.Fprintf(w, "  %-65s (%s)\n", r.Key, formatName(r.Format))
	}
	if r.Description != "" {
		_, _ = fmt.Fprintf(w, "    %s\n", r.Description)
	}
}
