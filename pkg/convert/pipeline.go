package convert

// Ordered conversion pipeline (middle layer). Emitters must not re-parse Ingress.
//
//	1. Host-index + quirks   nginxquirks.Analyze  → HostModes, quirk findings
//	2. Canary split          splitCanaries
//	3. Adapters              nginx.Translate per Ingress → filters/policies/findings
//	4. Route build           FromIngress path/TLS/filter assembly
//	5. Canary merge          applyCanary weighted/header backends
//	6. Quirk attach          append Analyze findings; optional preserve/emit flags
//	7. FinalizeIR            normalize findings, sort, SchemaVersion, RequiredFeatures
//
// Contract: MigrationBundle (gateshift.ir/v1) is the only input to emitters,
// conformance.ValidateBundle, audit, and GitOps. Unknown annotations become
// findings (never silently dropped).
