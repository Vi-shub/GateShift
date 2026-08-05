package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MigrationRequestSpec defines an Ingress → Gateway API migration.
type MigrationRequestSpec struct {
	SourceIngressRef ObjectRef   `json:"sourceIngressRef"`
	TargetGatewayRef *ObjectRef  `json:"targetGatewayRef,omitempty"`
	TargetProvider   string      `json:"targetProvider"`
	GatewayClassName string      `json:"gatewayClassName,omitempty"`
	DryRun           bool        `json:"dryRun,omitempty"`
	GitOps           *GitOpsSpec `json:"gitops,omitempty"`
}

// ObjectRef references a namespaced object.
type ObjectRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// GitOpsSpec configures automated PR creation.
type GitOpsSpec struct {
	GitHubRepo   string `json:"githubRepo,omitempty"`
	AutoCreatePR bool   `json:"autoCreatePR,omitempty"`
	BranchPrefix string `json:"branchPrefix,omitempty"`
	BaseBranch   string `json:"baseBranch,omitempty"`
}

// MigrationRequestStatus is the observed state.
type MigrationRequestStatus struct {
	Phase        string             `json:"phase,omitempty"`
	Message      string             `json:"message,omitempty"`
	PRURL        string             `json:"prURL,omitempty"`
	AuditSummary *AuditSummary      `json:"auditSummary,omitempty"`
	Conditions   []metav1.Condition `json:"conditions,omitempty"`
}

// AuditSummary counts translation outcomes.
type AuditSummary struct {
	Direct         int `json:"direct"`
	RequiresPolicy int `json:"requiresPolicy"`
	Untranslatable int `json:"untranslatable"`
}

// MigrationRequest is the Schema for the migrationrequests API.
type MigrationRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MigrationRequestSpec   `json:"spec,omitempty"`
	Status MigrationRequestStatus `json:"status,omitempty"`
}

// MigrationRequestList contains a list of MigrationRequest.
type MigrationRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MigrationRequest `json:"items"`
}
