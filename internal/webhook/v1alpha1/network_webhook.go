// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var networklog = logf.Log.WithName("network-resource")

func SetupNetworkWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.Network{}).
		WithValidator(&NetworkCustomValidator{}).
		WithDefaulter(&NetworkCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ipam-metal-ironcore-dev-v1alpha1-network,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=networks,verbs=create;update,versions=v1,name=mnetwork-v1alpha1.kb.io,admissionReviewVersions=v1

// NetworkCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Network when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type NetworkCustomDefaulter struct {
}

var _ webhook.CustomDefaulter = &NetworkCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Network
func (d *NetworkCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	net, ok := obj.(*v1alpha1.Network)

	if !ok {
		return fmt.Errorf("expected an Network object but got %T", obj)
	}
	networklog.Info("Defaulting for Network", "name", net.GetName())

	return nil
}

// +kubebuilder:webhook:path=/validate-ipam-metal-ironcore-dev-v1alpha1-network,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=networks,verbs=create;update;delete,versions=v1alpha1,name=vnetwork.kb.io,admissionReviewVersions={v1,v1beta1}

// NetworkCustomValidator struct is responsible for validating the IP resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NetworkCustomValidator struct {
}

var _ webhook.CustomValidator = &IPCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	net, ok := obj.(*v1alpha1.Network)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to Network CR type"))
	}
	networklog.Info("validate create", "name", net.GetName())

	if net.Spec.Type == "" && net.Spec.ID != nil {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.id"), net.Spec.ID, "setting network ID without type is disallowed"))
	}

	if err := validateID(net); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		gvk := net.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return warnings, apierrors.NewInvalid(gk, net.Name, allErrs)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	var warnings admission.Warnings
	var allErrs field.ErrorList

	oldNetwork, ok := oldObj.(*v1alpha1.Network)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to Network CR type"))
	}
	newNetwork, ok := newObj.(*v1alpha1.Network)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to Network CR type"))
	}
	networklog.Info("validate update", "name", oldNetwork.GetName())

	if oldNetwork.Spec.Type != "" &&
		oldNetwork.Spec.Type != newNetwork.Spec.Type {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.type"), newNetwork.Spec.Type, "network type change is disallowed; resource should be released (deleted) first"))
	}

	if (oldNetwork.Spec.ID != nil && oldNetwork.Spec.ID.Cmp(&newNetwork.Spec.ID.Int) != 0) ||
		(oldNetwork.Spec.ID == nil && oldNetwork.Spec.Type != "" && newNetwork.Spec.ID != nil) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.id"), newNetwork.Spec.ID,
			"network ID change after assignment is disallowed; resource should be released (deleted) first"))
	}

	if err := validateID(newNetwork); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		gvk := newNetwork.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return warnings, apierrors.NewInvalid(gk, newNetwork.Name, allErrs)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	network, ok := obj.(*v1alpha1.Network)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to Network CR type"))
	}
	networklog.Info("validate delete", "name", network.Name)

	if len(network.Status.IPv4Ranges) > 0 {
		allErrs = append(allErrs, field.InternalError(
			field.NewPath("metadata.name"), errors.New("Network has active IPv4 subnets")))
	}

	if len(network.Status.IPv6Ranges) > 0 {
		allErrs = append(allErrs, field.InternalError(
			field.NewPath("metadata.name"), errors.New("Network has active IPv6 subnets")))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{
				Group: v1alpha1.SchemeGroupVersion.Group,
				Kind:  "Network",
			}, network.Name, allErrs)
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
