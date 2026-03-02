// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// log is for logging in this package.
var iplog = logf.Log.WithName("ip-resource")

// SetupIPWebhookWithManager sets up and registers the webhook with the manager.
func SetupIPWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &v1alpha1.IP{}).
		WithValidator(&IPCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ipam-metal-ironcore-dev-v1alpha1-ip,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=ips,verbs=create;update,versions=v1,name=mip-v1alpha1.kb.io,admissionReviewVersions=v1

// +kubebuilder:webhook:path=/validate-ipam-metal-ironcore-dev-v1alpha1-ip,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=ips,verbs=create;update;delete,versions=v1alpha1,name=vip.kb.io,admissionReviewVersions={v1,v1beta1}

// IPCustomValidator struct is responsible for validating the IP resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type IPCustomValidator struct {
	client.Client
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *IPCustomValidator) ValidateCreate(ctx context.Context, obj *v1alpha1.IP) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	iplog.Info("validate create", "name", obj.GetName())

	if obj.Spec.Consumer != nil {
		if _, err := schema.ParseGroupVersion(obj.Spec.Consumer.APIVersion); err != nil {
			allErrs = append(allErrs,
				field.Invalid(
					field.NewPath("spec.consumer.apiVersion"), obj.Spec.Consumer.APIVersion, err.Error()))
		}
	}

	if obj.Spec.Subnet.Name == "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.subnet.name"), obj.Spec.IP, "Parent subnet should be defined"))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(obj.GroupVersionKind().GroupKind(), obj.Name, allErrs)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *IPCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *v1alpha1.IP) (admission.Warnings, error) {
	var warnings admission.Warnings

	iplog.Info("validate update", "name", oldObj.GetName())

	var allErrs field.ErrorList

	if oldObj.Spec.IP != nil || newObj.Spec.IP != nil {
		if oldObj.Spec.IP == nil || newObj.Spec.IP == nil ||
			!oldObj.Spec.IP.Equal(newObj.Spec.IP) {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.ip"), newObj.Spec.IP, "IP change is disallowed"))
		}
	}

	if oldObj.Spec.Subnet.Name != newObj.Spec.Subnet.Name {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.subnet.name"), newObj.Spec.Subnet.Name, "Subnet change is disallowed"))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(newObj.GroupVersionKind().GroupKind(), newObj.Name, allErrs)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *IPCustomValidator) ValidateDelete(ctx context.Context, obj *v1alpha1.IP) (admission.Warnings, error) {
	var warnings admission.Warnings

	iplog.Info("validate delete", "name", obj.GetName())

	if obj.Spec.Consumer == nil {
		return warnings, nil
	}

	unstruct := &unstructured.Unstructured{}
	gv, err := schema.ParseGroupVersion(obj.Spec.Consumer.APIVersion)
	if err != nil {
		message := fmt.Sprintf("unable to parse APIVerson of consumer resource, therefore allowing to delete IP."+
			"name: %s, api version: %s",
			obj.Name, obj.Spec.Consumer.APIVersion)
		iplog.Error(err, message)
		return append(warnings, message), nil
	}

	gvk := gv.WithKind(obj.Spec.Consumer.Kind)
	unstruct.SetGroupVersionKind(gvk)
	namespacedName := types.NamespacedName{
		Namespace: obj.Namespace,
		Name:      obj.Spec.Consumer.Name,
	}

	err = v.Get(ctx, namespacedName, unstruct)
	if !apierrors.IsNotFound(err) {
		var allErrs field.ErrorList
		consumerUnstruct := unstruct.Object
		deletionTimestamp, _, err := unstructured.NestedString(consumerUnstruct, "metadata", "deletionTimestamp")
		switch {
		case err != nil:
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer"), obj.Spec.Consumer, err.Error()))
			return warnings, apierrors.NewInvalid(gvk.GroupKind(), obj.Name, allErrs)
		case deletionTimestamp == "":
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer"), obj.Spec.Consumer, "Consumer is not deleted"))
			return warnings, apierrors.NewInvalid(gvk.GroupKind(), obj.Name, allErrs)
		}
	}

	return warnings, nil
}
