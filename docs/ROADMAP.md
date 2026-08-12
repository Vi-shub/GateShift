# GateShift Detailed Roadmap

**North star:** migrate large Ingress fleets to Gateway API with explicit annotation reporting, a typed middle-layer IR, and safe cutover. Do not invent unsafe proxy config. Do not claim 100% automatic conversion.

Related: [ARCHITECTURE.md](ARCHITECTURE.md) · [MIDDLE_LAYER.md](MIDDLE_LAYER.md) · [COMPARE.md](COMPARE.md) · [TESTING.md](TESTING.md) · [SCOREBOARD.md](SCOREBOARD.md) · [scoreboard.md](scoreboard.md)

---

## 1. Where we are (baseline)

GateShift is a **shipping CLI** with a real conversion engine. The operator/Helm path is still a scaffold.

| Area | Status | Notes |
|------|--------|-------|
| L1/L2/L3 NGINX + cert-manager adapters | Done | Tracked catalog 100% |
| CLI: `audit`, `convert`, `dual-run`, `diff`, `validate`, `migrate`, `coverage`, `scoreboard` | Done | Most are file-based; `audit` can use `--namespace` |
| IR `gateshift.ir/v1` + ordered pipeline + FinalizeIR | Done | Findings IDs/fixes, `RequiredFeatures` |
| Ingress-NGINX quirks + preserve/emit flags | Done | `pkg/nginxquirks` |
| Canary merge, snippet patterns | Done | |
| Multi-provider validate + corpus scoreboard (~170) | Done | CI hard-gates unreported = 0 |
| Dual-run CLI (`ApplyDualRunMode`) | Done | Unit tests exist; **no KinD e2e yet** |
| Releases / install script / convert KinD smoke | Done | |
| Operator + Helm | Scaffold | Converts + status + optional PR; does **not** apply dual-run or shadow resources |

### Cutover path (supported today via CLI)

1. `gateshift audit`
2. Fix or accept L3
3. `gateshift convert` + `validate`
4. `gateshift dual-run` (staging Gateway + `*-shadow` HTTPRoute; Ingress untouched)
5. Compare shadow vs live
6. `gateshift migrate` / GitOps PR
7. Flip DNS / listeners; delete Ingress last

---

## 2. Gap summary (what is missing)

### Product gaps

| Gap | Why it matters |
|-----|----------------|
| Dual-run KinD demo / CI | Cutover story proven (Sprint A); keep green on PRs |
| Fleet batch (`--all-namespaces`, `--selector`) | Real teams have dozens of Ingresses |
| `--http-only` / TLS secret awareness | Lab clusters break on missing certs |
| Operator DualRun mode + conditions | In-cluster adoption blocked |
| Helm NOTES + values docs | **Done (Sprint A)** |
| Traefik / ALB / GCE adapters | Locked to NGINX-heavy fleets |
| ReferenceGrant / cross-namespace | Common in multi-ns platforms |
| Untested packages | CLI, controller, gitops, audit, diff, cluster (Helm lint now in CI) |

### Test coverage today

**Have tests:** `pkg/convert` (strong), `pkg/adapters/nginx`, `pkg/nginxquirks`, `pkg/patterns`, `pkg/conformance`, `pkg/loader`, `pkg/scoreboard`, `pkg/ir` (partial).

**Missing / thin:** `internal/cli`, `internal/controller`, `pkg/cluster`, `pkg/gitops`, `pkg/audit`, `pkg/diff`, dual-run KinD e2e, operator e2e, Helm lint in CI.

---

## 3. Code to add (by phase)

Paths are suggestions; keep IR contract (`gateshift.ir/v1`) stable unless bumping to v2.

### Phase 0. Harden what we have *(do next)*

Make the current CLI trustworthy in labs and early production.

