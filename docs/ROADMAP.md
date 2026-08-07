# GateShift Roadmap

**North star:** migrate large Ingress fleets to Gateway API with explicit annotation reporting, a typed middle-layer IR, and safe cutover  -  without inventing unsafe proxy config or claiming 100% automatic conversion.

Related: [ARCHITECTURE.md](ARCHITECTURE.md) · [MIDDLE_LAYER.md](MIDDLE_LAYER.md) · [COMPARE.md](COMPARE.md) · [TESTING.md](TESTING.md) · [scoreboard.md](scoreboard.md)

---

## Where we are (successful baseline)

GateShift is a **shipping CLI** (optional operator) with a full conversion engine  -  not a thin wrapper around ingress2gateway.

| Area | Status |
|------|--------|
| L1/L2/L3 NGINX + cert-manager adapters | Done  -  tracked catalog **100%** |
| CLI: `audit`, `convert`, `dual-run`, `diff`, `validate`, `migrate`, `coverage`, `scoreboard` | Done |
| Typed IR `gateshift.ir/v1` (findings IDs/fixes, `RequiredFeatures`) | Done |
| Ordered pipeline (host-index → adapters → quirks → canary → finalize) | Done |
| Ingress-NGINX behavioral quirks + preserve/emit flags | Done |
| Canary merge → weighted HTTPRoute | Done |
| Snippet pattern library (promote safe idioms; residual L3) | Done |
| Multi-provider Policies (Envoy focus; Cilium/Istio/Kong/standard) | Done |
| Public corpus (~170 Ingresses) + CI scoreboard | Done |
| Dual-run / shadow cutover (`gateshift dual-run`) | Done |
| Releases (GoReleaser), install script, KinD smoke CI | Done |
| Operator CRD + reconciler scaffold + Helm stub | Scaffold only |

### Recommended cutover path (today)

1. `gateshift audit`: inventory L1/L2/L3 debt  
2. Rewrite or accept L3 snippets  
3. `gateshift convert` + `validate`  
4. `gateshift dual-run`: keep Ingress; apply staging Gateway + `*-shadow` HTTPRoute  
5. Compare shadow vs live Ingress  
6. `gateshift migrate` / GitOps PR  
7. Flip DNS / listeners; **delete Ingress last**

---

## Phase plan

### Phase 0  -  Harden what we have *(near-term)*

Make the current product trustworthy in labs and early production.

- [ ] KinD / demo script path for **dual-run** (apply shadow YAML, curl staging Gateway, leave Ingress)
- [ ] `--http-only` / TLS-secret awareness in convert for clusters without cert-manager secrets
- [ ] Helm chart: values reference + install `NOTES.txt`: [ ] Signed / provenance-attested operator images (cosign or equivalent)
- [ ] Refresh committed `docs/scoreboard.md` after major adapter changes
- [ ] README “migration story” section: audit → dual-run → cutover (short)

**Exit criteria:** a new user can install CLI, run dual-run on the podinfo/checkout demo, and follow NOTES for the operator stub without tribal knowledge.

### Phase 1  -  Fleet adoption *(highest product leverage)*

Help teams migrate many Ingresses safely.

- [ ] Dual-run: optional RequestMirror / traffic-split helpers (where controller supports it)
- [ ] Namespace / label selectors: `audit` / `dual-run` over a live fleet (batch report)
- [ ] Grow corpus: charts + more upstream examples; promote `[??]` keys → catalog + adapters
- [ ] Traefik annotation adapters (L1 first, then common L2)
- [ ] AWS ALB / GCE Ingress adapters (structure + high-value annotations)
- [ ] Live `GatewayClass` / controller feature discovery (reduce static validate matrix drift)
- [ ] ReferenceGrant + cross-namespace `parentRefs` / backendRefs
- [ ] HTML/Markdown conformance pack for change-review boards

**Exit criteria:** a platform team can score a namespace, dual-run a subset, and produce a reviewable report without hand-writing Gateway YAML.

### Phase 2  -  Operator maturity

Make in-cluster migration first-class (not only CLI).

