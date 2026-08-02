package nginx

import "github.com/gateshift/gateshift/pkg/adapters"

// CatalogEntry describes a known annotation GateShift tracks for coverage scoring.
type CatalogEntry struct {
	Key         string
	Level       adapters.Level
	Implemented bool
	Notes       string
}

// Catalog is the frequency-ranked NGINX / cert-manager annotation set GateShift
// aims to cover. Rank order roughly follows public Ingress corpus frequency.
func Catalog() []CatalogEntry {
	return []CatalogEntry{
		// Core L1
		{AnnRewriteTarget, adapters.Level1, true, "URLRewrite"},
		{AnnSSLRedirect, adapters.Level1, true, "RequestRedirect"},
		{AnnForceSSLRedirect, adapters.Level1, true, "RequestRedirect"},
		{AnnPermanentRedirect, adapters.Level1, true, "RequestRedirect"},
		{AnnTemporalRedirect, adapters.Level1, true, "RequestRedirect"},
		{AnnFromToWWWRedirect, adapters.Level1, true, "RequestRedirect hostname"},
		{AnnAppRoot, adapters.Level1, true, "RequestRedirect to app root"},
		{AnnCORSEnable, adapters.Level1, true, "CORS headers"},
		{AnnCORSAllowOrigin, adapters.Level1, true, "CORS headers"},
		{AnnCORSAllowMethods, adapters.Level1, true, "CORS headers"},
		{AnnCORSAllowHeaders, adapters.Level1, true, "CORS headers"},
		{AnnXForwardedPrefix, adapters.Level1, true, "RequestHeaderModifier"},
		{AnnUseRegex, adapters.Level2, true, "RegularExpression path"},
		// L2 traffic / security
		{AnnLimitRPS, adapters.Level2, true, "BackendTrafficPolicy"},
		{AnnLimitRPM, adapters.Level2, true, "BackendTrafficPolicy"},
		{AnnLimitConnections, adapters.Level2, true, "BackendTrafficPolicy"},
		{AnnWhitelistSourceRange, adapters.Level2, true, "SecurityPolicy"},
		{AnnDenylistSourceRange, adapters.Level2, true, "SecurityPolicy"},
		{AnnAffinity, adapters.Level2, true, "SessionPersistence"},
		{AnnSessionCookieName, adapters.Level2, true, "SessionPersistence"},
		{AnnSessionCookieExpires, adapters.Level2, true, "SessionPersistence cookie TTL"},
		{AnnSessionCookieMaxAge, adapters.Level2, true, "SessionPersistence cookie TTL"},
		{AnnSessionCookieSecure, adapters.Level2, true, "SessionPersistence cookie Secure"},
		{AnnSessionCookieSameSite, adapters.Level2, true, "SessionPersistence cookie SameSite"},
		{AnnSessionCookieConditionalSameSiteNone, adapters.Level2, true, "SessionPersistence conditional SameSite=None"},
		{AnnSessionCookiePath, adapters.Level2, true, "SessionPersistence cookie path"},
		{AnnSessionCookieChangeOnFailure, adapters.Level2, true, "SessionPersistence cookie rotate"},
		{AnnSessionCookieHash, adapters.Level2, true, "SessionPersistence cookie hash"},
		{AnnProxyBodySize, adapters.Level2, true, "BackendTrafficPolicy"},
		{AnnProxyReadTimeout, adapters.Level2, true, "BackendTrafficPolicy"},
		{AnnProxySendTimeout, adapters.Level2, true, "BackendTrafficPolicy"},
		{AnnBackendProtocol, adapters.Level2, true, "BackendTLS / protocol"},
		{AnnProxySSLSecret, adapters.Level2, true, "BackendTLSPolicy"},
		{AnnProxySSLVerify, adapters.Level2, true, "BackendTLSPolicy"},
		{AnnCanary, adapters.Level2, true, "Weighted backends"},
		{AnnCanaryWeight, adapters.Level2, true, "Weighted backends"},
		{AnnCanaryByHeader, adapters.Level2, true, "Header match"},
		{AnnMirror, adapters.Level2, true, "RequestMirror filter"},
		{AnnMirrorTarget, adapters.Level2, true, "RequestMirror filter"},
		{AnnUpstreamHashBy, adapters.Level2, true, "consistentHash policy"},
		{AnnAuthURL, adapters.Level2, true, "SecurityPolicy extAuth scaffold"},
		{AnnAuthSignin, adapters.Level2, true, "SecurityPolicy extAuth scaffold"},
		{AnnAuthTLSSecret, adapters.Level2, true, "client mTLS"},
		{AnnCertManagerClusterIssuer, adapters.Level2, true, "Certificate"},
		{AnnCertManagerIssuer, adapters.Level2, true, "Certificate"},
		// L3 — pattern-assisted
		{AnnConfigurationSnippet, adapters.Level3, true, "pattern library promotion"},
		{AnnServerSnippet, adapters.Level3, true, "pattern library promotion"},
		{AnnModsecuritySnippet, adapters.Level3, true, "always manual / WAF"},
		// Known but not yet implemented (tracked gaps vs ingress2gateway+)
		{AnnEnableAccessLog, adapters.Level2, false, "TODO: Observability / access log policy"},
		{AnnCustomHTTPErrors, adapters.Level2, false, "TODO: custom error pages → filter/direct response"},
		{AnnDefaultBackend, adapters.Level1, false, "TODO: prefer spec.defaultBackend"},
		{AnnProxyBuffering, adapters.Level2, false, "TODO: BackendTrafficPolicy buffering"},
		{AnnProxyNextUpstream, adapters.Level2, false, "TODO: retry policy"},
		{AnnSSLPassthrough, adapters.Level2, false, "TODO: TLSRoute / passthrough listener"},
		{AnnAuthType, adapters.Level3, false, "TODO: basic auth → SecurityPolicy"},
		{AnnAuthSecret, adapters.Level3, false, "TODO: basic auth secret"},
		{AnnClientBodyBufferSize, adapters.Level2, false, "TODO: body buffer policy"},
		{AnnProxyConnectTimeout, adapters.Level2, false, "TODO: connect timeout"},
	}
}

// CoverageStats summarizes catalog implementation status.
type CoverageStats struct {
	Total        int
	Implemented  int
	ByLevel      map[adapters.Level]int
	MissingKeys  []string
	Percent      float64
}

// CatalogCoverage returns how much of the tracked annotation catalog is implemented.
func CatalogCoverage() CoverageStats {
	cat := Catalog()
	stats := CoverageStats{Total: len(cat), ByLevel: map[adapters.Level]int{}}
	for _, e := range cat {
		if e.Implemented {
			stats.Implemented++
			stats.ByLevel[e.Level]++
		} else {
			stats.MissingKeys = append(stats.MissingKeys, e.Key)
		}
	}
	if stats.Total > 0 {
		stats.Percent = float64(stats.Implemented) / float64(stats.Total) * 100
	}
	return stats
}