| # | Work item | Code / files to add or change | Done when |
|---|-----------|-------------------------------|-----------|
| 0.1 | Dual-run KinD demo | `scripts/test-dual-run.sh` (and/or extend `scripts/demo-podinfo.sh`); wire step in `.github/workflows/smoke.yml` | Apply shadow YAML, curl staging Gateway, assert Ingress still present | **Done (Sprint A)** |
| 0.2 | `--http-only` + TLS awareness | `pkg/convert/convert.go` (`Options`), emit listeners/certs; flags in `internal/cli/convert.go`, `dual_run.go` | Lab convert works without cert-manager secrets | Open |
| 0.3 | Helm NOTES + values docs | `charts/gateshift-operator/templates/NOTES.txt`, `charts/gateshift-operator/README.md` | Fresh install prints next steps | **Done (Sprint A)** |
| 0.4 | README migration story | `README.md` short section: audit → dual-run → cutover | Linked from Quick start | **Done (Sprint A)** |
| 0.5 | Scoreboard doc hygiene | Keep guide in `docs/SCOREBOARD.md`; CI writes `docs/scoreboard.md` only | Guide never overwritten by CI | Open (verify) |
| 0.6 | Dual-run docs in TESTING | `docs/TESTING.md` dual-run e2e section | Contributors know how to run it | **Done (Sprint A)** |
| 0.7 | Signed operator images *(later in P0)* | Cosign / provenance in `.github/workflows/release.yml` + image build | Attested operator image on release | Open |

**Phase 0 exit:** new user installs CLI, runs dual-run on checkout/podinfo demo in KinD, and Helm NOTES explain operator stub without tribal knowledge.

### Phase 1. Fleet adoption *(highest product leverage)*

Help teams migrate many Ingresses safely.

| # | Work item | Code / files | Done when |
|---|-----------|--------------|-----------|
| 1.1 | Fleet list API | `pkg/cluster/client.go`: `AllNamespaces`, `LabelSelector` | Fake-client unit tests pass |
| 1.2 | Batch `audit` | `internal/cli/audit.go`: `--all-namespaces`, `--selector`; `pkg/audit/fleet.go` report | Namespace score in one command |
| 1.3 | Live / batch `dual-run` | `internal/cli/dual_run.go`: `--namespace` / `--selector` parity with audit | Shadow YAML for a label set |
| 1.4 | Optional RequestMirror helper | `pkg/convert/dualrun.go` + CLI `--request-mirror` | Shadow route can mirror when controller supports it |
| 1.5 | ReferenceGrant emission | `pkg/ir/types.go` + `pkg/convert` emitter | Cross-ns parent/backend documented + tested |
| 1.6 | Traefik adapters (L1 first) | `pkg/adapters/traefik/` + registry wire-up + corpus fixtures | Common Traefik keys → findings/filters |
| 1.7 | Promote `[??]` corpus keys | `pkg/adapters/nginx/catalog.go` + adapters; update `docs/CORPUS_GAPS.md` | High-freq unknowns become cataloged |
| 1.8 | Live GatewayClass discovery | `pkg/conformance` + optional `pkg/cluster` probe | Validate softens static-only matrix where possible |
| 1.9 | Conformance pack export | `pkg/conformance` HTML/Markdown writer + CLI flag | Change-review board artifact |
| 1.10 | ALB / GCE adapters | `pkg/adapters/alb/`, `pkg/adapters/gce/` | Structure + high-value annotations |

**Phase 1 exit:** platform team can score a namespace, dual-run a subset, and produce a reviewable report without hand-writing Gateway YAML.

### Phase 2. Operator maturity

Make in-cluster migration first-class.

| # | Work item | Code / files | Done when |
|---|-----------|--------------|-----------|
| 2.1 | API: mode + conditions | `api/v1alpha1/migrationrequest_types.go`, CRD YAMLs | `spec.mode`: Convert \| DualRun \| Cutover; conditions Ready / Blocked / DualRun / PROpened |
| 2.2 | Reconciler DualRun + apply | `internal/controller/migrationrequest_controller.go` | Calls `ApplyDualRunMode`; creates/updates shadow resources; **never deletes Ingress**; Events recorded |
| 2.3 | Prometheus metrics | controller + `cmd/gateshift-operator/main.go` | conversions, L3 counts, readiness histogram, PR failures |
| 2.4 | Admission / readiness gate | webhook or validating policy under `internal/` + chart templates | Block apply when readiness &lt; threshold or hard L3 |
| 2.5 | Namespace-scoped RBAC | `config/rbac/`, chart `watchNamespace` | Multi-tenant example works |
| 2.6 | GitLab + Azure DevOps GitOps | `pkg/gitops/gitlab.go`, `azuredevops.go` + migrate flags | Non-GitHub dry-run/PR path |

