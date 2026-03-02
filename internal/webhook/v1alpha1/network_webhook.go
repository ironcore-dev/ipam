// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"math"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// log is for logging in this package.
var networklog = logf.Log.WithName("network-resource")

func SetupNetworkWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &v1alpha1.Network{}).
		WithValidator(&NetworkCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ipam-metal-ironcore-dev-v1alpha1-network,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=networks,verbs=create;update,versions=v1,name=mnetwork-v1alpha1.kb.io,admissionReviewVersions=v1

// +kubebuilder:webhook:path=/validate-ipam-metal-ironcore-dev-v1alpha1-network,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=networks,verbs=create;update;delete,versions=v1alpha1,name=vnetwork.kb.io,admissionReviewVersions={v1,v1beta1}

// NetworkCustomValidator struct is responsible for validating the IP resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NetworkCustomValidator struct {
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCustomValidator) ValidateCreate(ctx context.Context, obj *v1alpha1.Network) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	networklog.Info("validate create", "name", obj.GetName())

	if obj.Spec.Type == "" && obj.Spec.ID != nil {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.id"), obj.Spec.ID, "setting network ID without type is disallowed"))
	}

	if err := validateID(obj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		gvk := obj.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return warnings, apierrors.NewInvalid(gk, obj.Name, allErrs)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *v1alpha1.Network) (admission.Warnings, error) {
	var warnings admission.Warnings
	var allErrs field.ErrorList

	networklog.Info("validate update", "name", oldObj.GetName())

	if oldObj.Spec.Type != "" &&
		oldObj.Spec.Type != newObj.Spec.Type {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.type"), newObj.Spec.Type, "network type change is disallowed; resource should be released (deleted) first"))
	}

	if (oldObj.Spec.ID != nil && oldObj.Spec.ID.Cmp(&newObj.Spec.ID.Int) != 0) ||
		(oldObj.Spec.ID == nil && oldObj.Spec.Type != "" && newObj.Spec.ID != nil) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.id"), newObj.Spec.ID,
			"network ID change after assignment is disallowed; resource should be released (deleted) first"))
	}

	if err := validateID(newObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		gvk := newObj.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return warnings, apierrors.NewInvalid(gk, newObj.Name, allErrs)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCustomValidator) ValidateDelete(ctx context.Context, obj *v1alpha1.Network) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	networklog.Info("validate delete", "name", obj.Name)

	if len(obj.Status.IPv4Ranges) > 0 {
		allErrs = append(allErrs, field.InternalError(
			field.NewPath("metadata.name"), errors.New("Network has active IPv4 subnets")))
	}

	if len(obj.Status.IPv6Ranges) > 0 {
		allErrs = append(allErrs, field.InternalError(
			field.NewPath("metadata.name"), errors.New("Network has active IPv6 subnets")))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{
				Group: v1alpha1.SchemeGroupVersion.Group,
				Kind:  "Network",
			}, obj.Name, allErrs)
	}

	return warnings, nil
}

func validateID(in *v1alpha1.Network) *field.Error {
	if in.Spec.ID == nil {
		return nil
	}

	switch in.Spec.Type {
	case v1alpha1.VXLANNetworkType:
		if in.Spec.ID.Cmp(&v1alpha1.VXLANFirstAvaliableID.Int) < 0 ||
			in.Spec.ID.Cmp(&v1alpha1.VXLANMaxID.Int) > 0 {
			return field.Invalid(field.NewPath("spec.id"), in.Spec.ID, fmt.Sprintf("value for the ID for network type %s should be in interval [%s; %s]", in.Spec.Type, v1alpha1.VXLANFirstAvaliableID, v1alpha1.VXLANMaxID))
		}
	case v1alpha1.GENEVENetworkType:
		if in.Spec.ID.Cmp(&v1alpha1.GENEVEFirstAvaliableID.Int) < 0 ||
			in.Spec.ID.Cmp(&v1alpha1.GENEVEMaxID.Int) > 0 {
			return field.Invalid(field.NewPath("spec.id"), in.Spec.ID, fmt.Sprintf("value for the ID for network type %s should be in interval [%s; %s]", in.Spec.Type, v1alpha1.GENEVEFirstAvaliableID, v1alpha1.GENEVEMaxID))
		}
	case v1alpha1.MPLSNetworkType:
		if in.Spec.ID.Cmp(&v1alpha1.MPLSFirstAvailableID.Int) < 0 {
			return field.Invalid(field.NewPath("spec.id"), in.Spec.ID, fmt.Sprintf("value for the ID for network type %s should be in interval [%s; %f]", in.Spec.Type, v1alpha1.MPLSFirstAvailableID, math.Inf(1)))
		}
	default:
		return field.Invalid(field.NewPath("spec.type"), in.Spec.Type, "unknown network type")
	}

	return nil
}
