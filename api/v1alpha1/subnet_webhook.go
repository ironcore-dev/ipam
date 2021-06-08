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
	"reflect"

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
var subnetlog = logf.Log.WithName("subnet-resource")

func (r *Subnet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-ipam-onmetal-de-v1alpha1-subnet,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=subnets,verbs=create;update,versions=v1alpha1,name=vsubnet.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Subnet{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Subnet) ValidateCreate() error {
	subnetlog.Info("validate create", "name", r.Name)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Subnet) ValidateUpdate(old runtime.Object) error {
	subnetlog.Info("validate update", "name", r.Name)

	oldSubnet, ok := old.(*Subnet)
	if !ok {
		return errors.New("cannot cast previous object version to Subnet CR type")
	}

	var allErrs field.ErrorList

	if !oldSubnet.Spec.CIDR.Equal(&r.Spec.CIDR) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.cidr"), r.Spec.CIDR, "CIDR change is disallowed"))
	}

	if oldSubnet.Spec.ParentSubnetName != r.Spec.ParentSubnetName {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.parentSubnetName"), r.Spec.CIDR, "Parent Subnet change is disallowed"))
	}

	if oldSubnet.Spec.NetworkName != r.Spec.NetworkName {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.networkName"), r.Spec.CIDR, "Network change is disallowed"))
	}

	if !reflect.DeepEqual(oldSubnet.Spec.Regions, r.Spec.Regions) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.regions"), r.Spec.CIDR, "Regions change is disallowed"))
	}

	if !reflect.DeepEqual(oldSubnet.Spec.AvailabilityZones, r.Spec.AvailabilityZones) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.availabilityZones"), r.Spec.CIDR, "Availability zones change is disallowed"))
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "Subnet",
			}, r.Name, allErrs)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Subnet) ValidateDelete() error {
	subnetlog.Info("validate delete", "name", r.Name)
	return nil
}
