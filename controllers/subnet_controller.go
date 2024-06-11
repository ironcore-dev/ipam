// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
)

const (
	CSubnetFinalizer = "subnet.ipam.metal.ironcore.dev/finalizer"

	CSubnetFinalizationSuccessReason = "SubnetFinalizationSuccess"

	CTopSubnetReservationFailureReason = "TopSubnetReservationFailure"
	CTopSubnetReservationSuccessReason = "TopSubnetReservationSuccess"
	CTopSubnetReleaseSuccessReason     = "TopSubnetReleaseSuccess"

	CChildSubnetAZScopeFailureReason      = "ChildSubnetAZScopeFailure"
	CChildSubnetRegionScopeFailureReason  = "ChildSubnetRegionScopeFailure"
	CChildSubnetCIDRProposalFailureReason = "ChildSubnetCIDRProposalFailure"
	CChildSubnetReservationFailureReason  = "ChildSubnetReservationFailure"
	CChildSubnetReservationSuccessReason  = "ChildSubnetReservationSuccess"
	CChildSubnetReleaseSuccessReason      = "ChildSubnetReleaseSuccess"

	CFailedChildSubnetIndexKey = "failedChildSubnet"
	CFailedIPIndexKey          = "failedIP"
)

// SubnetReconciler reconciles a Subnet object
type SubnetReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=ipam.metal.ironcore.dev,resources=subnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.metal.ironcore.dev,resources=subnets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.metal.ironcore.dev,resources=subnets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *SubnetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("subnet", req.NamespacedName)

	// Get subnet
	// If resource is not found, then most likely it was deleted
	subnet := &v1alpha1.Subnet{}
	err := r.Get(ctx, req.NamespacedName, subnet)
	if apierrors.IsNotFound(err) {
		log.Info("Resource not found, it might have been deleted.")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err != nil {
		log.Error(err, "unable to get subnet resource", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// If deletion timestamp is present,
	// then resource is scheduled for deletion.
	if subnet.GetDeletionTimestamp() != nil {
		// If finalizer is set, then finalizer should be called to release
		// resources.
		if controllerutil.ContainsFinalizer(subnet, CSubnetFinalizer) {
			if err := r.finalizeSubnet(ctx, log, req.NamespacedName, subnet); err != nil {
				log.Error(err, "unable to finalize subnet resource", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(subnet, CSubnetFinalizer)
			err := r.Update(ctx, subnet)
			if err != nil {
				log.Error(err, "unable to update subnet resource on finalizer removal", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			r.EventRecorder.Event(subnet, v1.EventTypeNormal, CSubnetFinalizationSuccessReason, "Subnet deleted")
		}
		return ctrl.Result{}, nil
	}

	// If finalizer is not set, then resource should be updated with finalizer.
	if !controllerutil.ContainsFinalizer(subnet, CSubnetFinalizer) {
		controllerutil.AddFinalizer(subnet, CSubnetFinalizer)
		err = r.Update(ctx, subnet)
		if err != nil {
			log.Error(err, "unable to update subnet resource with finalizer", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If state is empty, then subnet just have been created
	// and subnet status should be populated.
	if subnet.Status.State == "" {
		subnet.PopulateStatus()
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If state is "Finished" or "Finalized", then
	// resource processing has been completed.
	if subnet.Status.State == v1alpha1.CFailedSubnetState ||
		subnet.Status.State == v1alpha1.CFinishedSubnetState {
		if err := r.requeueFailedSubnets(ctx, log, subnet); err != nil {
			log.Error(err, "unable to requeue child subnets", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		if err := r.requeueFailedIPs(ctx, log, subnet); err != nil {
			log.Error(err, "unable to requeue child ips", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If parent subnet is not set, then CIDR should be reserved in
	// network resource.
	if subnet.Spec.ParentSubnet.Name == "" {
		networkNamespacedName := types.NamespacedName{
			Namespace: subnet.Namespace,
			Name:      subnet.Spec.Network.Name,
		}

		network := &v1alpha1.Network{}

		if err := r.Get(ctx, networkNamespacedName, network); err != nil {
			log.Error(err, "unable to get network", "name", req.NamespacedName, "network name", networkNamespacedName)
			return ctrl.Result{}, err
		}

		// If it is not possible to reserve subnet's CIDR in network,
		// then CIDR (or its part) is already reserved,
		// and CIDR allocation has failed.
		if err := network.Reserve(subnet.Spec.CIDR); err != nil {
			log.Error(err, "unable to reserve subnet in network", "name", req.NamespacedName, "network name", networkNamespacedName)
			subnet.Status.State = v1alpha1.CFailedSubnetState
			subnet.Status.Message = err.Error()
			if err := r.Status().Update(ctx, subnet); err != nil {
				log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			r.EventRecorder.Event(subnet, v1.EventTypeWarning, CTopSubnetReservationFailureReason, subnet.Status.Message)
			return ctrl.Result{}, err
		}

		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network", "name", req.NamespacedName, "network name", networkNamespacedName)
			return ctrl.Result{}, err
		}

		subnet.FillStatusFromCidr(subnet.Spec.CIDR)
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		r.EventRecorder.Eventf(subnet, v1.EventTypeNormal, CTopSubnetReservationSuccessReason, "CIDR %s in network %s reserved successfully", subnet.Status.Reserved.String(), network.Name)

		return ctrl.Result{}, nil
	}

	// If parent subnet is set, then current subnet's CIDR
	// should be registered in parent subnet.
	parentSubnetNamespacedName := types.NamespacedName{
		Namespace: subnet.Namespace,
		Name:      subnet.Spec.ParentSubnet.Name,
	}

	parentSubnet := &v1alpha1.Subnet{}

	if err := r.Get(ctx, parentSubnetNamespacedName, parentSubnet); err != nil {
		log.Error(err, "unable to get parent subnet", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		return ctrl.Result{}, err
	}

	if err := regionSubset(parentSubnet.Spec.Regions, subnet.Spec.Regions); err != nil {
		err := errors.Wrap(err, "subnet's region set is not a part of parent region set")
		log.Error(err, "unable to use provided region set", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		subnet.Status.State = v1alpha1.CFailedSubnetState
		subnet.Status.Message = err.Error()
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		r.EventRecorder.Event(subnet, v1.EventTypeWarning, CChildSubnetRegionScopeFailureReason, subnet.Status.Message)
		return ctrl.Result{}, err
	}

	if err := azSubset(parentSubnet.Spec.Regions, subnet.Spec.Regions); err != nil {
		err := errors.Wrap(err, "subnet's az set is not a part of parent az set")
		log.Error(err, "unable to use provided az set", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		subnet.Status.State = v1alpha1.CFailedSubnetState
		subnet.Status.Message = err.Error()
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		r.EventRecorder.Event(subnet, v1.EventTypeWarning, CChildSubnetAZScopeFailureReason, subnet.Status.Message)
		return ctrl.Result{}, err
	}

	var cidrToReserve *v1alpha1.CIDR
	if subnet.Spec.CIDR != nil {
		cidrToReserve = subnet.Spec.CIDR
	} else if subnet.Spec.PrefixBits != nil {
		cidrToReserve, err = parentSubnet.ProposeForBits(*subnet.Spec.PrefixBits)
	} else {
		cidrToReserve, err = parentSubnet.ProposeForCapacity(subnet.Spec.Capacity)
	}

	if err != nil {
		log.Error(err, "unable to find cidr that will fit in parent subnet", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		subnet.Status.State = v1alpha1.CFailedSubnetState
		subnet.Status.Message = err.Error()
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		r.EventRecorder.Event(subnet, v1.EventTypeWarning, CChildSubnetCIDRProposalFailureReason, subnet.Status.Message)
		return ctrl.Result{}, err
	}

	// If it is not possible to reserve subnet's CIDR in parent subnet,
	// then CIDR (or its part) is already reserved, and CIDR allocation has failed.
	if err := parentSubnet.Reserve(cidrToReserve); err != nil {
		log.Error(err, "unable to reserve cidr in parent subnet", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		subnet.Status.State = v1alpha1.CFailedSubnetState
		subnet.Status.Message = err.Error()
		if err := r.Status().Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		r.EventRecorder.Event(subnet, v1.EventTypeWarning, CChildSubnetReservationFailureReason, subnet.Status.Message)
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, parentSubnet); err != nil {
		log.Error(err, "unable to update parent subnet status after cidr reservation", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		return ctrl.Result{}, err
	}

	subnet.FillStatusFromCidr(cidrToReserve)
	if err := r.Status().Update(ctx, subnet); err != nil {
		log.Error(err, "unable to update parent subnet status after cidr reservation", "name", req.NamespacedName, "parent name", parentSubnetNamespacedName)
		return ctrl.Result{}, err
	}
	r.EventRecorder.Eventf(subnet, v1.EventTypeNormal, CChildSubnetReservationSuccessReason, "CIDR %s in subnet %s reserved successfully", subnet.Status.Reserved.String(), parentSubnet.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubnetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	createFailedSubnetIndexValue := func(object client.Object) []string {
		subnet, ok := object.(*v1alpha1.Subnet)
		if !ok {
			return nil
		}
		state := subnet.Status.State
		parentSubnet := subnet.Spec.ParentSubnet.Name
		if parentSubnet == "" {
			return nil
		}
		if state != v1alpha1.CFailedSubnetState {
			return nil
		}
		return []string{parentSubnet}
	}

	createFailedIPIndexValue := func(object client.Object) []string {
		ip, ok := object.(*v1alpha1.IP)
		if !ok {
			return nil
		}
		state := ip.Status.State
		parentSubnet := ip.Spec.Subnet.Name
		if parentSubnet == "" {
			return nil
		}
		if state != v1alpha1.CFailedIPState {
			return nil
		}
		return []string{parentSubnet}
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &v1alpha1.Subnet{}, CFailedChildSubnetIndexKey, createFailedSubnetIndexValue); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &v1alpha1.IP{}, CFailedIPIndexKey, createFailedIPIndexValue); err != nil {
		return err
	}

	r.EventRecorder = mgr.GetEventRecorderFor("subnet-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Subnet{}).
		Complete(r)
}

// finalizeSubnet releases subnet CIDR from parent subnet of network.
func (r *SubnetReconciler) finalizeSubnet(ctx context.Context, log logr.Logger, namespacedName types.NamespacedName, subnet *v1alpha1.Subnet) error {
	// If subnet has failed to reserve the CIDR
	// it may be released
	if subnet.Status.Reserved == nil {
		log.Info("subnet hasn't been booked, releasing", "name", namespacedName)
		return nil
	}

	// If subnet has no parent subnet, then it should be released from network.
	// Otherwise, release from parent subnet
	if subnet.Spec.ParentSubnet.Name == "" {
		networkNamespacedName := types.NamespacedName{
			Namespace: subnet.Namespace,
			Name:      subnet.Spec.Network.Name,
		}

		network := &v1alpha1.Network{}

		// If parent entity is not found, then it is possible to release CIDR.
		if err := r.Get(ctx, networkNamespacedName, network); err != nil {
			log.Error(err, "unable to get network", "name", namespacedName, "network name", networkNamespacedName)
			if apierrors.IsNotFound(err) {
				log.Error(err, "network not found, going to complete finalizer", "name", namespacedName, "network name", networkNamespacedName)
				return nil
			}
			return err
		}

		// If release fails and it is possible to reserve the same CIDR,
		// then it can be considered as already released by 3rd party.
		if err := network.Release(subnet.Status.Reserved); err != nil {
			log.Error(err, "unable to release subnet in network", "name", namespacedName, "network name", networkNamespacedName)
			if network.CanReserve(subnet.Status.Reserved) {
				log.Error(err, "seems that CIDR was released beforehand", "name", namespacedName, "network name", networkNamespacedName)
				return nil
			}
			return err
		}

		if err := r.Status().Update(ctx, network); err != nil {
			log.Error(err, "unable to update network", "name", namespacedName, "network name", networkNamespacedName)
			return err
		}
		r.EventRecorder.Eventf(subnet, v1.EventTypeNormal, CTopSubnetReleaseSuccessReason, "CIDR %s in network %s released successfully", subnet.Status.Reserved.String(), network.Name)
	} else {
		parentSubnetNamespacedName := types.NamespacedName{
			Namespace: subnet.Namespace,
			Name:      subnet.Spec.ParentSubnet.Name,
		}

		parentSubnet := &v1alpha1.Subnet{}

		if err := r.Get(ctx, parentSubnetNamespacedName, parentSubnet); err != nil {
			log.Error(err, "unable to get parent subnet", "name", namespacedName, "parent name", parentSubnetNamespacedName)
			if apierrors.IsNotFound(err) {
				log.Error(err, "parent subnet not found, going to complete finalize", "name", namespacedName, "parent name", parentSubnetNamespacedName)
				return nil
			}
			return err
		}

		if err := parentSubnet.Release(subnet.Status.Reserved); err != nil {
			log.Error(err, "unable to release cidr in parent subnet", "name", namespacedName, "parent name", parentSubnetNamespacedName)
			if parentSubnet.CanReserve(subnet.Status.Reserved) {
				log.Error(err, "seems that CIDR was released beforehand", "name", namespacedName, "parent name", parentSubnetNamespacedName)
				return nil
			}
			return err
		}

		if err := r.Status().Update(ctx, parentSubnet); err != nil {
			log.Error(err, "unable to update parent subnet status after cidr reservation", "name", namespacedName, "parent name", parentSubnetNamespacedName)
			return err
		}
		r.EventRecorder.Eventf(subnet, v1.EventTypeNormal, CChildSubnetReleaseSuccessReason, "CIDR %s in subnet %s released successfully", subnet.Status.Reserved.String(), parentSubnet.Name)
	}

	return nil
}

func (r *SubnetReconciler) requeueFailedSubnets(ctx context.Context, log logr.Logger, subnet *v1alpha1.Subnet) error {
	matchingFields := client.MatchingFields{
		CFailedChildSubnetIndexKey: subnet.Name,
	}

	subnets := &v1alpha1.SubnetList{}
	if err := r.List(context.Background(), subnets, client.InNamespace(subnet.Namespace), matchingFields); err != nil {
		log.Error(err, "unable to get connected child subnets", "name", types.NamespacedName{Namespace: subnet.Namespace, Name: subnet.Name})
		return err
	}

	for _, subnet := range subnets.Items {
		subnet.Status.State = v1alpha1.CProcessingSubnetState
		subnet.Status.Message = ""
		if err := r.Status().Update(ctx, &subnet); err != nil {
			log.Error(err, "unable to update child subnet", "name", types.NamespacedName{Namespace: subnet.Namespace, Name: subnet.Name}, "subnet", subnet.Name)
			return err
		}
	}

	return nil
}

func (r *SubnetReconciler) requeueFailedIPs(ctx context.Context, log logr.Logger, subnet *v1alpha1.Subnet) error {
	matchingFields := client.MatchingFields{
		CFailedIPIndexKey: subnet.Name,
	}

	ips := &v1alpha1.IPList{}
	if err := r.List(context.Background(), ips, client.InNamespace(subnet.Namespace), matchingFields); err != nil {
		log.Error(err, "unable to get connected ips", "name", types.NamespacedName{Namespace: subnet.Namespace, Name: subnet.Name})
		return err
	}

	for _, ip := range ips.Items {
		ip.Status.State = v1alpha1.CProcessingIPState
		ip.Status.Message = ""
		if err := r.Status().Update(ctx, &ip); err != nil {
			log.Error(err, "unable to update child ips", "name", types.NamespacedName{Namespace: subnet.Namespace, Name: subnet.Name}, "subnet", subnet.Name)
			return err
		}
	}

	return nil
}

func regionSubset(set []v1alpha1.Region, subset []v1alpha1.Region) error {
	nameSet := make([]string, len(set))
	for i := range set {
		nameSet[i] = set[i].Name
	}
	nameSubset := make([]string, len(subset))
	for i := range subset {
		nameSubset[i] = subset[i].Name
	}
	return isSubset(nameSet, nameSubset)
}

func azSubset(set []v1alpha1.Region, subset []v1alpha1.Region) error {
	setIndices := make(map[string]int)

	for i := range set {
		setIndices[set[i].Name] = i
	}

	for i := range subset {
		setIdx, ok := setIndices[subset[i].Name]
		if !ok {
			return errors.Errorf("parent region set does not have region %s", subset[i].Name)
		}

		if err := isSubset(set[setIdx].AvailabilityZones, subset[i].AvailabilityZones); err != nil {
			return errors.Wrapf(err, "az list of %s is not a subset of %s az list", subset[i].Name, set[setIdx].Name)
		}
	}

	return nil
}

// Returns error if b is not a subset of a
func isSubset(set []string, subset []string) error {
	setmap := make(map[string]bool)

	for _, val := range set {
		if _, ok := setmap[val]; ok {
			return errors.Errorf("parent set contains duplicate value %s", val)
		}
		setmap[val] = false
	}

	for _, val := range subset {
		checked, ok := setmap[val]
		if !ok {
			return errors.Errorf("parent set does not contain value %s", val)
		}
		if checked {
			return errors.Errorf("child subset contains duplicate value %s", val)
		}
		setmap[val] = true
	}

	return nil
}