- [ ] `MigrationRequest` status conditions + events (Ready / Blocked / DualRun / PROpened)
- [ ] Operator supports dual-run mode (emit shadow resources; never delete Ingress)
- [ ] Prometheus metrics: conversions, L3 counts, readiness histogram, PR failures
- [ ] Admission / policy gate: block apply when readiness &lt; threshold or L3 present
- [ ] GitLab + Azure DevOps GitOps backends (beyond GitHub PR)
- [ ] Multi-tenant RBAC examples + restricted namespace watch

**Exit criteria:** `MigrationRequest` can drive dual-run → validate → PR with observable status in a real cluster.

### Phase 3  -  Ecosystem & distribution

Grow awareness and install surface.

- [ ] Homebrew + Scoop (and optional `winget`) formulas
- [ ] Gateway API / CNCF community announcement pack (blog outline, scoreboard, COMPARE)
- [ ] Optional corpus “nearest known migration” search (embedding or keyword)
- [ ] Documented contribution path for new provider adapters
- [ ] Stable IR schema compatibility policy (what can change in `gateshift.ir/v1` vs `v2`)

**Exit criteria:** installable from common package managers; public narrative is “annotation-faithful migrator with dual-run,” not “another YAML converter.”

### Phase 4  -  Advanced fidelity *(as demand appears)*

Deeper correctness after the above is solid.

- [ ] More Ingress-NGINX quirk parity (URL normalization edge cases, merge of conflicting annotations)
- [ ] Stronger provider Policy emission for Cilium / Istio / Kong (parity with Envoy where possible)
- [ ] Diff / compare: live Ingress traffic shape vs shadow HTTPRoute (structured report)
- [ ] Progressive delivery hooks (Argo Rollouts / Flagger parentRef patterns)  -  optional
- [ ] Multi-doc / multi-cluster inventory export for platform backplanes

---

## Priority cheat-sheet (what to build next)

| Order | Item | Why |
|------:|------|-----|
| 1 | Dual-run KinD demo + docs polish | Proves the cutover story end-to-end |
| 2 | Fleet batch audit / dual-run | Real multi-Ingress users |
| 3 | Traefik (then ALB/GCE) adapters | Broaden beyond NGINX |
| 4 | Operator status + dual-run mode | In-cluster adoption |
| 5 | Helm NOTES + signed images | Production trust |
| 6 | Package managers + community pack | Distribution |

---

## Done log (compact)

- Plug-in L1/L2/L3 adapter engine; NGINX catalog closed
- Snippet pattern library; canary merge
- Readiness score; coverage catalog; multi-provider validate
- IR contract + ordered pipeline + goldens/fuzz (`gateshift.ir/v1`)
- Behavioral quirks (`nginxquirks` + CLI flags)
- Corpus + `scoreboard` CI; COMPARE / ANNOTATION_FIDELITY docs
- `gateshift dual-run` staging Gateway + shadow HTTPRoute
- GoReleaser, install script, KinD smoke workflow
- Operator / CRD / Helm scaffold; GitOps dry-run + GitHub PR helper

---

## Non-goals

- Full nginx / Lua → Envoy compiler
- Guaranteeing 100% automatic conversion of every Ingress
- Replacing controller Policy CRDs with invented Core API fields
- Competing on “hosts/paths/TLS only”  -  that niche is already served; GateShift wins on **annotation fidelity + honest findings + dual-run**

---

## Success metrics

| Metric | Current target | Notes |
|--------|----------------|-------|
| Tracked NGINX catalog | **100%** implemented | Maintain; grow via `[??]` → catalog |
| Unreported migration annotations | **0** | Scoreboard / CI hard gate |
| IR schema | `gateshift.ir/v1` stable for emitters | Bump only with intentional v2 |
| Fleet readiness ≥ 60 | ≥ 80% of scanned Ingresses (stretch) | Depends on customer fleets |
| Dual-run demo | Green on KinD | Phase 0 exit |
| KinD smoke (convert path) | Green on PR / release | Already wired |
| Operator dual-run | Status Ready + shadow applied | Phase 2 exit |
| External install | Homebrew or Scoop published | Phase 3 exit |
