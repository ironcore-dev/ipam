// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var iplog = logf.Log.WithName("ip-resource")

// SetupIPWebhookWithManager sets up and registers the webhook with the manager.
func SetupIPWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.IP{}).
		WithValidator(&IPCustomValidator{mgr.GetClient()}).
		WithDefaulter(&IPCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ipam-metal-ironcore-dev-v1alpha1-ip,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=ips,verbs=create;update,versions=v1,name=mip-v1alpha1.kb.io,admissionReviewVersions=v1

// IPCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind CronJob when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type IPCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &IPCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind IP
func (d *IPCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	ip, ok := obj.(*v1alpha1.IP)

	if !ok {
		return fmt.Errorf("expected an IP object but got %T", obj)
	}
	iplog.Info("Defaulting for IP", "name", ip.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// +kubebuilder:webhook:path=/validate-ipam-metal-ironcore-dev-v1alpha1-ip,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=ips,verbs=create;update;delete,versions=v1alpha1,name=vip.kb.io,admissionReviewVersions={v1,v1beta1}

// IPCustomValidator struct is responsible for validating the IP resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type IPCustomValidator struct {
	client.Client
}

var _ webhook.CustomValidator = &IPCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *IPCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	ip, ok := obj.(*v1alpha1.IP)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to IP CR type"))
	}
	iplog.Info("validate create", "name", ip.GetName())

	if ip.Spec.Consumer != nil {
		if _, err := schema.ParseGroupVersion(ip.Spec.Consumer.APIVersion); err != nil {
			allErrs = append(allErrs,
				field.Invalid(
					field.NewPath("spec.consumer.apiVersion"), ip.Spec.Consumer.APIVersion, err.Error()))
		}
	}

	if ip.Spec.Subnet.Name == "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.subnet.name"), ip.Spec.IP, "Parent subnet should be defined"))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(ip.GroupVersionKind().GroupKind(), ip.Name, allErrs)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *IPCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	var warnings admission.Warnings

	oldIP, ok := oldObj.(*v1alpha1.IP)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to IP CR type"))
	}
	newIP, ok := newObj.(*v1alpha1.IP)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to IP CR type"))
	}
	iplog.Info("validate update", "name", oldIP.GetName())

	var allErrs field.ErrorList

	if oldIP.Spec.IP != nil || newIP.Spec.IP != nil {
		if oldIP.Spec.IP == nil || newIP.Spec.IP == nil ||
			!oldIP.Spec.IP.Equal(newIP.Spec.IP) {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.ip"), newIP.Spec.IP, "IP change is disallowed"))
		}
	}

	if oldIP.Spec.Subnet.Name != newIP.Spec.Subnet.Name {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.subnet.name"), newIP.Spec.Subnet.Name, "Subnet change is disallowed"))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(newIP.GroupVersionKind().GroupKind(), newIP.Name, allErrs)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *IPCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var warnings admission.Warnings

	ip, ok := obj.(*v1alpha1.IP)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to IP CR type"))
	}
	iplog.Info("validate delete", "name", ip.GetName())

	if ip.Spec.Consumer == nil {
		return warnings, nil
	}

	unstruct := &unstructured.Unstructured{}
	gv, err := schema.ParseGroupVersion(ip.Spec.Consumer.APIVersion)
	if err != nil {
		message := fmt.Sprintf("unable to parse APIVerson of consumer resource, therefore allowing to delete IP."+
			"name: %s, api version: %s",
			ip.Name, ip.Spec.Consumer.APIVersion)
		iplog.Error(err, message)
		return append(warnings, message), nil
	}

	gvk := gv.WithKind(ip.Spec.Consumer.Kind)
	unstruct.SetGroupVersionKind(gvk)
	namespacedName := types.NamespacedName{
		Namespace: ip.Namespace,
		Name:      ip.Spec.Consumer.Name,
	}

	err = v.Get(ctx, namespacedName, unstruct)
	if !apierrors.IsNotFound(err) {
		var allErrs field.ErrorList
		consumerUnstruct := unstruct.Object
		deletionTimestamp, _, err := unstructured.NestedString(consumerUnstruct, "metadata", "deletionTimestamp")
		switch {
		case err != nil:
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer"), ip.Spec.Consumer, err.Error()))
			return warnings, apierrors.NewInvalid(gvk.GroupKind(), ip.Name, allErrs)
		case deletionTimestamp == "":
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer"), ip.Spec.Consumer, "Consumer is not deleted"))
			return warnings, apierrors.NewInvalid(gvk.GroupKind(), ip.Name, allErrs)
		}
	}

	return warnings, nil
}
