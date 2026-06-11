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

## Controller-Managed Annotations (set by OLM, not users)

| Annotation | Resource |
|---|---|
| `olm.operatorGroup` | ClusterServiceVersion |
| `olm.operatorNamespace` | ClusterServiceVersion |
| `olm.targetNamespaces` | OperatorGroup |
| `olm.providedAPIs` | OperatorGroup |

## Console Annotations

| Annotation | Resource | Format |
|---|---|---|
| `console.openshift.io/disable-operand-delete` | ClusterServiceVersion | string |
| `features.operators.openshift.io/*` | ClusterServiceVersion | string |
| `operators.openshift.io/valid-subscription` | ClusterServiceVersion | JSON |
| `operatorframework.io/properties` | ClusterServiceVersion | JSON |

Use `olm-annotation-lint --list-rules` to see the complete list with format details.
