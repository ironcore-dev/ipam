// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var networkcounterlog = logf.Log.WithName("networkcounter-resource")

func SetupNetworkCounterWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.NetworkCounter{}).
		WithValidator(&NetworkCounterCustomValidator{}).
		WithDefaulter(&NetworkCounterCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ipam-metal-ironcore-dev-v1alpha1-networkcounter,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=networkcounters,verbs=create;update,versions=v1,name=mnetworkcounter-v1alpha1.kb.io,admissionReviewVersions=v1

// NetworkCounterCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind NetworkCounter when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type NetworkCounterCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &NetworkCounterCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind NetworkCounter
func (d *NetworkCounterCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	networkCounter, ok := obj.(*v1alpha1.NetworkCounter)

	if !ok {
		return fmt.Errorf("expected an NetworkCounter object but got %T", obj)
	}
	networklog.Info("Defaulting for NetworkCounter", "name", networkCounter.GetName())

	return nil
}

// +kubebuilder:webhook:path=/validate-ipam-metal-ironcore-dev-v1alpha1-networkcounter,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=networkcounters,verbs=create;update;delete,versions=v1alpha1,name=vnetworkcounter.kb.io,admissionReviewVersions={v1,v1beta1}

// NetworkCounterCustomValidator struct is responsible for validating the NetworkCounter resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NetworkCounterCustomValidator struct {
}

var _ webhook.CustomValidator = &NetworkCounterCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCounterCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var warnings admission.Warnings

	nc, ok := obj.(*v1alpha1.NetworkCounter)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to NetworkCounter CR type"))
	}
	networkcounterlog.Info("validate create", "name", nc.GetName())
	return nil, nil

}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCounterCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	var warnings admission.Warnings

	nc, ok := oldObj.(*v1alpha1.NetworkCounter)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to NetworkCounter CR type"))
	}
	networkcounterlog.Info("validate update", "name", nc.GetName())
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCounterCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	nc, ok := obj.(*v1alpha1.NetworkCounter)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to NetworkCounter CR type"))
	}
	networkcounterlog.Info("validate delete", "name", nc.GetName())

	if len(nc.Spec.Vacant) == 0 {
		allErrs = append(allErrs, field.InternalError(
			field.NewPath("metadata.name"),
			errors.New("Network Counter is still in use by networks")))
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{
				Group: v1alpha1.SchemeGroupVersion.Group,
				Kind:  "NetworkCounter",
			}, nc.Name, allErrs)
	}

	begin := nc.Spec.Vacant[0].Begin
	end := nc.Spec.Vacant[0].End

	if end == nil && begin.Eq(v1alpha1.MPLSFirstAvailableID) {
		return warnings, nil
	}

	if begin.Eq(v1alpha1.VXLANFirstAvaliableID) && end.Eq(v1alpha1.VXLANMaxID) {
		return warnings, nil
	}

	allErrs = append(allErrs, field.InternalError(field.NewPath("metadata.name"), errors.New("Network Counter is still in use by networks")))
	return warnings, apierrors.NewInvalid(
		schema.GroupKind{
			Group: v1alpha1.SchemeGroupVersion.Group,
			Kind:  "NetworkCounter",
		}, nc.Name, allErrs)
}
