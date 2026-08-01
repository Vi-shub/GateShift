package nginx

import (
	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// RewriteAdapter maps rewrite-target → HTTPRoute URLRewrite (Level 1).
type RewriteAdapter struct{}

func (RewriteAdapter) Name() string       { return "rewrite-target" }
func (RewriteAdapter) Level() adapters.Level { return adapters.Level1 }
func (RewriteAdapter) CanHandle(key string) bool {
	return key == AnnRewriteTarget
}

func (RewriteAdapter) Transform(key, value string, ctx *adapters.Context) error {
	rewritten := normalizeRewrite(value)
	ctx.Filters = append(ctx.Filters, ir.FilterIR{
		Kind:               ir.FilterURLRewrite,
		ReplacePrefixMatch: strPtr(rewritten),
	})
	ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1,
		"HTTPRoute.spec.rules[].filters[type=URLRewrite]",
		"Mapped rewrite-target to URLRewrite ReplacePrefixMatch")
	return nil
}
