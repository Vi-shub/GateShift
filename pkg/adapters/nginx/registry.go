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
		// Level 2
		RateLimitAdapter{},
		CertManagerAdapter{},
		AffinityAdapter{},
		IPAllowAdapter{},
		TimeoutBodyAdapter{},
		CanaryAdapter{},
		RegexAdapter{},
		// Level 3
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
