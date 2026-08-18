package ir

import "strings"

// Severity ranks how loudly a finding should block automation.
type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityBlock Severity = "block"
)

// Stable finding IDs used across audit / CI / docs.
const (
	FindingIDSpecRules              = "route.spec-rules"
	FindingIDAnnotationUnknown      = "annotation.unknown"
	FindingIDPathTypeRegex          = "path.regex"
	FindingIDPathTypeImplSpecific   = "path.implementation-specific"
	FindingIDQuirkHostRegex         = "quirk.host-regex"
	FindingIDQuirkPathAsRegex       = "quirk.path-as-regex"
	FindingIDQuirkTrailingSlash     = "quirk.trailing-slash"
	FindingIDQuirkTrailingSlashEmit = "quirk.trailing-slash-emit"
	FindingIDQuirkPreserveRegex     = "quirk.preserve-regex"
	FindingIDQuirkURLNormalization  = "quirk.url-normalization"
	FindingIDCanaryMerge            = "canary.merge"
	FindingIDDualRun                = "dual-run.shadow"
	FindingIDHTTPOnly               = "convert.http-only"
)

// Evidence pins a finding to source objects.
type Evidence struct {
	IngressName string `json:"ingressName,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Host        string `json:"host,omitempty"`
	Path        string `json:"path,omitempty"`
	Annotation  string `json:"annotation,omitempty"`
}

// NewFinding builds a first-class IR finding.
func NewFinding(id string, status Status, level int, key, message string) AuditFinding {
	sev := SeverityInfo
	switch status {
	case StatusRequiresPolicy:
		sev = SeverityWarn
	case StatusUntranslatable:
		sev = SeverityBlock
	}
	return AuditFinding{
		ID:       id,
		Key:      key,
		Status:   status,
		Level:    level,
		Message:  message,
		Severity: sev,
	}
}

// WithValue sets Value.
func (f AuditFinding) WithValue(v string) AuditFinding {
	f.Value = v
	return f
}

// WithTarget sets Target.
func (f AuditFinding) WithTarget(t string) AuditFinding {
	f.Target = t
	return f
}

// WithEvidence sets evidence + legacy ingress/namespace fields.
func (f AuditFinding) WithEvidence(ev Evidence) AuditFinding {
	f.Evidence = ev
	f.IngressName = ev.IngressName
	f.Namespace = ev.Namespace
	if f.Key == "" && ev.Annotation != "" {
		f.Key = ev.Annotation
	}
	return f
}

// WithFix marks the finding as automatically remediable.
func (f AuditFinding) WithFix(flag string) AuditFinding {
	f.Fixable = true
	f.Fix = flag
	return f
}

// NormalizeFindings fills defaults so older call sites remain valid.
func NormalizeFindings(findings []AuditFinding) {
	for i := range findings {
		f := &findings[i]
		if f.ID == "" {
			switch {
			case strings.HasPrefix(f.Key, "gateshift.io/nginx-quirk/"):
				f.ID = "quirk." + strings.TrimPrefix(f.Key, "gateshift.io/nginx-quirk/")
			case f.Key != "":
				f.ID = "annotation." + sanitizeFindingID(f.Key)
			default:
				f.ID = "finding.unnamed"
			}
		}
		if f.Severity == "" {
			switch f.Status {
			case StatusRequiresPolicy:
				f.Severity = SeverityWarn
			case StatusUntranslatable:
				f.Severity = SeverityBlock
			default:
				f.Severity = SeverityInfo
			}
		}
		if f.Evidence.IngressName == "" {
			f.Evidence.IngressName = f.IngressName
		}
		if f.Evidence.Namespace == "" {
			f.Evidence.Namespace = f.Namespace
		}
		if f.Evidence.Annotation == "" && f.Key != "" && !strings.HasPrefix(f.Key, "gateshift.io/") {
			f.Evidence.Annotation = f.Key
		}
	}
}

func sanitizeFindingID(key string) string {
	b := make([]byte, 0, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b = append(b, c)
		case c == '/' || c == '.' || c == '-' || c == '_':
			b = append(b, '.')
		}
	}
	return string(b)
}
