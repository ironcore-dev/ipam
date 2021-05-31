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
	"errors"
	"fmt"
	"github.com/onmetal/ipam/api/v1alpha1/cidr"
	subnetv1alpha1 "github.com/onmetal/k8s-subnet/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var log = logf.Log.WithName("ipam-resource")
var c client.Client

func (r *Ipam) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-ipam-onmetal-de-v1alpha1-ipam,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=ipams,verbs=create;update,versions=v1alpha1,name=mipam.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Ipam{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Ipam) Default() {
	log.Info("default", "name", r.Name)

	if r.Spec.IP == "" {
		ctx := context.Background()
		var subnet subnetv1alpha1.Subnet
		if err := c.Get(ctx, client.ObjectKey{Namespace: r.Namespace, Name: r.Spec.Subnet}, &subnet); err != nil {
			log.Error(err, "unable to get gateway of Subnet")
			return
		}
		ip, err := r.getFreeIP(context.Background(), subnet.Spec.CIDR, r.Namespace, r.Spec.Subnet)
		if err != nil {
			log.Error(err, "unable to get free IP for Ipam")
			return
		}
		r.Spec.IP = ip
	}
}

//+kubebuilder:webhook:path=/validate-ipam-onmetal-de-v1alpha1-ipam,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=ipams,verbs=create;update,versions=v1alpha1,name=vipam.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Ipam{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Ipam) ValidateCreate() error {
	log.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Ipam) ValidateUpdate(old runtime.Object) error {
	log.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Ipam) ValidateDelete() error {
	log.Info("validate delete", "name", r.Name)
	return nil
}

func (r *Ipam) validate() error {
	ctx := context.Background()
	var subnet subnetv1alpha1.Subnet
	if err := c.Get(ctx, client.ObjectKey{Namespace: r.Namespace, Name: r.Spec.Subnet}, &subnet); err != nil {
		log.Error(err, "unable to get gateway of Subnet")
		return errors.New("Subnet is not found: " + r.Spec.Subnet)
	}
	//var machineRequest machinerequestv1alpha1.MachineRequest
	//if err := c.Get(ctx, client.ObjectKey{Namespace: r.Namespace, Name: r.Spec.MachineRequest}, &machineRequest); err != nil {
	//	log.Error(err, "unable to fetch MachineRequest")
	//	return errors.New("MachineRequest is not found: " + r.Spec.Subnet)
	//}
	// TODO validate related CRD
	if r.Spec.IP != "" {
		free, err := r.isIPFree(ctx, r.Spec.IP, r.Namespace, r.Spec.Subnet)
		if err != nil {
			log.Error(err, "unable to check if IP is free")
			return err
		}
		if !free {
			return errors.New("IP is already allocated")
		}
	}
	return nil
}

func (r *Ipam) isIPFree(ctx context.Context, ip string, namespace string, subnetName string) (bool, error) {
	ranges, err := r.findChildrenSubnetRanges(ctx, namespace, subnetName)
	if err != nil {
		return false, fmt.Errorf("unable to get children ranges: %w", err)
	}
	reserved, err := r.findReservedIPs(ctx, namespace, subnetName)
	if err != nil {
		return false, fmt.Errorf("unable to get reserved IPs: %w", err)
	}
	free, err := cidr.IsIpFree(ranges, reserved, ip)
	if err != nil {
		return false, fmt.Errorf("unable to get free IP: %w", err)
	}
	return free, nil
}

func (r *Ipam) getFreeIP(ctx context.Context, rootCidr string, namespace string, subnetName string) (string, error) {
	ranges, err := r.findChildrenSubnetRanges(ctx, namespace, subnetName)
	if err != nil {
		return "", fmt.Errorf("unable to get children ranges: %w", err)
	}
	reserved, err := r.findReservedIPs(ctx, namespace, subnetName)
	if err != nil {
		return "", fmt.Errorf("unable to get reserved IPs: %w", err)
	}
	ip, err := cidr.GetFirstFreeIP(rootCidr, ranges, reserved)
	if err != nil {
		return "", fmt.Errorf("unable to get free IP: %w", err)
	}
	return ip, nil
}

func (r *Ipam) findChildrenSubnetRanges(ctx context.Context, namespace string, subnetName string) ([]string, error) {
	subnets := []string{}
	subnetList := &subnetv1alpha1.SubnetList{}
	err := c.List(ctx, subnetList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	for index, subnet := range subnetList.Items {
		if subnet.Spec.SubnetParentID == subnetName {
			subnets = append(subnets, subnetList.Items[index].Spec.CIDR)
		}
	}
	return subnets, nil
}

func (r *Ipam) findReservedIPs(ctx context.Context, namespace string, subnetName string) ([]string, error) {
	reservedIPs := []string{}
	ipamList := &IpamList{}
	err := c.List(ctx, ipamList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	for index, ipam := range ipamList.Items {
		if ipam.Spec.Subnet == subnetName && ipam.Spec.IP != "" && ipam.Name != r.Name {
			reservedIPs = append(reservedIPs, ipamList.Items[index].Spec.IP)
		}
	}
	return reservedIPs, nil
}
