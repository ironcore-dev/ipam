// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var networkcounterlog = logf.Log.WithName("networkcounter-resource")

func SetupNetworkCounterWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &v1alpha1.NetworkCounter{}).
		WithValidator(&NetworkCounterCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ipam-metal-ironcore-dev-v1alpha1-networkcounter,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=networkcounters,verbs=create;update,versions=v1,name=mnetworkcounter-v1alpha1.kb.io,admissionReviewVersions=v1

// +kubebuilder:webhook:path=/validate-ipam-metal-ironcore-dev-v1alpha1-networkcounter,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=networkcounters,verbs=create;update;delete,versions=v1alpha1,name=vnetworkcounter.kb.io,admissionReviewVersions={v1,v1beta1}

// NetworkCounterCustomValidator struct is responsible for validating the NetworkCounter resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NetworkCounterCustomValidator struct {
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCounterCustomValidator) ValidateCreate(ctx context.Context, obj *v1alpha1.NetworkCounter) (admission.Warnings, error) {
	networkcounterlog.Info("validate create", "name", obj.GetName())
	return nil, nil

}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCounterCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *v1alpha1.NetworkCounter) (admission.Warnings, error) {
	networkcounterlog.Info("validate update", "name", oldObj.GetName())
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *NetworkCounterCustomValidator) ValidateDelete(ctx context.Context, obj *v1alpha1.NetworkCounter) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	networkcounterlog.Info("validate delete", "name", obj.GetName())

	if len(obj.Spec.Vacant) == 0 {
		allErrs = append(allErrs, field.InternalError(
			field.NewPath("metadata.name"),
			errors.New("Network Counter is still in use by networks")))
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{
				Group: v1alpha1.SchemeGroupVersion.Group,
				Kind:  "NetworkCounter",
			}, obj.Name, allErrs)
	}

	begin := obj.Spec.Vacant[0].Begin
	end := obj.Spec.Vacant[0].End

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
		}, obj.Name, allErrs)
}
