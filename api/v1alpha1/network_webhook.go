package v1alpha1

import (
	"fmt"
	"math"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var networklog = logf.Log.WithName("network-resource")

func (n *Network) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(n).
		Complete()
}

// +kubebuilder:webhook:path=/validate-ipam-onmetal-de-v1alpha1-network,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=networks,verbs=create;update,versions=v1alpha1,name=vnetwork.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Network{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (n *Network) ValidateCreate() error {
	networklog.Info("validate create", "name", n.Name)

	var allErrs field.ErrorList

	if n.Spec.Type == "" && n.Spec.ID != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.id"), n.Spec.ID, "setting network ID without type is disallowed"))
	}

	if err := n.validateID(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		gvk := n.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return apierrors.NewInvalid(gk, n.Name, allErrs)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (n *Network) ValidateUpdate(old runtime.Object) error {
	networklog.Info("validate update", "name", n.Name)
	oldNetwork, ok := old.(*Network)
	if !ok {
		return apierrors.NewInternalError(errors.New("cannot cast previous object version to Network CR type"))
	}

	var allErrs field.ErrorList

	if oldNetwork.Spec.Type != "" &&
		oldNetwork.Spec.Type != n.Spec.Type {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.type"), n.Spec.ID, "network type change is disallowed; resource should be released (deleted) first"))
	}

	if (oldNetwork.Spec.ID != nil &&
		oldNetwork.Spec.ID.Cmp(&n.Spec.ID.Int) != 0) ||
		(oldNetwork.Spec.ID == nil &&
			oldNetwork.Spec.Type != "") {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.id"), n.Spec.ID, "network ID change after assignment is disallowed; resource should be released (deleted) first"))
	}

	if err := n.validateID(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		gvk := n.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return apierrors.NewInvalid(gk, n.Name, allErrs)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (n *Network) ValidateDelete() error {
	networklog.Info("validate delete", "name", n.Name)
	return nil
}

func (n *Network) validateID() *field.Error {
	if n.Spec.ID == nil {
		return nil
	}

	switch n.Spec.Type {
	case CVXLANNetworkType:
		if n.Spec.ID.Cmp(&CVXLANFirstAvaliableID.Int) < 0 ||
			n.Spec.ID.Cmp(&CVXLANMaxID.Int) > 0 {
			return field.Invalid(field.NewPath("spec.id"), n.Spec.ID, fmt.Sprintf("value for the ID for network type %s should be in interval [%s; %s]", n.Spec.Type, CVXLANFirstAvaliableID, CVXLANMaxID))
		}
	case CMPLSNetworkType:
		if n.Spec.ID.Cmp(&CMPLSFirstAvailableID.Int) < 0 {
			return field.Invalid(field.NewPath("spec.id"), n.Spec.ID, fmt.Sprintf("value for the ID for network type %s should be in interval [%s; %f]", n.Spec.Type, CVXLANFirstAvaliableID, math.Inf(1)))
		}
	default:
		return field.Invalid(field.NewPath("spec.type"), n.Spec.Type, "unknown network type")
	}

	return nil
}
