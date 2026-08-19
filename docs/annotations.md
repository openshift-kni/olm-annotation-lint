# Annotations Reference

## What It Catches

| Rule | Severity | Example |
|---|---|---|
| Unknown OLM annotation | ERROR | `olm.operatorframework.io/bundle-install-timeout` (doesn't exist) |
| Wrong resource type | ERROR | `bundle-unpack-timeout` on a Subscription instead of OperatorGroup |
| Invalid value format | ERROR | `bundle-unpack-timeout: "not-a-duration"` |
| Wrong prefix | ERROR | `olm.operatorframework.io/` instead of `operatorframework.io/` |
| Case mismatch | ERROR | `OLM.providedAPIs` instead of `olm.providedAPIs` |
| Controller-managed annotation | WARNING | User setting `olm.operatorGroup` (OLM manages this) |

## User-Settable Annotations

| Annotation | Resource | Format |
|---|---|---|
| `operatorframework.io/bundle-unpack-timeout` | OperatorGroup | duration (e.g. `10m`) |
| `operatorframework.io/bundle-unpack-min-retry-interval` | OperatorGroup | duration |
| `operatorframework.io/priorityclass` | CatalogSource | string |
| `olm.catalogImageTemplate` | CatalogSource | template |
| `olm.skipRange` | ClusterServiceVersion | semver range |
| `olm.operatorframework.io/exclude-global-namespace-resolution` | Subscription | string |
| `operatorframework.io/suggested-namespace` | ClusterServiceVersion | string |
| `operatorframework.io/suggested-namespace-template` | ClusterServiceVersion | JSON |
| `operatorframework.io/cluster-monitoring` | ClusterServiceVersion | string |

## Controller-Managed Annotations (set by OLM, not users)

| Annotation | Resource |
|---|---|
| `olm.operatorGroup` | ClusterServiceVersion |
| `olm.operatorNamespace` | ClusterServiceVersion |
| `olm.targetNamespaces` | OperatorGroup |
| `olm.providedAPIs` | OperatorGroup |

## Bundle Annotations (OLM v1)

These annotations appear in `metadata/annotations.yaml` files inside operator bundles. The linter auto-detects bundle files by their content — no special flags needed.

| Annotation | Format |
|---|---|
| `operators.operatorframework.io.bundle.mediatype.v1` | bundle mediatype (`registry+v1`, `plain+v0`, `helm+v0`) |
| `operators.operatorframework.io.bundle.manifests.v1` | string (directory path) |
| `operators.operatorframework.io.bundle.metadata.v1` | string (directory path) |
| `operators.operatorframework.io.bundle.package.v1` | string |
| `operators.operatorframework.io.bundle.channels.v1` | comma-separated list |
| `operators.operatorframework.io.bundle.channel.default.v1` | string |
| `operators.operatorframework.io.metrics.builder` | string |
| `operators.operatorframework.io.metrics.mediatype.v1` | string |
| `operators.operatorframework.io.metrics.project_layout` | string |
| `operators.operatorframework.io.test.config.v1` | string |
| `operators.operatorframework.io.test.mediatype.v1` | string |

## OLM v1 Controller-Managed Annotations

These annotations are set by the OLM v1 operator-controller on `ClusterObjectSet` resources and should not be set by users.

| Annotation | Resource |
|---|---|
| `olm.operatorframework.io/bundle-name` | ClusterObjectSet |
| `olm.operatorframework.io/bundle-version` | ClusterObjectSet |
| `olm.operatorframework.io/bundle-release` | ClusterObjectSet |
| `olm.operatorframework.io/bundle-reference` | ClusterObjectSet |
| `olm.operatorframework.io/service-account-name` | ClusterObjectSet |
| `olm.operatorframework.io/service-account-namespace` | ClusterObjectSet |

## Console and OpenShift Annotations

These annotations commonly appear on ClusterServiceVersions. Annotations outside the `olm.`, `operatorframework.io/`, and `operators.operatorframework.io.` prefixes are not scanned. `operatorframework.io/properties` is scanned but is not in the known-rules list — use `--allow` if you set it intentionally.

| Annotation | Resource | Format | Notes |
|---|---|---|---|
| `console.openshift.io/disable-operand-delete` | ClusterServiceVersion | string | Not scanned (outside OLM prefixes) |
| `features.operators.openshift.io/*` | ClusterServiceVersion | string | Not scanned |
| `operators.openshift.io/valid-subscription` | ClusterServiceVersion | JSON | Not scanned |
| `operatorframework.io/properties` | ClusterServiceVersion | JSON | Scanned; flagged as unknown unless allowed |

Use `olm-annotation-lint --list-rules` to see the complete list with format details.
