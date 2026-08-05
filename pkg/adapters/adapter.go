// Package adapters defines the plug-in annotation translation engine.
//
// Difficulty levels:
//
//	Level1 — Direct Gateway API HTTPRoute filter mappings
//	Level2 — Provider-specific Policy / Certificate CRDs
//	Level3 — Untranslatable nginx magic (snippets, Lua, auth-url)
package adapters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gateshift/gateshift/pkg/ir"
)

// Level classifies how hard an annotation is to migrate.
type Level int

const (
	Level1 Level = 1 // Direct standard Gateway API mapping
	Level2 Level = 2 // Provider extension / secondary CRD
	Level3 Level = 3 // Hard / impossible — flag for humans
)

func (l Level) String() string {
	switch l {
	case Level1:
		return "L1"
	case Level2:
		return "L2"
	case Level3:
		return "L3"
	default:
		return fmt.Sprintf("L%d", int(l))
	}
}

// Meta identifies the source Ingress for audit findings.
type Meta struct {
	IngressName string
	Namespace   string
}

// Context is mutable state adapters write into during Transform.
type Context struct {
	Provider     ir.Provider
	Meta         Meta
	Annotations  map[string]string
	Filters      []ir.FilterIR
	Policies     []ir.PolicyIR
	Certificates []ir.CertificateIR
	Findings     []ir.AuditFinding
	TLS          *ir.TLSIR
	Claimed      map[string]bool

	// Optional structured intents consumed by convert.
	DefaultBackend string // service name or namespace/name from annotation
	SSLPassthrough bool
}

// Claim marks annotation keys as handled (for multi-key adapters).
func (c *Context) Claim(keys ...string) {
	if c.Claimed == nil {
		c.Claimed = map[string]bool{}
	}
	for _, k := range keys {
		c.Claimed[k] = true
	}
}

// AddFinding appends a classified audit finding with a stable ID.
func (c *Context) AddFinding(key, value string, status ir.Status, level Level, target, msg string) {
	c.Claim(key)
	f := ir.NewFinding("annotation."+sanitizeID(key), status, int(level), key, msg).
		WithValue(value).
		WithTarget(target).
		WithEvidence(ir.Evidence{
			IngressName: c.Meta.IngressName,
			Namespace:   c.Meta.Namespace,
			Annotation:  key,
		})
	c.Findings = append(c.Findings, f)
}

func sanitizeID(key string) string {
	b := make([]byte, 0, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b = append(b, c)
		default:
			b = append(b, '.')
		}
	}
	return string(b)
}

// AnnotationAdapter is a plug-in that handles one annotation family.
type AnnotationAdapter interface {
	Name() string
	Level() Level
	CanHandle(key string) bool
	Transform(key, value string, ctx *Context) error
}

// Registry dispatches annotations to registered adapters.
type Registry struct {
	adapters []AnnotationAdapter
}

// NewRegistry returns a registry with the given adapters.
func NewRegistry(list ...AnnotationAdapter) *Registry {
	return &Registry{adapters: append([]AnnotationAdapter{}, list...)}
}

// Register appends an adapter.
func (r *Registry) Register(a AnnotationAdapter) {
	r.adapters = append(r.adapters, a)
}

// Translate runs matching adapters over an annotation map.
func (r *Registry) Translate(annotations map[string]string, provider ir.Provider, meta Meta) *Context {
	ctx := &Context{
		Provider:    provider,
		Meta:        meta,
		Annotations: annotations,
		Claimed:     map[string]bool{},
	}
	if len(annotations) == 0 {
		return ctx
	}

	keys := make([]string, 0, len(annotations))
	for k := range annotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if ctx.Claimed[key] {
			continue
		}
		value := annotations[key]
		matched := false
		for _, a := range r.adapters {
			if !a.CanHandle(key) {
				continue
			}
			matched = true
			_ = a.Transform(key, value, ctx)
			ctx.Claim(key)
			break
		}
		if !matched && isMigrationAnnotation(key) {
			// Unknown / unhandled annotations become IR findings (never silent).
			// Warn-level so validate fail-closed is reserved for true L3 (snippets, etc.).
			f := ir.NewFinding(ir.FindingIDAnnotationUnknown, ir.StatusRequiresPolicy, 2, key,
				fmt.Sprintf("Unknown or unhandled migration annotation %s — recorded for coverage; not silently dropped", key)).
				WithValue(value).
				WithTarget("catalog / new adapter").
				WithEvidence(ir.Evidence{
					IngressName: meta.IngressName,
					Namespace:   meta.Namespace,
					Annotation:  key,
				})
			ctx.Findings = append(ctx.Findings, f)
			ctx.Claim(key)
		}
	}
	return ctx
}

func isMigrationAnnotation(key string) bool {
	return strings.HasPrefix(key, "nginx.ingress.kubernetes.io/") ||
		strings.HasPrefix(key, "cert-manager.io/") ||
		strings.HasPrefix(key, "nginx.org/")
}
