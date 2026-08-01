package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies the receiver into out.
func (in *MigrationRequest) DeepCopyInto(out *MigrationRequest) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy returns a copy of the receiver.
func (in *MigrationRequest) DeepCopy() *MigrationRequest {
	if in == nil {
		return nil
	}
	out := new(MigrationRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a runtime.Object copy.
func (in *MigrationRequest) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out.
func (in *MigrationRequestList) DeepCopyInto(out *MigrationRequestList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MigrationRequest, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy returns a copy of the receiver.
func (in *MigrationRequestList) DeepCopy() *MigrationRequestList {
	if in == nil {
		return nil
	}
	out := new(MigrationRequestList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a runtime.Object copy.
func (in *MigrationRequestList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies Spec.
func (in *MigrationRequestSpec) DeepCopyInto(out *MigrationRequestSpec) {
	*out = *in
	out.SourceIngressRef = in.SourceIngressRef
	if in.TargetGatewayRef != nil {
		in, out := &in.TargetGatewayRef, &out.TargetGatewayRef
		*out = new(ObjectRef)
		**out = **in
	}
	if in.GitOps != nil {
		in, out := &in.GitOps, &out.GitOps
		*out = new(GitOpsSpec)
		**out = **in
	}
}

// DeepCopyInto copies Status.
func (in *MigrationRequestStatus) DeepCopyInto(out *MigrationRequestStatus) {
	*out = *in
	if in.AuditSummary != nil {
		in, out := &in.AuditSummary, &out.AuditSummary
		*out = new(AuditSummary)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
