# gateshift-operator Helm chart

Scaffold chart for the GateShift Kubernetes operator (`MigrationRequest` reconciler).

**Status:** usable for dry-run / PR workflows. Dual-run apply mode is on the roadmap (Phase 2). Prefer the CLI `gateshift dual-run` for shadow cutover today.

## Install

```bash
# From repo root
helm upgrade --install gateshift-operator charts/gateshift-operator \
  --namespace gateshift-system --create-namespace

# CRD (if not already applied via chart/crds)
kubectl apply -f config/crd/migrationrequest.yaml
```

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `replicaCount` | `1` | Operator replicas |
| `image.repository` | `ghcr.io/gateshift/gateshift-operator` | Container image |
| `image.tag` | chart `appVersion` | Image tag |
| `image.pullPolicy` | `IfNotPresent` | Pull policy |
| `serviceAccount.create` | `true` | Create ServiceAccount |
| `serviceAccount.name` | `gateshift-operator` | SA name |
| `resources.requests/limits` | see `values.yaml` | CPU/memory |
| `metrics.enabled` | `true` | Expose metrics port |
| `metrics.port` | `8080` | Metrics bind port |
| `githubToken.existingSecret` | `""` | Optional Secret name for `GITHUB_TOKEN` |
| `githubToken.key` | `token` | Key inside that Secret |

### GitHub token for GitOps PRs

```bash
kubectl -n gateshift-system create secret generic gateshift-github \
  --from-literal=token="$GITHUB_TOKEN"

helm upgrade --install gateshift-operator charts/gateshift-operator \
  --namespace gateshift-system \
  --set githubToken.existingSecret=gateshift-github
```

## Example MigrationRequest

```bash
kubectl apply -f examples/migrationrequest.yaml
kubectl get migrationrequest -A
```

## Lint

```bash
helm lint charts/gateshift-operator
helm template gateshift-operator charts/gateshift-operator >/dev/null
```

## Related

- CLI dual-run: `gateshift dual-run -h`
- Roadmap: [docs/ROADMAP.md](../../docs/ROADMAP.md)
- Testing: [docs/TESTING.md](../../docs/TESTING.md)
