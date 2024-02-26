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
	"context"
	"fmt"
	"reflect"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	_ "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	CFinishedChildSubnetToSubnetIndexKey = "finishedChildSubnetToSubnet"
	CFinishedChildIPToSubnetIndexKey     = "finishedChildIPToSubnet"
)

// log is for logging in this package.
var subnetlog = logf.Log.WithName("subnet-resource")
var subnetWebhookClient client.Client

func (in *Subnet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	createChildSubnetIndexValue := func(object client.Object) []string {
		subnet, ok := object.(*Subnet)
		if !ok {
			return nil
		}
		state := subnet.Status.State
		parentSubnet := subnet.Spec.ParentSubnet.Name
		if parentSubnet == "" {
			return nil
		}
		if state != CFinishedSubnetState {
			return nil
		}
		return []string{parentSubnet}
	}

	createChildIPIndexValue := func(object client.Object) []string {
		ip, ok := object.(*IP)
		if !ok {
			return nil
		}
		state := ip.Status.State
		parentSubnet := ip.Spec.Subnet.Name
		if state != CFinishedIPState {
			return nil
		}
		return []string{parentSubnet}
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &Subnet{}, CFinishedChildSubnetToSubnetIndexKey, createChildSubnetIndexValue); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &IP{}, CFinishedChildIPToSubnetIndexKey, createChildIPIndexValue); err != nil {
		return err
	}

	subnetWebhookClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// +kubebuilder:webhook:path=/validate-ipam-onmetal-de-v1alpha1-subnet,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=subnets,verbs=create;update;delete,versions=v1alpha1,name=vsubnet.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Subnet{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Subnet) ValidateCreate() (admission.Warnings, error) {
	subnetlog.Info("validate create", "name", in.Name)

	var allErrs field.ErrorList
	var warnings admission.Warnings

	rulesCount := in.countCIDRReservationRules()
	rulesPaths := []string{"spec.cidr", "spec.capacity", "spec.hostIdentifierBits"}
	minQuantity := resource.NewQuantity(1, resource.DecimalSI)
	maxQuantity, err := resource.ParseQuantity("340282366920938463463374607431768211456")
	if err != nil {
		return warnings, apierrors.NewInternalError(err)
	}

	if rulesCount == 0 || rulesCount > 1 {
		errMsg := fmt.Sprintf("value should be set for the one of the following fields: %s", strings.Join(rulesPaths, ", "))
		for _, path := range rulesPaths {
			allErrs = append(allErrs, field.Invalid(field.NewPath(path), in.Spec.CIDR, errMsg))
		}
	}

	if in.Spec.Consumer != nil {
		if _, err := schema.ParseGroupVersion(in.Spec.Consumer.APIVersion); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer.apiVersion"), in.Spec.Consumer, err.Error()))
		}
	}

	if in.Spec.ParentSubnet.Name == "" &&
		in.Spec.CIDR == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.cidr"), in.Spec.CIDR, "cidr should be set explicitly if a top level subnet (without parent subnet) is created"))
	}

	if in.Spec.Capacity != nil && maxQuantity.Cmp(*in.Spec.Capacity) < 0 &&
		minQuantity.Cmp(*in.Spec.Capacity) > 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.capacity"), in.Spec.CIDR, "if set, capacity value should be between 1 and 2^128"))
	}

	if !in.uniqueRegionSet() {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.regions"), in.Spec.Regions, "region values should be unique"))
	}

	for i, region := range in.Spec.Regions {
		if !in.uniqueAZSet(region.AvailabilityZones) {
			allErrs = append(allErrs, field.Invalid(field.NewPath(fmt.Sprintf("spec.regions[%d].availabilityZones", i)), region.AvailabilityZones, "availability zone values should be unique"))
		}
	}

	if len(allErrs) > 0 {
		gvk := in.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return warnings, apierrors.NewInvalid(gk, in.Name, allErrs)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Subnet) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	subnetlog.Info("validate update", "name", in.Name)

	var warnings admission.Warnings

	oldSubnet, ok := old.(*Subnet)
	if !ok {
		return warnings, errors.New("cannot cast previous object version to Subnet CR type")
	}

	var allErrs field.ErrorList

	if !(oldSubnet.Spec.CIDR == nil && in.Spec.CIDR == nil) {
		if oldSubnet.Spec.CIDR == nil || in.Spec.CIDR == nil ||
			!oldSubnet.Spec.CIDR.Equal(in.Spec.CIDR) {
			allErrs = append(allErrs,
				field.Invalid(
					field.NewPath("spec.cidr"), in.Spec.CIDR, "CIDR change is disallowed"))
		}
	}

	if !(oldSubnet.Spec.PrefixBits == nil && in.Spec.PrefixBits == nil) {
		if oldSubnet.Spec.PrefixBits == nil || in.Spec.PrefixBits == nil ||
			*oldSubnet.Spec.PrefixBits != *in.Spec.PrefixBits {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.hostIdentifierBits"), in.Spec.PrefixBits, "Host identifier bits change is disallowed"))
		}
	}

	if !(oldSubnet.Spec.Capacity == nil && in.Spec.Capacity == nil) {
		if oldSubnet.Spec.Capacity == nil || in.Spec.Capacity == nil ||
			!oldSubnet.Spec.Capacity.Equal(*in.Spec.Capacity) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.capacity"), in.Spec.Capacity, "Capacity change is disallowed"))
		}
	}

	if oldSubnet.Spec.ParentSubnet.Name != in.Spec.ParentSubnet.Name {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.parentSubnet.name"), in.Spec.CIDR, "Parent Subnet change is disallowed"))
	}

	if oldSubnet.Spec.Network.Name != in.Spec.Network.Name {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.network.name"), in.Spec.CIDR, "Network change is disallowed"))
	}

	if !reflect.DeepEqual(oldSubnet.Spec.Regions, in.Spec.Regions) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.regions"), in.Spec.CIDR, "Regions change is disallowed"))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{
				Group: SchemeGroupVersion.Group,
				Kind:  "Subnet",
			}, in.Name, allErrs)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Subnet) ValidateDelete() (admission.Warnings, error) {
	subnetlog.Info("validate delete", "name", in.Name)

	var allErrs field.ErrorList
	var warnings admission.Warnings

	if in.Spec.Consumer != nil {
		unstruct := &unstructured.Unstructured{}
		gv, err := schema.ParseGroupVersion(in.Spec.Consumer.APIVersion)
		if err != nil {
			message := fmt.Sprintf(
				"unable to parse APIVersion of consumer resource, therefore allowing to delete Subnet."+
					" name: %s, api version: %s", in.Name, in.Spec.Consumer.APIVersion)
			subnetlog.Error(
				err, message)
			return append(warnings, message), nil
		}

		gvk := gv.WithKind(in.Spec.Consumer.Kind)
		unstruct.SetGroupVersionKind(gvk)
		namespacedName := types.NamespacedName{
			Namespace: in.Namespace,
			Name:      in.Spec.Consumer.Name,
		}
		ctx := context.Background()

		err = subnetWebhookClient.Get(ctx, namespacedName, unstruct)
		if !apierrors.IsNotFound(err) {
			consumerUnstruct := unstruct.Object
			deletionTimestamp, _, err := unstructured.NestedString(consumerUnstruct, "metadata", "deletionTimestamp")
			switch {
			case err != nil:
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer"), in.Spec.Consumer, err.Error()))
			case deletionTimestamp == "":
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer"), in.Spec.Consumer, "Consumer is not deleted"))
			}
		}
	}

	childSubnetsMatchingFields := client.MatchingFields{
		CFinishedChildSubnetToSubnetIndexKey: in.Name,
	}

	subnets := &SubnetList{}
	if err := subnetWebhookClient.List(context.Background(), subnets, client.InNamespace(in.Namespace), childSubnetsMatchingFields, client.Limit(1)); err != nil {
		wrappedErr := errors.Wrap(err, "unable to get connected child subnets")
		subnetlog.Error(wrappedErr,
			"", "name", types.NamespacedName{Namespace: in.Namespace, Name: in.Name})
		return append(warnings, wrappedErr.Error()), wrappedErr
	}

	if len(subnets.Items) > 0 {
		allErrs = append(allErrs, field.InternalError(
			field.NewPath("metadata.name"),
			errors.New("Subnet is still in use by another subnets")))
	}

	childIPsMatchingFields := client.MatchingFields{
		CFinishedChildIPToSubnetIndexKey: in.Name,
	}

	ips := &IPList{}
	if err := subnetWebhookClient.List(context.Background(), ips, client.InNamespace(in.Namespace), childIPsMatchingFields, client.Limit(1)); err != nil {
		wrappedErr := errors.Wrap(err, "unable to get connected child ips")
		subnetlog.Error(wrappedErr, "", "name", types.NamespacedName{Namespace: in.Namespace, Name: in.Name})
		return append(warnings, wrappedErr.Error()), wrappedErr
	}

	if len(ips.Items) > 0 {
		allErrs = append(allErrs, field.InternalError(field.NewPath("metadata.name"), errors.New("Subnet is still in use by IPs")))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{
				Group: SchemeGroupVersion.Group,
				Kind:  "Subnet",
			}, in.Name, allErrs)
	}

	return warnings, nil
}

func (in *Subnet) countCIDRReservationRules() int {
	count := 0
	if in.Spec.CIDR != nil {
		count += 1
	}
	if in.Spec.Capacity != nil {
		count += 1
	}
	if in.Spec.PrefixBits != nil {
		count += 1
	}

	return count
}

func (in *Subnet) uniqueRegionSet() bool {
	regionset := make(StringSet)
	for _, item := range in.Spec.Regions {
		if err := regionset.Put(item.Name); err != nil {
			return false
		}
	}
	return true
}

func (in *Subnet) uniqueAZSet(azs []string) bool {
	azset := make(StringSet)
	for _, item := range azs {
		if err := azset.Put(item); err != nil {
			return false
		}
	}
	return true
}

type StringSet map[string]struct{}

func (s StringSet) Put(item string) error {
	_, ok := s[item]
	if ok {
		return errors.Errorf("set already has value %s", item)
	}
	s[item] = struct{}{}
	return nil
}
