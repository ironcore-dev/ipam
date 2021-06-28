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
	"github.com/onmetal/ipam/api/v1alpha1/cidr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var iplog = logf.Log.WithName("ip-resource")
var c client.Client

func (r *Ip) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-ipam-onmetal-de-v1alpha1-ip,mutating=true,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=ips,verbs=create;update,versions=v1alpha1,name=mip.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Ip{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Ip) Default() {
	iplog.Info("default", "name", r.Name)

	if r.Spec.IP == "" {
		ctx := context.Background()
		var subnet Subnet
		if err := c.Get(ctx, client.ObjectKey{Namespace: r.Namespace, Name: r.Spec.Subnet}, &subnet); err != nil {
			iplog.Error(err, "unable to get gateway of Subnet")
			return
		}
		ip, err := r.getFreeIP(context.Background(), subnet.Spec.CIDR.String(), r.Namespace, r.Spec.Subnet)
		if err != nil {
			iplog.Error(err, "unable to get free IP")
			return
		}
		r.Spec.IP = ip
	}
}

//+kubebuilder:webhook:path=/validate-ipam-onmetal-de-v1alpha1-ip,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=ips,verbs=create;update,versions=v1alpha1,name=vip.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Ip{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Ip) ValidateCreate() error {
	iplog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Ip) ValidateUpdate(old runtime.Object) error {
	iplog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Ip) ValidateDelete() error {
	iplog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *Ip) validate() error {
	ctx := context.Background()
	var subnet Subnet
	subnetKey := client.ObjectKey{
		Namespace: r.Namespace,
		Name:      r.Spec.Subnet,
	}
	if err := c.Get(ctx, subnetKey, &subnet); err != nil {
		iplog.Error(err, "unable to find subnet", "name", r.Name, "subnet", subnetKey)
		return errors.Wrapf(err, "Subnet is not found: "+r.Spec.Subnet)
	}
	// Only check for CRD if it is specified
	if r.Spec.CRD != nil {
		// Lookup related CRD
		u := &unstructured.Unstructured{}
		gv, err := schema.ParseGroupVersion(r.Spec.CRD.GroupVersion)
		if err != nil {
			iplog.Error(err, "unable to parse CRD GroupVersion", "name", r.Name, "crd", r.Spec.CRD)
			return errors.Wrapf(err, "unable to parse CRD GroupVersion")
		}
		gvk := gv.WithKind(r.Spec.CRD.Kind)
		u.SetGroupVersionKind(gvk)
		key := client.ObjectKey{
			Namespace: r.Namespace,
			Name:      r.Spec.CRD.Name,
		}
		if err = c.Get(context.Background(), key, u); err != nil {
			iplog.Error(err, "unable to find CRD", "name", r.Name, "crd", r.Spec.CRD)
			return errors.Wrapf(err, "unable to find CRD")
		}
	}
	if r.Spec.IP != "" {
		free, err := r.isIPFree(ctx, r.Spec.IP, r.Namespace, r.Spec.Subnet)
		if err != nil {
			iplog.Error(err, "unable to check if IP is free")
			return errors.Wrapf(err, "unable to check if IP is free")
		}
		if !free {
			return errors.New("IP is already allocated")
		}
	}
	return nil
}

func (r *Ip) isIPFree(ctx context.Context, ip string, namespace string, subnetName string) (bool, error) {
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

func (r *Ip) getFreeIP(ctx context.Context, rootCidr string, namespace string, subnetName string) (string, error) {
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

func (r *Ip) findChildrenSubnetRanges(ctx context.Context, namespace string, subnetName string) ([]string, error) {
	subnets := []string{}
	subnetList := &SubnetList{}
	err := c.List(ctx, subnetList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	for index, subnet := range subnetList.Items {
		if subnet.Spec.ParentSubnetName == subnetName {
			subnets = append(subnets, subnetList.Items[index].Spec.CIDR.String())
		}
	}
	return subnets, nil
}

func (r *Ip) findReservedIPs(ctx context.Context, namespace string, subnetName string) ([]string, error) {
	reservedIPs := []string{}
	ipList := &IpList{}
	err := c.List(ctx, ipList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	for index, ip := range ipList.Items {
		if ip.Spec.Subnet == subnetName && ip.Spec.IP != "" && ip.Name != r.Name {
			reservedIPs = append(reservedIPs, ipList.Items[index].Spec.IP)
		}
	}
	return reservedIPs, nil
}
