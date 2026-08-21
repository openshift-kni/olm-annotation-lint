# Adopters

Repositories with OLM annotation usage that could run olm-annotation-lint as a GitHub Action or CI step.

## Adoption snippet

```yaml
- uses: openshift-kni/olm-annotation-lint@v1
  with:
    path: "."
    exclude: "vendor,testdata"
    format: "github"
```

## openshift-kni

| Repo | YAML files | Key annotations | Notes |
|------|-----------|-----------------|-------|
| [cnf-features-deploy](https://github.com/openshift-kni/cnf-features-deploy) | 76 | `operatorframework.io/bundle-unpack-min-retry-interval`, `olm.skipRange`, `olm.providedAPIs` | [PR #4197](https://github.com/openshift-kni/cnf-features-deploy/pull/4197) |
| [telco-reference](https://github.com/openshift-kni/telco-reference) | 46 | `operatorframework.io/bundle-unpack-min-retry-interval`, `olm.providedAPIs` | [PR #760](https://github.com/openshift-kni/telco-reference/pull/760) |
| [lifecycle-agent](https://github.com/openshift-kni/lifecycle-agent) | 18 | `olm.skipRange`, `operatorframework.io/suggested-namespace` | [PR #7163](https://github.com/openshift-kni/lifecycle-agent/pull/7163) |
| [cluster-group-upgrades-operator](https://github.com/openshift-kni/cluster-group-upgrades-operator) | 14 | `olm.skipRange`, `operatorframework.io/suggested-namespace` | |
| [oran-o2ims](https://github.com/openshift-kni/oran-o2ims) | 14 | `olm.skipRange`, `operatorframework.io/suggested-namespace` | |
| [example-cnf](https://github.com/openshift-kni/example-cnf) | 10 | `olm.skipRange` | |
| [performance-addon-operators](https://github.com/openshift-kni/performance-addon-operators) | 9 | `olm.skipRange`, `olm.targetNamespaces` | |
| [numaresources-operator](https://github.com/openshift-kni/numaresources-operator) | 7 | `olm.skipRange`, `operatorframework.io/cluster-monitoring` | |
| [oran-hwmgr-plugin](https://github.com/openshift-kni/oran-hwmgr-plugin) | 5 | `olm.skipRange` | |
| [eapol-operator](https://github.com/openshift-kni/eapol-operator) | 4 | builder/layout annotations | |
| [kubernetes-power-manager](https://github.com/openshift-kni/kubernetes-power-manager) | 4 | builder/layout annotations | |
| [openperouter](https://github.com/openshift-kni/openperouter) | 3 | bundle annotations | |
| [eco-ci-cd](https://github.com/openshift-kni/eco-ci-cd) | 1 | CatalogSource/OperatorGroup | |

## openshift

| Repo | YAML files | Key annotations |
|------|-----------|-----------------|
| [custom-metrics-autoscaler-operator](https://github.com/openshift/custom-metrics-autoscaler-operator) | 51 | `olm.skipRange`, `olm.targetNamespaces`, `operatorframework.io/suggested-namespace` |
| [external-dns-operator](https://github.com/openshift/external-dns-operator) | 36 | `olm.skipRange`, `operatorframework.io/suggested-namespace` |
| [lightspeed-operator](https://github.com/openshift/lightspeed-operator) | 25 | `olm.bundle`, `operatorframework.io/suggested-namespace` |
| [aws-load-balancer-operator](https://github.com/openshift/aws-load-balancer-operator) | 22 | `olm.skipRange` |
| [sandboxed-containers-operator](https://github.com/openshift/sandboxed-containers-operator) | 12 | `olm.bundle`, `olm.channel`, `olm.deprecations` |
| [loki](https://github.com/openshift/loki) | 12 | `olm.skipRange`, `olm.maxOpenShiftVersion` |
| [lvm-operator](https://github.com/openshift/lvm-operator) | 11 | `olm.skipRange`, `operatorframework.io/internal-objects` |
| [metallb-operator](https://github.com/openshift/metallb-operator) | 11 | `olm.skipRange`, `olm.targetNamespaces` |
| [grafana-tempo-operator](https://github.com/openshift/grafana-tempo-operator) | 9 | OLM annotations |
| [sriov-network-operator](https://github.com/openshift/sriov-network-operator) | 8 | `olm.skipRange`, `operatorframework.io/internal-objects` |
| [nbde-tang-server](https://github.com/openshift/nbde-tang-server) | 8 | OLM annotations |
| [ptp-operator](https://github.com/openshift/ptp-operator) | 7 | `olm.skipRange`, `olm.targetNamespaces` |
| [pf-status-relay-operator](https://github.com/openshift/pf-status-relay-operator) | 7 | `olm.skipRange`, `olm.targetNamespaces` |
| [dpu-operator](https://github.com/openshift/dpu-operator) | 7 | OLM annotations |
| [route-monitor-operator](https://github.com/openshift/route-monitor-operator) | 7 | OLM annotations |
| [vertical-pod-autoscaler-operator](https://github.com/openshift/vertical-pod-autoscaler-operator) | 7 | OLM annotations |
| [ingress-node-firewall](https://github.com/openshift/ingress-node-firewall) | 6 | `olm.skipRange` |
| [cert-manager-operator](https://github.com/openshift/cert-manager-operator) | 6 | `olm.skipRange`, `olm.targetNamespaces` |
| [cluster-nfd-operator](https://github.com/openshift/cluster-nfd-operator) | 6 | `olm.skipRange` |
| [multiarch-tuning-operator](https://github.com/openshift/multiarch-tuning-operator) | 6 | OLM annotations |
| [external-secrets-operator](https://github.com/openshift/external-secrets-operator) | 6 | `olm.skipRange`, `olm.targetNamespaces` |
| [open-telemetry-opentelemetry-operator](https://github.com/openshift/open-telemetry-opentelemetry-operator) | 6 | OLM annotations |
| [bpfman-catalog](https://github.com/openshift/bpfman-catalog) | 6 | OLM annotations |
| [cluster-logging-operator](https://github.com/openshift/cluster-logging-operator) | 5 | `olm.skipRange`, `olm.targetNamespaces` |
| [file-integrity-operator](https://github.com/openshift/file-integrity-operator) | 5 | `olm.skipRange`, `olm.targetNamespaces` |
| [oadp-operator](https://github.com/openshift/oadp-operator) | 5 | `olm.skipRange`, `olm.targetNamespaces` |
| [bpfman-operator](https://github.com/openshift/bpfman-operator) | 5 | OLM annotations |
| [trustee-operator](https://github.com/openshift/trustee-operator) | 5 | OLM annotations |
| [windows-machine-config-operator](https://github.com/openshift/windows-machine-config-operator) | 5 | `olm.skipRange`, `olm.targetNamespaces` |
| [zero-trust-workload-identity-manager](https://github.com/openshift/zero-trust-workload-identity-manager) | 5 | OLM annotations |

## redhat-openshift-ecosystem

| Repo | YAML files | Type |
|------|-----------|------|
| [operator-certification-operator](https://github.com/redhat-openshift-ecosystem/operator-certification-operator) | 4 | Operator |
| [openshift-preflight](https://github.com/redhat-openshift-ecosystem/openshift-preflight) | 2 | Tooling |
| [operator-pipelines](https://github.com/redhat-openshift-ecosystem/operator-pipelines) | 2 | CI/CD |

## Priority

1. **openshift-kni** — we own these and can open PRs directly
2. **openshift operator repos** — especially sriov, ptp, metallb
3. **redhat-openshift-ecosystem** — useful for operator certification

This is a living document. Update it when adoption PRs are opened or merged.