**Phase 2 exit:** `MigrationRequest` drives dual-run → validate → PR with observable status in a real cluster.

### Phase 3. Ecosystem and distribution

| # | Work item | Deliverable |
|---|-----------|-------------|
| 3.1 | Homebrew + Scoop (+ optional winget) | Install formulas |
| 3.2 | Community announcement pack | Blog + scoreboard + COMPARE one-pager |
| 3.3 | Adapter contribution guide | `CONTRIBUTING.md` section + template adapter |
| 3.4 | IR compatibility policy | What may change in `gateshift.ir/v1` vs `v2` |
| 3.5 | Optional corpus search | Nearest-known migration (keyword first) |

### Phase 4. Advanced fidelity *(as demand appears)*

| # | Work item | Notes |
|---|-----------|-------|
| 4.1 | Deeper NGINX quirk parity | Conflicting annotations, URL normalization edges |
| 4.2 | Stronger Cilium / Istio / Kong Policies | Closer Envoy parity |
| 4.3 | Shadow vs live structured diff | Beyond YAML diff |
| 4.4 | Progressive delivery hooks | Argo Rollouts / Flagger (optional) |
| 4.5 | Multi-cluster inventory export | Platform backplane |

---

## 4. Tests to add (detailed)

### P0 (add with Phase 0)

| Test | Path / harness | Assert |
|------|----------------|--------|
| Dual-run KinD e2e | `scripts/test-dual-run.sh` + smoke workflow | Shadow Gateway/HTTPRoute applied; Ingress unchanged; HTTP through staging works |
| `--http-only` unit | `pkg/convert/convert_test.go` | No HTTPS listener / no cert require when flag set |
| Dual-run annotation stability | extend `pkg/convert/dualrun_test.go` | `gateshift.io/mode=dual-run`, `*-shadow` names, no Ingress docs in emit |
| Helm lint | CI step `helm lint charts/gateshift-operator` | Chart templates render |

### P1 (add with Phase 1)

| Test | Path | Assert |
|------|------|--------|
| Fleet list + selector | `pkg/cluster/client_test.go` (fake clientset) | Label/ns filters |
| Batch audit report | `pkg/audit/fleet_test.go` | Aggregated readiness / L3 counts |
| Multi-Ingress dual-run | `pkg/convert/dualrun_test.go` | Combined parentRefs / staging names |
| Traefik L1 adapters | `pkg/adapters/traefik/*_test.go` | Key → filter/finding |
| ReferenceGrant emit | `pkg/convert` test | Cross-ns backend emits grant |

### P2 (add with Phase 2)

| Test | Path | Assert |
|------|------|--------|
| Reconciler DualRun | `internal/controller/migrationrequest_controller_test.go` (envtest/fake) | Shadow created; Ingress not deleted; conditions set |
| Condition transitions | same | Blocked on L3; Ready after convert; PROpened after gitops |
| GitOps dry-run layout | `pkg/gitops/pr_test.go` | Files written under dry-run dir |

### Cross-cutting (cheap wins anytime)

| Test | Path |
|------|------|
| Readiness score edges | `pkg/ir/readiness_test.go` |
| Diff formatting | `pkg/diff/diff_test.go` |
| Audit matrix columns (ID/FIX) | `pkg/audit/report_test.go` |
| CLI smoke (optional) | `internal/cli/*_test.go` via cobra execute + temp files |

### CI matrix (target state)

| Workflow | Today | Target |
|----------|-------|--------|
| `ci.yml` | gofmt, `go test`, build, scoreboard, goreleaser check | + `helm lint` |
| `smoke.yml` | convert KinD | + dual-run job/step |
| `release.yml` | CLI binaries | + operator image build/sign (Phase 0.7 / 2) |

---

## 5. Priority order (what to build next)

