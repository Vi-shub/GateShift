package convert

import (
	"strings"
)

// Envoy Gateway policy emission targets the intersection of fields that have been
// stable since EG 1.2 (gateway.envoyproxy.io/v1alpha1). Newer-only fields
// (requestBuffer, bandwidthLimit, admissionControl, telemetry, …) are intentionally
// omitted so `kubectl apply` works across EG versions without strict-decode errors.

var egBackendTrafficPolicySpecAllow = map[string]bool{
	"targetRefs": true, "targetRef": true, "targetSelectors": true,
	"loadBalancer": true, "retry": true, "proxyProtocol": true, "tcpKeepalive": true,
	"healthCheck": true, "circuitBreaker": true, "timeout": true, "connection": true,
	"dns": true, "http2": true, "rateLimit": true, "faultInjection": true,
	"useClientProtocol": true, "responseOverride": true,
}

var egSecurityPolicySpecAllow = map[string]bool{
	"targetRefs": true, "targetRef": true, "targetSelectors": true,
	"cors": true, "basicAuth": true, "jwt": true, "oidc": true,
	"extAuth": true, "authorization": true,
}

// sanitizeEnvoyGatewayPolicySpec drops unknown top-level keys and returns false when
// nothing useful remains beyond targeting (skip emitting empty shells).
func sanitizeEnvoyGatewayPolicySpec(kind string, spec map[string]any) (map[string]any, bool) {
	var allow map[string]bool
	switch kind {
	case "BackendTrafficPolicy":
		allow = egBackendTrafficPolicySpecAllow
	case "SecurityPolicy":
		allow = egSecurityPolicySpecAllow
	default:
		// ClientTrafficPolicy and other EG kinds: do not emit from GateShift yet
		// (schemas differ by version and are easy to get wrong).
		return nil, false
	}

	out := make(map[string]any, len(spec))
	for k, v := range spec {
		if allow[k] {
			out[k] = v
		}
	}

	// Prefer targetRefs; drop deprecated targetRef if both somehow present.
	if _, ok := out["targetRefs"]; ok {
		delete(out, "targetRef")
	}

	// Nested safety for common adapters.
	if rl, ok := out["rateLimit"].(map[string]any); ok {
		out["rateLimit"] = sanitizeRateLimit(rl)
	}
	if lb, ok := out["loadBalancer"].(map[string]any); ok {
		out["loadBalancer"] = sanitizeLoadBalancer(lb)
	}
	if to, ok := out["timeout"].(map[string]any); ok {
		out["timeout"] = sanitizeTimeout(to)
	}
	if conn, ok := out["connection"].(map[string]any); ok {
		out["connection"] = sanitizeConnection(conn)
	}
	if ba, ok := out["basicAuth"].(map[string]any); ok {
		out["basicAuth"] = sanitizeBasicAuth(ba)
	}
	if ea, ok := out["extAuth"].(map[string]any); ok {
		clean, ok := sanitizeExtAuth(ea)
		if !ok {
			delete(out, "extAuth")
		} else {
			out["extAuth"] = clean
		}
	}

	useful := false
	for k := range out {
		if k != "targetRefs" && k != "targetRef" && k != "targetSelectors" {
			useful = true
			break
		}
	}
	return out, useful
}

func sanitizeRateLimit(rl map[string]any) map[string]any {
	out := map[string]any{}
	if t, ok := rl["type"]; ok {
		out["type"] = t
	} else if rl["local"] != nil {
		out["type"] = "Local"
	} else if rl["global"] != nil {
		out["type"] = "Global"
	}
	if v, ok := rl["local"]; ok {
		out["local"] = v
	}
	if v, ok := rl["global"]; ok {
		out["global"] = v
	}
	return out
}

func sanitizeLoadBalancer(lb map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"type", "consistentHash", "slowStart"} {
		if v, ok := lb[k]; ok {
			out[k] = v
		}
	}
	return out
}

func sanitizeTimeout(to map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := to["tcp"]; ok {
		out["tcp"] = v
	}
	if v, ok := to["http"]; ok {
		out["http"] = v
	}
	return out
}

func sanitizeConnection(conn map[string]any) map[string]any {
	out := map[string]any{}
	// bufferLimit is stable on BackendConnection since EG 1.2.
	if v, ok := conn["bufferLimit"]; ok {
		out["bufferLimit"] = v
	}
	return out
}

func sanitizeBasicAuth(ba map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := ba["users"]; ok {
		out["users"] = v
	}
	if v, ok := ba["forwardUsernameHeader"]; ok {
		out["forwardUsernameHeader"] = v
	}
	return out
}

// sanitizeExtAuth requires a real backend; scaffolds with empty backendRefs are dropped.
func sanitizeExtAuth(ea map[string]any) (map[string]any, bool) {
	out := map[string]any{}
	httpSvc, _ := ea["http"].(map[string]any)
	grpcSvc, _ := ea["grpc"].(map[string]any)
	if httpSvc != nil {
		cleanHTTP := map[string]any{}
		refs, _ := httpSvc["backendRefs"].([]any)
		ref, _ := httpSvc["backendRef"].(map[string]any)
		if len(refs) == 0 && len(ref) == 0 {
			return nil, false
		}
		if len(refs) > 0 {
			cleanHTTP["backendRefs"] = refs
		}
		if len(ref) > 0 {
			cleanHTTP["backendRef"] = ref
		}
		// EG uses path / pathOverride, not uri.
		if p, ok := httpSvc["path"]; ok {
			cleanHTTP["path"] = p
		} else if uri, ok := httpSvc["uri"].(string); ok && uri != "" {
			cleanHTTP["path"] = uri
		}
		if p, ok := httpSvc["pathOverride"]; ok {
			cleanHTTP["pathOverride"] = p
		}
		out["http"] = cleanHTTP
	}
	if grpcSvc != nil {
		refs, _ := grpcSvc["backendRefs"].([]any)
		ref, _ := grpcSvc["backendRef"].(map[string]any)
		if len(refs) == 0 && len(ref) == 0 {
			return nil, false
		}
		out["grpc"] = grpcSvc
	}
	if len(out) == 0 {
		return nil, false
	}
	for _, k := range []string{"failOpen", "headersToExtAuth", "timeout"} {
		if v, ok := ea[k]; ok {
			out[k] = v
		}
	}
	return out, true
}

func isEnvoyGatewayAPI(apiVersion string) bool {
	return strings.HasPrefix(apiVersion, "gateway.envoyproxy.io/")
}
