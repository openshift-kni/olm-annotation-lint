package rules

import (
	"strings"
	"time"
)

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

type ValueFormat int

const (
	FormatString ValueFormat = iota
	FormatDuration
	FormatJSON
	FormatCSV
	FormatTemplate
)

type AnnotationRule struct {
	Key           string
	ResourceKinds []string
	UserSettable  bool
	Format        ValueFormat
	Description   string
}

var UserSettable = []AnnotationRule{
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
		Format:        FormatString,
		Description:   "Version range to skip during upgrades",
	},
	{
		Key:           "olm.operatorframework.io/exclude-global-namespace-resolution",
		ResourceKinds: []string{"Subscription"},
		UserSettable:  true,
		Format:        FormatString,
		Description:   "Exclude subscription from global namespace resolution",
	},
}

var ControllerManaged = []AnnotationRule{
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
}

var olmPrefixes = []string{
	"olm.",
	"operatorframework.io/",
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

func FindRule(key string) (*AnnotationRule, bool) {
	for i := range UserSettable {
		if UserSettable[i].Key == key {
			return &UserSettable[i], true
		}
	}
	for i := range ControllerManaged {
		if ControllerManaged[i].Key == key {
			return &ControllerManaged[i], true
		}
	}
	return nil, false
}

func FindRuleCaseInsensitive(key string) (*AnnotationRule, bool) {
	lower := strings.ToLower(key)
	for i := range UserSettable {
		if strings.ToLower(UserSettable[i].Key) == lower {
			return &UserSettable[i], true
		}
	}
	for i := range ControllerManaged {
		if strings.ToLower(ControllerManaged[i].Key) == lower {
			return &ControllerManaged[i], true
		}
	}
	return nil, false
}

func IsValidResourceKind(rule *AnnotationRule, kind string) bool {
	for _, k := range rule.ResourceKinds {
		if k == kind {
			return true
		}
	}
	return false
}

func ValidateDuration(value string) bool {
	_, err := time.ParseDuration(value)
	return err == nil
}