| Order | Item | Phase | Why |
|------:|------|-------|-----|
| 1 | Dual-run KinD demo + CI | 0 | **Done (Sprint A)** |
| 2 | README migration story + TESTING dual-run | 0 | Users find the path |
| 3 | `--http-only` | 0 | Unblocks lab clusters |
| 4 | Helm NOTES + values README | 0 | Operator installable without Slack |
| 5 | Fleet `--selector` / `--all-namespaces` on audit + dual-run | 1 | Real multi-Ingress users |
| 6 | Traefik L1 adapters + `[??]` promotion | 1 | Broaden beyond NGINX |
| 7 | Operator DualRun mode + conditions + tests | 2 | In-cluster adoption |
| 8 | ReferenceGrant | 1–2 | Multi-ns platforms |
| 9 | Signed images + package managers | 0.7 / 3 | Production trust + distribution |

---

## 6. Suggested sprint breakdown

### Sprint A (Phase 0 core) — dual-run proof

- [x] `scripts/test-dual-run.sh`
- [x] Wire dual-run into `smoke.yml`
- [x] README migration story
- [x] `docs/TESTING.md` dual-run section
- [x] Helm `NOTES.txt` + chart README
- [x] Helm lint in CI (`ci.yml` + `make helm-lint`)
- [ ] `--http-only` convert flag (moved to next / Sprint D if deferred)

**Phase 0 remaining:** `--http-only`, signed operator images, scoreboard refresh after adapter changes.

### Sprint B (Fleet MVP) — Phase 1 start

- [ ] `pkg/cluster` selector / all-namespaces
- [ ] `audit --selector` / `--all-namespaces` + fleet report
- [ ] `dual-run --namespace` / `--selector`
- [ ] Cluster + audit fleet unit tests

### Sprint C (Operator DualRun) — Phase 2 start

- [ ] API `spec.mode` + conditions
- [ ] Reconciler apply shadow resources
- [ ] Controller unit/envtest
- [ ] Example `MigrationRequest` for dual-run

### Sprint D (Breadth)

- [ ] Traefik L1 adapters + fixtures
- [ ] `--http-only` if not done earlier
- [ ] Promote top `[??]` keys from `CORPUS_GAPS.md`
- [ ] Refresh `docs/scoreboard.md`

---

## 7. Done log (compact)

- Plug-in L1/L2/L3 adapter engine; NGINX catalog closed
- Snippet pattern library; canary merge
- Readiness score; coverage catalog; multi-provider validate
- IR contract + ordered pipeline + goldens/fuzz (`gateshift.ir/v1`)
- Behavioral quirks (`nginxquirks` + CLI flags)
- Corpus + `scoreboard` CI; COMPARE / fidelity docs
- `gateshift dual-run` staging Gateway + shadow HTTPRoute (unit-tested)
- GoReleaser, install script (`vi-shub/gateshift`), convert KinD smoke
- Operator / CRD / Helm scaffold; GitOps dry-run + GitHub PR helper

---

## 8. Non-goals

- Full nginx / Lua → Envoy compiler
- Guaranteeing 100% automatic conversion of every Ingress
- Replacing controller Policy CRDs with invented Core API fields
- Competing only on hosts/paths/TLS (that niche is already served)

GateShift wins on **annotation fidelity + honest findings + dual-run**.

---

## 9. Success metrics

| Metric | Target | Status |
|--------|--------|--------|
| Tracked NGINX catalog | 100% | Met (maintain) |
| Unreported migration annotations | 0 | Met (CI gate) |
| IR schema | `gateshift.ir/v1` stable | Met |
| Convert KinD smoke | Green on PR | Met |
| Dual-run KinD demo | Green on PR | **Met (Sprint A script + smoke job)** |
| Fleet batch audit | One command per namespace/selector | **Not met** (Phase 1) |
| Operator DualRun | Shadow applied + conditions | **Not met** (Phase 2) |
| Helm NOTES | Present | **Met (Sprint A)** |
| Homebrew / Scoop | Published | **Not met** (Phase 3) |
| Fleet readiness ≥ 60 | ≥ 80% of scanned Ingresses (stretch) | Customer-dependent |

---

## 10. How to use this roadmap

1. Pick the highest open item in **§5 Priority order**.
2. Implement the **Code to add** row in **§3**.
3. Land the matching **Tests to add** in **§4** in the same PR when possible.
4. Check the phase **exit criteria** before starting the next phase.
5. After adapter or corpus changes, refresh scoreboard: `make scoreboard`.
