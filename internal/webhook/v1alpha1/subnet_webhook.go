// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"

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
	FinishedChildSubnetToSubnetIndexKey = "finishedChildSubnetToSubnet"
	FinishedChildIPToSubnetIndexKey     = "finishedChildIPToSubnet"
)

// log is for logging in this package.
var subnetlog = logf.Log.WithName("subnet-resource")

func SetupSubnetWebhookWithManager(mgr ctrl.Manager) error {
	createChildSubnetIndexValue := func(object client.Object) []string {
		subnet, ok := object.(*v1alpha1.Subnet)
		if !ok {
			return nil
		}
		state := subnet.Status.State
		parentSubnet := subnet.Spec.ParentSubnet.Name
		if parentSubnet == "" {
			return nil
		}
		if state != v1alpha1.FinishedSubnetState {
			return nil
		}
		return []string{parentSubnet}
	}

	createChildIPIndexValue := func(object client.Object) []string {
		ip, ok := object.(*v1alpha1.IP)
		if !ok {
			return nil
		}
		state := ip.Status.State
		parentSubnet := ip.Spec.Subnet.Name
		if state != v1alpha1.IPStateAllocated {
			return nil
		}
		return []string{parentSubnet}
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &v1alpha1.Subnet{}, FinishedChildSubnetToSubnetIndexKey, createChildSubnetIndexValue); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &v1alpha1.IP{}, FinishedChildIPToSubnetIndexKey, createChildIPIndexValue); err != nil {
		return err
	}

	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.Subnet{}).
		WithValidator(&SubnetCustomValidator{mgr.GetClient()}).
		WithDefaulter(&SubnetCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ipam-metal-ironcore-dev-v1alpha1-subnet,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=subnets,verbs=create;update,versions=v1,name=msubnet-v1alpha1.kb.io,admissionReviewVersions=v1

// SubnetCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Subnet when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type SubnetCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &SubnetCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Subnet
func (d *SubnetCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	subnet, ok := obj.(*v1alpha1.Subnet)

	if !ok {
		return fmt.Errorf("expected an Subnet object but got %T", obj)
	}
	iplog.Info("Defaulting for Subnet", "name", subnet.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// +kubebuilder:webhook:path=/validate-ipam-metal-ironcore-dev-v1alpha1-subnet,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.metal.ironcore.dev,resources=subnets,verbs=create;update;delete,versions=v1alpha1,name=vsubnet.kb.io,admissionReviewVersions={v1,v1beta1}

// SubnetCustomValidator struct is responsible for validating the Subnet resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type SubnetCustomValidator struct {
	client.Client
}

var _ webhook.CustomValidator = &SubnetCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *SubnetCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	subnet, ok := obj.(*v1alpha1.Subnet)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to Subnet CR type"))
	}
	subnetlog.Info("validate create", "name", subnet.GetName())

	rulesCount := countCIDRReservationRules(subnet)
	rulesPaths := []string{"spec.cidr", "spec.capacity", "spec.hostIdentifierBits"}
	minQuantity := resource.NewQuantity(1, resource.DecimalSI)
	maxQuantity, err := resource.ParseQuantity("340282366920938463463374607431768211456")
	if err != nil {
		return warnings, apierrors.NewInternalError(err)
	}

	if rulesCount == 0 || rulesCount > 1 {
		errMsg := fmt.Sprintf("value should be set for the one of the following fields: %s", strings.Join(rulesPaths, ", "))
		for _, path := range rulesPaths {
			allErrs = append(allErrs, field.Invalid(field.NewPath(path), subnet.Spec.CIDR, errMsg))
		}
	}

	if subnet.Spec.Consumer != nil {
		if _, err := schema.ParseGroupVersion(subnet.Spec.Consumer.APIVersion); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer.apiVersion"), subnet.Spec.Consumer, err.Error()))
		}
	}

	if subnet.Spec.ParentSubnet.Name == "" &&
		subnet.Spec.CIDR == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.cidr"), subnet.Spec.CIDR, "cidr should be set explicitly if a top level subnet (without parent subnet) is created"))
	}

	if subnet.Spec.Capacity != nil && maxQuantity.Cmp(*subnet.Spec.Capacity) < 0 &&
		minQuantity.Cmp(*subnet.Spec.Capacity) > 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.capacity"), subnet.Spec.CIDR, "if set, capacity value should be between 1 and 2^128"))
	}

	if !uniqueRegionSet(subnet) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.regions"), subnet.Spec.Regions, "region values should be unique"))
	}

	for i, region := range subnet.Spec.Regions {
		if !uniqueAZSet(region.AvailabilityZones) {
			allErrs = append(allErrs, field.Invalid(field.NewPath(fmt.Sprintf("spec.regions[%d].availabilityZones", i)), region.AvailabilityZones, "availability zone values should be unique"))
		}
	}

	if len(allErrs) > 0 {
		gvk := subnet.GroupVersionKind()
		gk := schema.GroupKind{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		}
		return warnings, apierrors.NewInvalid(gk, subnet.Name, allErrs)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *SubnetCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	var warnings admission.Warnings
	var allErrs field.ErrorList

	oldSubnet, ok := oldObj.(*v1alpha1.Subnet)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to Subnet CR type"))
	}
	newSubnet, ok := newObj.(*v1alpha1.Subnet)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to Subnet CR type"))
	}

	subnetlog.Info("validate update", "name", oldSubnet.Name)

	if !(oldSubnet.Spec.CIDR == nil && newSubnet.Spec.CIDR == nil) {
		if oldSubnet.Spec.CIDR == nil || newSubnet.Spec.CIDR == nil ||
			!oldSubnet.Spec.CIDR.Equal(newSubnet.Spec.CIDR) {
			allErrs = append(allErrs,
				field.Invalid(
					field.NewPath("spec.cidr"), newSubnet.Spec.CIDR, "CIDR change is disallowed"))
		}
	}

	if !(oldSubnet.Spec.PrefixBits == nil && newSubnet.Spec.PrefixBits == nil) {
		if oldSubnet.Spec.PrefixBits == nil || newSubnet.Spec.PrefixBits == nil ||
			*oldSubnet.Spec.PrefixBits != *newSubnet.Spec.PrefixBits {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.hostIdentifierBits"), newSubnet.Spec.PrefixBits, "Host identifier bits change is disallowed"))
		}
	}

	if !(oldSubnet.Spec.Capacity == nil && newSubnet.Spec.Capacity == nil) {
		if oldSubnet.Spec.Capacity == nil || newSubnet.Spec.Capacity == nil ||
			!oldSubnet.Spec.Capacity.Equal(*newSubnet.Spec.Capacity) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec.capacity"), newSubnet.Spec.Capacity, "Capacity change is disallowed"))
		}
	}

	if oldSubnet.Spec.ParentSubnet.Name != newSubnet.Spec.ParentSubnet.Name {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.parentSubnet.name"), newSubnet.Spec.CIDR, "Parent Subnet change is disallowed"))
	}

	if oldSubnet.Spec.Network.Name != newSubnet.Spec.Network.Name {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.network.name"), newSubnet.Spec.CIDR, "Network change is disallowed"))
	}

	if !reflect.DeepEqual(oldSubnet.Spec.Regions, newSubnet.Spec.Regions) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.regions"), newSubnet.Spec.CIDR, "Regions change is disallowed"))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{
				Group: v1alpha1.SchemeGroupVersion.Group,
				Kind:  "Subnet",
			}, newSubnet.Name, allErrs)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *SubnetCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	subnet, ok := obj.(*v1alpha1.Subnet)
	if !ok {
		return warnings, apierrors.NewInternalError(
			errors.New("cannot cast previous object version to Subnet CR type"))
	}

	subnetlog.Info("validate delete", "name", subnet.Name)

	if subnet.Spec.Consumer != nil {
		unstruct := &unstructured.Unstructured{}
		gv, err := schema.ParseGroupVersion(subnet.Spec.Consumer.APIVersion)
		if err != nil {
			message := fmt.Sprintf(
				"unable to parse APIVersion of consumer resource, therefore allowing to delete Subnet."+
					" name: %s, api version: %s", subnet.Name, subnet.Spec.Consumer.APIVersion)
			subnetlog.Error(
				err, message)
			return append(warnings, message), nil
		}

		gvk := gv.WithKind(subnet.Spec.Consumer.Kind)
		unstruct.SetGroupVersionKind(gvk)
		namespacedName := types.NamespacedName{
			Namespace: subnet.Namespace,
			Name:      subnet.Spec.Consumer.Name,
		}
		ctx := context.Background()

		err = v.Get(ctx, namespacedName, unstruct)
		if !apierrors.IsNotFound(err) {
			consumerUnstruct := unstruct.Object
			deletionTimestamp, _, err := unstructured.NestedString(consumerUnstruct, "metadata", "deletionTimestamp")
			switch {
			case err != nil:
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer"), subnet.Spec.Consumer, err.Error()))
			case deletionTimestamp == "":
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec.consumer"), subnet.Spec.Consumer, "Consumer is not deleted"))
			}
		}
	}

	childSubnetsMatchingFields := client.MatchingFields{
		FinishedChildSubnetToSubnetIndexKey: subnet.Name,
	}

	subnets := &v1alpha1.SubnetList{}
	if err := v.List(context.Background(), subnets, client.InNamespace(subnet.Namespace), childSubnetsMatchingFields, client.Limit(1)); err != nil {
		wrappedErr := errors.Wrap(err, "unable to get connected child subnets")
		subnetlog.Error(wrappedErr,
			"", "name", types.NamespacedName{Namespace: subnet.Namespace, Name: subnet.Name})
		return append(warnings, wrappedErr.Error()), wrappedErr
	}

	if len(subnets.Items) > 0 {
		allErrs = append(allErrs, field.InternalError(
			field.NewPath("metadata.name"),
			errors.New("Subnet is still in use by another subnets")))
	}

	childIPsMatchingFields := client.MatchingFields{
		FinishedChildIPToSubnetIndexKey: subnet.Name,
	}

	ips := &v1alpha1.IPList{}
	if err := v.List(context.Background(), ips, client.InNamespace(subnet.Namespace), childIPsMatchingFields, client.Limit(1)); err != nil {
		wrappedErr := errors.Wrap(err, "unable to get connected child ips")
		subnetlog.Error(wrappedErr, "", "name", types.NamespacedName{Namespace: subnet.Namespace, Name: subnet.Name})
		return append(warnings, wrappedErr.Error()), wrappedErr
	}

	if len(ips.Items) > 0 {
		allErrs = append(allErrs, field.InternalError(field.NewPath("metadata.name"), errors.New("Subnet is still in use by IPs")))
	}

	if len(allErrs) > 0 {
		return warnings, apierrors.NewInvalid(
			schema.GroupKind{
				Group: v1alpha1.SchemeGroupVersion.Group,
				Kind:  "Subnet",
			}, subnet.Name, allErrs)
	}

	return warnings, nil
}

func countCIDRReservationRules(in *v1alpha1.Subnet) int {
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

func uniqueRegionSet(in *v1alpha1.Subnet) bool {
	regionset := make(StringSet)
	for _, item := range in.Spec.Regions {
		if err := regionset.Put(item.Name); err != nil {
			return false
		}
	}
	return true
}

func uniqueAZSet(azs []string) bool {
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
