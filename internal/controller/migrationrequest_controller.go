package controller

import (
	"context"
	"fmt"
	"os"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gateshiftv1alpha1 "github.com/gateshift/gateshift/api/v1alpha1"
	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/gitops"
	"github.com/gateshift/gateshift/pkg/ir"
)

// MigrationRequestReconciler reconciles MigrationRequest objects.
type MigrationRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile converts the referenced Ingress and optionally opens a GitOps PR.
func (r *MigrationRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var mreq gateshiftv1alpha1.MigrationRequest
	if err := r.Get(ctx, req.NamespacedName, &mreq); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	ns := mreq.Spec.SourceIngressRef.Namespace
	if ns == "" {
		ns = mreq.Namespace
	}
	var ing networkingv1.Ingress
	if err := r.Get(ctx, client.ObjectKey{Name: mreq.Spec.SourceIngressRef.Name, Namespace: ns}, &ing); err != nil {
		return r.fail(ctx, &mreq, fmt.Errorf("source Ingress: %w", err))
	}

	provider, err := ir.ParseProvider(normalizeProvider(mreq.Spec.TargetProvider))
	if err != nil {
		return r.fail(ctx, &mreq, err)
	}
	gwName := ""
	if mreq.Spec.TargetGatewayRef != nil {
		gwName = mreq.Spec.TargetGatewayRef.Name
	}
	gwClass := mreq.Spec.GatewayClassName
	if gwClass == "" {
		gwClass = "envoy"
	}

	bundle, err := convert.FromIngress(&ing, convert.Options{
		Provider:       provider,
		GatewayName:    gwName,
		GatewayClass:   gwClass,
		IncludeGateway: gwName == "" || mreq.Spec.TargetGatewayRef == nil,
	})
	if err != nil {
		return r.fail(ctx, &mreq, err)
	}
	yamlBytes, err := convert.EmitYAML(bundle)
	if err != nil {
		return r.fail(ctx, &mreq, err)
	}

	d, p, u := bundle.Summary()
	mreq.Status.AuditSummary = &gateshiftv1alpha1.AuditSummary{
		Direct:         d,
		RequiresPolicy: p,
		Untranslatable: u,
	}

	if mreq.Spec.DryRun || mreq.Spec.GitOps == nil || !mreq.Spec.GitOps.AutoCreatePR {
		// Persist rendered manifests as an annotation-sized summary message.
		mreq.Status.Phase = "Converted"
		mreq.Status.Message = fmt.Sprintf("Converted %s/%s (%d bytes YAML); dryRun=%v", ing.Namespace, ing.Name, len(yamlBytes), mreq.Spec.DryRun)
		if err := r.Status().Update(ctx, &mreq); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("migration converted", "ingress", ing.Name, "direct", d, "policy", p, "manual", u)
		return ctrl.Result{}, nil
	}

	pr, err := gitops.CreateMigrationPR(ctx, gitops.PRRequest{
		Repo:         mreq.Spec.GitOps.GitHubRepo,
		BaseBranch:   mreq.Spec.GitOps.BaseBranch,
		BranchPrefix: mreq.Spec.GitOps.BranchPrefix,
		ManifestYAML: yamlBytes,
		Bundle:       bundle,
		IngressName:  ing.Name,
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
	})
	if err != nil {
		return r.fail(ctx, &mreq, err)
	}
	mreq.Status.Phase = "PRCreated"
	mreq.Status.PRURL = pr.URL
	mreq.Status.Message = "GitOps pull request created"
	if pr.DryRun {
		mreq.Status.Phase = "PRDryRun"
		mreq.Status.Message = "GitOps dry-run artifacts written (no GITHUB_TOKEN)"
	}
	if err := r.Status().Update(ctx, &mreq); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *MigrationRequestReconciler) fail(ctx context.Context, mreq *gateshiftv1alpha1.MigrationRequest, err error) (ctrl.Result, error) {
	mreq.Status.Phase = "Error"
	mreq.Status.Message = err.Error()
	mreq.Status.Conditions = []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "ReconcileError",
		Message:            err.Error(),
		LastTransitionTime: metav1.Now(),
	}}
	_ = r.Status().Update(ctx, mreq)
	return ctrl.Result{}, err
}

func normalizeProvider(s string) string {
	switch s {
	case "EnvoyGateway":
		return "envoy-gateway"
	case "Cilium":
		return "cilium"
	case "Istio":
		return "istio"
	case "Kong":
		return "kong"
	case "Standard":
		return "standard"
	default:
		return s
	}
}

// SetupWithManager registers the controller.
func (r *MigrationRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateshiftv1alpha1.MigrationRequest{}).
		Complete(r)
}
