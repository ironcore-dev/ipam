// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

func (in *Network) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// +kubebuilder:webhook:path=/validate-ipam-onmetal-de-v1alpha1-network,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=networks,verbs=create;update;delete,versions=v1alpha1,name=vnetwork.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Network{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Network) ValidateCreate() error {
	networklog.Info("validate create", "name", in.Name)

	var allErrs field.ErrorList

	if in.Spec.Type == "" && in.Spec.ID != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.id"), in.Spec.ID, "setting network ID without type is disallowed"))
	}

	if err := in.validateID(); err != nil {
		allErrs = append(allErrs, err)
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

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Network) ValidateUpdate(old runtime.Object) error {
	networklog.Info("validate update", "name", in.Name)
	oldNetwork, ok := old.(*Network)
	if !ok {
		return apierrors.NewInternalError(errors.New("cannot cast previous object version to Network CR type"))
	}

	var allErrs field.ErrorList

	if oldNetwork.Spec.Type != "" &&
		oldNetwork.Spec.Type != in.Spec.Type {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.type"), in.Spec.Type, "network type change is disallowed; resource should be released (deleted) first"))
	}

	if (oldNetwork.Spec.ID != nil && oldNetwork.Spec.ID.Cmp(&in.Spec.ID.Int) != 0) ||
		(oldNetwork.Spec.ID == nil && oldNetwork.Spec.Type != "" && in.Spec.ID != nil) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.id"), in.Spec.ID, "network ID change after assignment is disallowed; resource should be released (deleted) first"))
	}

	if err := in.validateID(); err != nil {
		allErrs = append(allErrs, err)
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
func (in *Network) ValidateDelete() error {
	networklog.Info("validate delete", "name", in.Name)

	var allErrs field.ErrorList

	if len(in.Status.IPv4Ranges) > 0 {
		allErrs = append(allErrs, field.InternalError(field.NewPath("metadata.name"), errors.New("Network has active IPv4 subnets")))
	}

	if len(in.Status.IPv6Ranges) > 0 {
		allErrs = append(allErrs, field.InternalError(field.NewPath("metadata.name"), errors.New("Network has active IPv6 subnets")))
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "Network",
			}, in.Name, allErrs)
	}

	return nil
}

func (in *Network) validateID() *field.Error {
	if in.Spec.ID == nil {
		return nil
	}

	switch in.Spec.Type {
	case CVXLANNetworkType:
		if in.Spec.ID.Cmp(&CVXLANFirstAvaliableID.Int) < 0 ||
			in.Spec.ID.Cmp(&CVXLANMaxID.Int) > 0 {
			return field.Invalid(field.NewPath("spec.id"), in.Spec.ID, fmt.Sprintf("value for the ID for network type %s should be in interval [%s; %s]", in.Spec.Type, CVXLANFirstAvaliableID, CVXLANMaxID))
		}
	case CGENEVENetworkType:
		if in.Spec.ID.Cmp(&CGENEVEFirstAvaliableID.Int) < 0 ||
			in.Spec.ID.Cmp(&CGENEVEMaxID.Int) > 0 {
			return field.Invalid(field.NewPath("spec.id"), in.Spec.ID, fmt.Sprintf("value for the ID for network type %s should be in interval [%s; %s]", in.Spec.Type, CGENEVEFirstAvaliableID, CGENEVEMaxID))
		}
	case CMPLSNetworkType:
		if in.Spec.ID.Cmp(&CMPLSFirstAvailableID.Int) < 0 {
			return field.Invalid(field.NewPath("spec.id"), in.Spec.ID, fmt.Sprintf("value for the ID for network type %s should be in interval [%s; %f]", in.Spec.Type, CMPLSFirstAvailableID, math.Inf(1)))
		}
	default:
		return field.Invalid(field.NewPath("spec.type"), in.Spec.Type, "unknown network type")
	}

	return nil
}
