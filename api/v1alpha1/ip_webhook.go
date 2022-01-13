/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"

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
var iplog = logf.Log.WithName("ip-resource")

func (in *IP) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// +kubebuilder:webhook:path=/validate-ipam-onmetal-de-v1alpha1-ip,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=ips,verbs=create;update,versions=v1alpha1,name=vip.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &IP{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *IP) ValidateCreate() error {
	iplog.Info("validate create", "name", in.Name)

	fmt.Println("in.Spec.Subnet.Name")
	fmt.Println(in.Spec.Subnet.Name)
	fmt.Println(in.Spec.Subnet.Name == "")
	if in.Spec.Subnet.Name == "" {
		gvk := in.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		var allErrs field.ErrorList
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.subnet.name"), in.Spec.IP, "Parent subnet should be defined"))
		return apierrors.NewInvalid(gk, in.Name, allErrs)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *IP) ValidateUpdate(old runtime.Object) error {
	iplog.Info("validate update", "name", in.Name)

	oldIP, ok := old.(*IP)
	if !ok {
		return apierrors.NewInternalError(errors.New("cannot cast previous object version to IP CR type"))
	}

	var allErrs field.ErrorList

	if !(oldIP.Spec.IP == nil && in.Spec.IP == nil) {
		if oldIP.Spec.IP == nil || in.Spec.IP == nil ||
			!oldIP.Spec.IP.Equal(in.Spec.IP) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.ip"), in.Spec.IP, "IP change is disallowed"))
		}
	}

	if oldIP.Spec.Subnet.Name != in.Spec.Subnet.Name {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.subnet.name"), in.Spec.Subnet.Name, "Subnet change is disallowed"))
	}

	if len(allErrs) > 0 {
		gvk := in.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return apierrors.NewInvalid(gk, in.Name, allErrs)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *IP) ValidateDelete() error {
	iplog.Info("validate delete", "name", in.Name)
	return nil
}
