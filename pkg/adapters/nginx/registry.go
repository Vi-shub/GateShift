package nginx

import "github.com/gateshift/gateshift/pkg/adapters"

// DefaultAdapters returns the built-in NGINX / cert-manager adapter plug-ins.
func DefaultAdapters() []adapters.AnnotationAdapter {
	return []adapters.AnnotationAdapter{
		// Level 1
		RewriteAdapter{},
		SSLRedirectAdapter{},
		ExternalRedirectAdapter{},
		CORSAdapter{},
		WWWRedirectAdapter{},
		AppRootAdapter{},
		XForwardedPrefixAdapter{},
		// Level 2
		RateLimitAdapter{},
		CertManagerAdapter{},
		AffinityAdapter{},
		IPAllowAdapter{},
		TimeoutBodyAdapter{},
		CanaryAdapter{},
		RegexAdapter{},
		BackendTLSAdapter{},
		MirrorAdapter{},
		// Level 3 (+ pattern promotion)
		SnippetAdapter{},
		AuthAdapter{},
	}
}

// NewRegistry builds a registry with all built-in NGINX adapters.
func NewRegistry() *adapters.Registry {
	return adapters.NewRegistry(DefaultAdapters()...)
}

// Default is the package-level registry used by Translate.
var Default = NewRegistry()
