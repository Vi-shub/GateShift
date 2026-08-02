package nginx

import (
	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// AuditMeta is kept for convert package compatibility.
type AuditMeta = adapters.Meta

// Result is the facade shape used by convert.
type Result struct {
	Filters        []ir.FilterIR
	Policies       []ir.PolicyIR
	Certificates   []ir.CertificateIR
	Findings       []ir.AuditFinding
	TLS            *ir.TLSIR
	DefaultBackend string
	SSLPassthrough bool
}

// Translate converts annotations via the plug-in adapter registry.
func Translate(annotations map[string]string, provider ir.Provider, meta AuditMeta) Result {
	ctx := Default.Translate(annotations, provider, meta)
	return Result{
		Filters:        ctx.Filters,
		Policies:       ctx.Policies,
		Certificates:   ctx.Certificates,
		Findings:       ctx.Findings,
		TLS:            ctx.TLS,
		DefaultBackend: ctx.DefaultBackend,
		SSLPassthrough: ctx.SSLPassthrough,
	}
}
