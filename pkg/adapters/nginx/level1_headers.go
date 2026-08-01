package nginx

import (
	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// CORSAdapter maps enable-cors + cors-allow-* → ResponseHeaderModifier (Level 1).
type CORSAdapter struct{}

func (CORSAdapter) Name() string          { return "cors" }
func (CORSAdapter) Level() adapters.Level { return adapters.Level1 }
func (CORSAdapter) CanHandle(key string) bool {
	switch key {
	case AnnCORSEnable, AnnCORSAllowOrigin, AnnCORSAllowMethods, AnnCORSAllowHeaders,
		AnnCORSExposeHeaders, AnnCORSAllowCredentials:
		return true
	default:
		return false
	}
}

func (CORSAdapter) Transform(key, value string, ctx *adapters.Context) error {
	siblings := []string{
		AnnCORSEnable, AnnCORSAllowOrigin, AnnCORSAllowMethods, AnnCORSAllowHeaders,
		AnnCORSExposeHeaders, AnnCORSAllowCredentials,
	}
	// Already handled by a prior CORS key in this pass.
	for _, s := range siblings {
		if s == key {
			continue
		}
		if ctx.Claimed[s] {
			ctx.Claim(key)
			return nil
		}
	}

	enabled := isTruthy(ctx.Annotations[AnnCORSEnable])
	origin := ctx.Annotations[AnnCORSAllowOrigin]
	if !enabled && origin == "" {
		ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1, "", "CORS disabled; no filter emitted")
		ctx.Claim(siblings...)
		return nil
	}
	if origin == "" {
		origin = "*"
	}
	set := map[string]string{
		"Access-Control-Allow-Origin":  origin,
		"Access-Control-Allow-Methods": "GET, PUT, POST, DELETE, PATCH, OPTIONS",
	}
	if m := ctx.Annotations[AnnCORSAllowMethods]; m != "" {
		set["Access-Control-Allow-Methods"] = m
	}
	if h := ctx.Annotations[AnnCORSAllowHeaders]; h != "" {
		set["Access-Control-Allow-Headers"] = h
	}
	if e := ctx.Annotations[AnnCORSExposeHeaders]; e != "" {
		set["Access-Control-Expose-Headers"] = e
	}
	if isTruthy(ctx.Annotations[AnnCORSAllowCredentials]) {
		set["Access-Control-Allow-Credentials"] = "true"
	}

	ctx.Filters = append(ctx.Filters, ir.FilterIR{
		Kind:       ir.FilterResponseHeader,
		SetHeaders: set,
	})
	ctx.AddFinding(AnnCORSAllowOrigin, origin, ir.StatusDirect, adapters.Level1,
		"HTTPRoute.spec.rules[].filters[type=ResponseHeaderModifier]",
		"Mapped CORS annotations to response header modifiers")
	ctx.Claim(siblings...)
	return nil
}
