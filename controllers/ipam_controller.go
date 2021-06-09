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

package controllers

import (
	"context"
	"fmt"
	"github.com/onmetal/ipam/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IpamFinalizer = "ipam.ipam.onmetal.de/finalizer"
)

// IpamReconciler reconciles a Ipam object
type IpamReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ipam.onmetal.de,resources=ipams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ipam.onmetal.de,resources=ipams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ipam.onmetal.de,resources=ipams/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *IpamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ipam", req.NamespacedName)

	ipam := &v1alpha1.Ipam{}
	err := r.Get(ctx, req.NamespacedName, ipam)
	if apierrors.IsNotFound(err) {
		log.Error(err, "requested ipam resource not found", "name", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err != nil {
		log.Error(err, "unable to get ipam resource", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if ipam.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(ipam, IpamFinalizer) {
			// Free IP on resource deletion
			if err := r.finalizeIpam(ctx, ipam); err != nil {
				log.Error(err, "unable to finalize ipam resource", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(ipam, IpamFinalizer)
			err := r.Update(ctx, ipam)
			if err != nil {
				log.Error(err, "unable to update ipam resource on finalizer removal", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	if !controllerutil.ContainsFinalizer(ipam, IpamFinalizer) {
		controllerutil.AddFinalizer(ipam, IpamFinalizer)
		err = r.Update(ctx, ipam)
		if err != nil {
			log.Error(err, "unable to update ipam resource with finalizer", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Resource created
	if ipam.Status.LastUsedIP == "" {
		subnet, err := r.findSubnet(ctx, ipam)
		if err != nil {
			log.Error(err, "unable to find ipam subnet", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		newCidr, err := r.getIpAsCidr(ipam.Spec.IP)
		if err != nil {
			log.Error(err, "unable to get ip as cidr", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// Occupy new IP
		err = subnet.Reserve(newCidr)
		if err != nil {
			log.Error(err, "unable to reserve IP", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		if err := r.Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet state", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		ipam.Status.LastUsedIP = ipam.Spec.IP
		if err := r.Update(ctx, ipam); err != nil {
			log.Error(err, "unable to update ipam state", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// Resource was updated - e.g. IP changed
	} else if ipam.Status.LastUsedIP != ipam.Spec.IP {
		// Free old IP
		subnet, err := r.findSubnet(ctx, ipam)
		if err != nil {
			log.Error(err, "unable to find ipam subnet", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		lastCidr, err := r.getIpAsCidr(ipam.Status.LastUsedIP)
		if err != nil {
			log.Error(err, "unable to get ip as cidr", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		err = subnet.Release(lastCidr)
		if err != nil {
			log.Error(err, "unable to release IP", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// Occupy new IP
		newCidr, err := r.getIpAsCidr(ipam.Spec.IP)
		if err != nil {
			log.Error(err, "unable to get ip as cidr", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		err = subnet.Reserve(newCidr)
		if err != nil {
			log.Error(err, "unable to reserve IP", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		if err := r.Update(ctx, subnet); err != nil {
			log.Error(err, "unable to update subnet state", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		ipam.Status.LastUsedIP = ipam.Spec.IP
		if err := r.Update(ctx, ipam); err != nil {
			log.Error(err, "unable to update ipam state", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *IpamReconciler) finalizeIpam(ctx context.Context, ipam *v1alpha1.Ipam) error {
	// Free subnet IP
	subnet, err := r.findSubnet(ctx, ipam)
	if err != nil {
		return fmt.Errorf("unable to find ipam subnet: %w", err)
	}
	ipCidr, err := r.getIpAsCidr(ipam.Spec.IP)
	if err != nil {
		return fmt.Errorf("unable to get ip as cidr: %w", err)
	}
	err = subnet.Release(ipCidr)
	if err != nil {
		return fmt.Errorf("unable to release IP: %w", err)
	}
	if err := r.Update(ctx, subnet); err != nil {
		return fmt.Errorf("\"unable to update subnet state: %w", err)
	}
	return nil
}

func (r *IpamReconciler) getIpAsCidr(ipStr string) (*v1alpha1.CIDR, error) {
	ip := net.ParseIP(ipStr)
	cidrRange := "/32"
	if ip.To4() == nil {
		cidrRange = "/128"
	}
	cidr, err := v1alpha1.CIDRFromString(ipStr + cidrRange)
	if err != nil {
		return nil, fmt.Errorf("unable to get CIDR from string: %w", err)
	}
	return cidr, nil
}

func (r *IpamReconciler) findSubnet(ctx context.Context, ipam *v1alpha1.Ipam) (*v1alpha1.Subnet, error) {
	subnet := &v1alpha1.Subnet{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: ipam.Namespace, Name: ipam.Spec.Subnet}, subnet); err != nil {
		return nil, fmt.Errorf("unable to get gateway of Subnet: %w", err)
	}
	return subnet, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&v1alpha1.Ipam{}).
		Complete(r)
}
