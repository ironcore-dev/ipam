# IPAM

k8s operator for IPAM.

With this operator it is possible to:
- manage networks and their IDs for VXLAN, MPLS and GENEVE types;
- manage subnets and their CIDRs;
- manage IP allocations.

## Getting started

### Required tools

Following tools are required to work on that package.

- k8s cluster access to deploy and test the result (via minikube or docker desktop locally)
- [make](https://www.gnu.org/software/make/) - to execute build goals
- [golang](https://golang.org/) - to compile the code
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) - to interact with k8s cluster via CLI
- [kustomize](https://kustomize.io/) - to generate deployment configs
- [kubebuilder](https://book.kubebuilder.io) - framework to build operators
- [operator framework](https://operatorframework.io/) - framework to maintain project structure
- [helm](https://helm.sh/) - to work with helm charts
- [cert-manager](https://cert-manager.io/) - to issue certificates for webhook endpoints

### Prepare environment

If you have an access to the docker registry and k8s installation that you can use for development purposes, you may skip
corresponding steps.

Otherwise, create a local instance of docker registry and k8s.

    # start minikube
    minikube start
    # enable registry
    minikube addons enable registry
    # run proxy to registry
    docker run --rm -d --name registry-bridge --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"
    # install cert-manager for k8s cluster
    # check "Releases" page to fetch up-to-date version
    # https://github.com/jetstack/cert-manager/releases 
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.1/cert-manager.yaml

### Build and install

In order to build and deploy, execute following command set.

Docker registry is required to build and push an image.

Replace `localhost:5000` with your registry if you're using quay or anything else.

    # generate code and configs
    make fmt vet generate manifests kustomize
    # build container and push to local registry
    make docker-build docker-push IMG="localhost:5000/onmetal/ipam:latest"
    # deploy controller
    make deploy IMG="localhost:5000/onmetal/ipam:latest"

Check `Makefile` for the full list of `make` goals with descriptions.

### Use

`./config/samples/` directory contains examples of manifests. They can be used to try out the controller.

    # apply config
    kubectl apply -f ./config/samples/ipam_v1alpha1_network.yaml
    # get resources
    kubectl get networks
    # get sample resource
    kubectl describe network network-sample

### Clean

After development is done, clean up local environment.

    # generate deployment config and delete corresponding entities
    kustomize build config/default | kubectl delete -f -
    # remove registry bridge
    docker stop registry-bridge
    # stop minikube
    minikube stop

## Deployment

Operator can be deployed to the environment with kubectl, kustomize or Helm. User may choose one that is more suitable.

### With kubectl using kustomize configs

    # deploy
    kubectl apply -k config/default/
    # remove
    kubectl delete -k config/default/

### With kustomize

    # build and apply
    kustomize build config/default | kubectl apply -f -
    # build and remove
    kustomize build config/default | kubectl delete -f -

### With Helm chart

    # install release "dev" to "onmetal" namespace
    helm install dev ./chart/ -n onmetal --create-namespace
    # remove release "dev" from "onmetal" namespace
    helm uninstall dev -n onmetal

## Usage

### Resources

IPAM process is held by 3 main resources: Networks, Subnets and IPs.
There is also a supplicant Network Counter resource that handles unique network IP accounting and acquisition.

All resources are sharing similar concepts in status representation. 
Every resource has a `state`, that may have `Processing`, `Finished` or `Failed` value.
If resource has a `Failed` state, `Message` should contain an explanation why it wasn't processed correctly.
If resource has been processed successfully, i.e. precessing has been `Finished`, `Reserved` field will have a 
corresponding operation result, ID for Network, CIDR for Subnet or IP addres for IP.

### Networks

Network is a top level resource that identifies unique address space.
It means that 2 different Networks may have clashing address zones, and, on the other hand, it is not possible to have 
matching addresses at the same subnet. 

Main responsibilities of the Network resource are to handle address space integrity, account child subnets and supply 
networks with valid unique IDs corresponding to network technology.

Network is able to handle both IPv4 and IPv6 address spaces simultaneously.

A proper Network CR should be formed following the rules below. 

```yaml
apiVersion: ipam.onmetal.de/v1alpha1
kind: Network
metadata:
  name: network-sample
spec:
  # Description is a free text description for network
  # Optional
  # String
  description: sample
  # ID is a network identifier  
  # Optional, will be generated if not set
  # Numeric string
  # Valid values for VXLAN: from 100 to 2^24 (3 byte value)
  # Valid values for GENEVE: from 100 to 2^24 (3 byte value)
  # Valid values for MPLS: from 16 to +inf (composite of 20 bit labels)
  id: "1000"
  # Type is a type of technology used to organize network
  # Optional, but required if ID is set
  # String (enum)
  # Valid values: VXLAN, GENEVE, MPLS
  type: GENEVE
```

When network is in use, `kubectl` is able to show its type, reserved ID and total amount of addresses in child subnets. 

```shell
[user@localhost ~]$ kubectl get networks
NAME                    TYPE     RESERVED   IPV4 CAPACITY   IPV6 CAPACITY          DESCRIPTION   STATE      MESSAGE
network-sample          MPLS     16         16777216        18446744073709551616   mpls net      Finished
```

If there is a need to have more precise date on ranges' availability for the selected network, Network request status
may be inspected. It contains a list of address ranges booked by subnets.  

```shell
[user@localhost ~]$ kubectl describe network mpls-network-sample
Name:         network-sample
Namespace:    default
API Version:  ipam.onmetal.de/v1alpha1
Kind:         Network
Status:
  ipv4Capacity:  16777216
  ipv4Ranges:
    10.0.0.0/8
  ipv6Capacity:  18446744073709551616
  ipv6Ranges:
    fd34:5d8f:e75e:f3a2::/64
  Reserved:      16
  State:  Finished
...
```

If an exact ID should be picked for the Network, a counter for the corresponding network technology may be checked.

```shell
[user@localhost ~]$ kubectl get networkcounters
NAME                             AGE
k8s-geneve-network-counter       6d
k8s-mpls-network-counter         6d
k8s-vxlan-network-counter        6d
```

Counter itself maintains ranges of vacant inclusive ID intervals. If interval has an `Exact` field set, as in example,
it means that it has only one value in the interval. Interval may also have an open border, i.e. no `Begin` or `End`
value; it means that there is no limitation on min/max value.
If `Vacant` collection is empty, then there are no intervals left.

```shell
[user@localhost ~]$ kubectl describe networkcounter k8s-vxlan-network-counter
Name:         k8s-vxlan-network-counter
Namespace:    default
API Version:  ipam.onmetal.de/v1alpha1
Kind:         NetworkCounter
Spec:
  Vacant:
    Exact:  100
    Begin:  102
    End:    16777215
```

Examples:
- [empty network](config/samples/ipam_v1alpha1_network.yaml);
- [network with VXLAN ID request](config/samples/ipam_v1alpha1_vxlan_network.yaml);
- [network with GENEVE ID request](config/samples/ipam_v1alpha1_geneve_network.yaml);
- [network with MPLS ID request](config/samples/ipam_v1alpha1_mpls_network.yaml).

### Subnets 

Subnets are representing an IP address ranges in a CIDR format.

Subnets may be split into 2 categories by their relations: 
1. Top level Subnets. These Subnets don't have a parent Subnet, they may define any unoccupied CIDR in the Network.
   To allocate a top level Subnet, it should specify CIDR explicitly.
2. Child Subnets. These Subnets have other Subnet as a parent, and their address range and region scope 
   should be within the scope of a parent. For child Subnets it is also possible to specify required address ranges 
   by capacity or netmask prefix bits (bits occupied by ones). In that case a first smallest subnet matching the criteria
   mey be picked.
   
Subnets may be also categorized by their regional affiliation:
1. Multiregional - Subnet that has more than one Region specified.
2. Regional - Subnet that has one region and multiple availability zones.
3. Local - Subnet with one region and one availability zone.

Here is an explanation on how to setup the Subnet.

```yaml
apiVersion: ipam.onmetal.de/v1alpha1
kind: Subnet
metadata:
  name: subnet-sample
spec:
  # CIDR describes an IP range for the subnet
  # Required for top level subnets
  # Optional for child subnets
  # String
  # Only and at least one of cidr, prefixBits, capacity should be set
  # If parent subnet is set, should be within address range of parent subnet
  cidr: "10.0.0.0/16"
  # PrefixBits is an amount of ones (occupied bits) in netmask 
  # Optional
  # Can not be set for top level subnet
  # Number
  # Only and at least one of cidr, prefixBits, capacity should be set
  # Valid values: 0-128
  # Usage will result in reservation of CIDR in address range of parent subnet
  # First smallest vacant CIDR in parent address range will be picked for range withdrawal
  prefixBits: 16
  # Capacity is an amount of addresses required
  # Optional
  # Can not be set for top level subnet
  # Numeric string
  # Only and at least one of cidr, prefixBits, capacity should be set
  # Valid values: from 1 to 2^128
  # Usage will result in reservation of CIDR in address range of parent subnet
  # Capacity will be ceiled to next power of 2, if it is not power of 2 itself
  # First smallest vacant CIDR in parent address range will be picked for range withdrawal
  capacity: "100"
  # ParentSubnetName refers to the parent network at the same namespace
  # Optional
  # String
  # Should refer an existing subnet resource
  parentSubnetName: "ipv4-parent-cidr-subnet-sample"
  # NetworkName refers to the parent network at the same namespace
  # Required
  # String
  # Should refer an existing network resource
  networkName: network-sample
  # Regions is a list of regions subnet is attached to
  # Required
  # Set of objects (uniqueness is defined by name)
  # If parent subnet is set, should be a subset of parent's region set, including AZ sets in matching regions
  regions:
      # Name is a unique name of the region for subnet tree 
      # Required
      # String 
      # Should meet DNS label rules
    - name: euw
      # AvailabilityZones is a list of availability zones subnet is attached to
      # Required
      # Set of strings
      # If parent subnet is set, should be a subset of parent's az set in matching region
      availabilityZones:
        - a
        - b
```

Apart of the data specified in manifest, Subnet's status also contains its address capacity (count) and capacity left,
that is total capacity, minus capacity of child Subnets and individual IPs allocated on that Subnet.

```shell
[user@localhost ~]$ kubectl get subnets
NAME                                PARENT SUBNET                    PARENT NETWORK   RESERVED                            ADDRESS TYPE   LOCALITY   PREFIX BITS   CAPACITY               CAPACITY LEFT          STATE      MESSAGE
ipv4-child-bits-subnet-sample       ipv4-parent-cidr-subnet-sample   network-sample   10.1.0.0/16                         IPv4           Regional   16            65536                  65536                  Finished   
ipv4-child-capacity-subnet-sample   ipv4-parent-cidr-subnet-sample   network-sample   10.2.0.0/25                         IPv4           Regional   25            128                    128                    Finished   
ipv4-child-cidr-subnet-sample       ipv4-parent-cidr-subnet-sample   network-sample   10.0.0.0/16                         IPv4           Regional   16            65536                  65532                  Finished   
ipv4-parent-cidr-subnet-sample                                       network-sample   10.0.0.0/8                          IPv4           Regional   8             16777216               16646016               Finished   
ipv6-child-bits-subnet-sample       ipv6-parent-cidr-subnet-sample   network-sample   fd34:5d8f:e75e:f3a2:1000::/88       IPv6           Regional   88            1099511627776          1099511627776          Finished   
ipv6-child-capacity-subnet-sample   ipv6-parent-cidr-subnet-sample   network-sample   fd34:5d8f:e75e:f3a2:1000:100::/95   IPv6           Regional   95            8589934592             8589934592             Finished   
ipv6-child-cidr-subnet-sample       ipv6-parent-cidr-subnet-sample   network-sample   fd34:5d8f:e75e:f3a2::/68            IPv6           Regional   68            1152921504606846976    1152921504606846973    Finished   
ipv6-parent-cidr-subnet-sample                                       network-sample   fd34:5d8f:e75e:f3a2::/64            IPv6           Regional   64            18446744073709551616   17293821461001142272   Finished
```

Vacant ranges left may be also checked with `describe` method of `kubectl`.

```shell
[user@localhost ~]$ kubectl describe subnet ipv4-parent-cidr-subnet-sample
Name:         ipv4-parent-cidr-subnet-sample
Namespace:    default
API Version:  ipam.onmetal.de/v1alpha1
Kind:         Subnet
Status:
  Capacity:       16777216
  Capacity Left:  16646016
  Locality:       Regional
  Prefix Bits:    8
  Reserved:       10.0.0.0/8
  State:          Finished
  Type:           IPv4
  Vacant:
    10.2.0.128/25
    10.2.1.0/24
    10.2.2.0/23
    10.2.4.0/22
    10.2.8.0/21
    10.2.16.0/20
    10.2.32.0/19
    10.2.64.0/18
    10.2.128.0/17
    10.3.0.0/16
    10.4.0.0/14
    10.8.0.0/13
    10.16.0.0/12
    10.32.0.0/11
    10.64.0.0/10
    10.128.0.0/9
```

Examples:
- [IPv4 parent (top level) subnet](config/samples/ipam_v1alpha1_ipv4_parent_cidr_subnet.yaml);
- [IPv4 child subnet with CIDR set explicitly](config/samples/ipam_v1alpha1_ipv4_child_cidr_subnet.yaml);
- [IPv4 child subnet with CIDR requested by network prefix bits](config/samples/ipam_v1alpha1_ipv4_child_bits_subnet.yaml);
- [IPv4 child subnet with CIDR requested by address capacity](config/samples/ipam_v1alpha1_ipv4_child_capacity_subnet.yaml);
- [IPv6 parent (top level) subnet](config/samples/ipam_v1alpha1_ipv6_parent_cidr_subnet.yaml);
- [IPv6 child subnet with CIDR set explicitly](config/samples/ipam_v1alpha1_ipv6_child_cidr_subnet.yaml);
- [IPv6 child subnet with CIDR requested by network prefix bits](config/samples/ipam_v1alpha1_ipv6_child_bits_subnet.yaml);
- [IPv6 child subnet with CIDR requested by address capacity](config/samples/ipam_v1alpha1_ipv6_child_capacity_subnet.yaml);

### IPs

IPs are basically individual addresses, and the may be also represented in a form of /32 or /128 CIDRs for IPv4 and IPv6
correspondingly.

IPs are always booked on specified Subnet as CIDRs, reducing their capacity.

IPs may or may not point to resource they are assigned to. 

```yaml
apiVersion: ipam.onmetal.de/v1alpha1
kind: Ip
metadata:
  name: ip-sample
spec:
  # SubnetName is a reference to subnet where IP should be reserved
  # Required
  # String
  # Should refer to an existing subnet at the same namespace
  subnetName: ipv4-child-cidr-subnet-sample
  # Resource is a reference to k8s resource IP would be bountd to
  # Optional
  # Object with string fields
  resourceReference:
    apiVersion: ipam.onmetal.de/v1alpha1
    kind: SampleReource
    name: sample-resorce-name
  # IP
  # Optional
  # String
  # If not specified, IP from the first smallest vacant CIDR of referred subnet would be picked
  ip: 10.0.0.2
```

Sample output for the `kubectl`.

```shell
[user@localhost ~]$ kubectl get ips
NAME                             IP                       SUBNET                          RESOURCE GROUP             RESOURCE KIND    RESOURCE NAME                    STATE      MESSAGE
ipv4-ip-ip-sample                10.0.0.1                 ipv4-child-cidr-subnet-sample                                                                                Finished   
ipv4-ip-sample                   10.0.0.3                 ipv4-child-cidr-subnet-sample                                                                                Finished   
ipv4-resource-and-ip-ip-sample   10.0.0.2                 ipv4-child-cidr-subnet-sample   ipam.onmetal.de/v1alpha1   NetworkCounter   referred-networkcounter-sample   Finished   
ipv4-resource-ip-sample          10.0.0.0                 ipv4-child-cidr-subnet-sample   ipam.onmetal.de/v1alpha1   NetworkCounter   referred-networkcounter-sample   Finished   
ipv6-ip-ip-sample                fd34:5d8f:e75e:f3a2::1   ipv6-child-cidr-subnet-sample                                                                                Finished   
ipv6-ip-sample                   fd34:5d8f:e75e:f3a2::3   ipv6-child-cidr-subnet-sample                                                                                Finished   
ipv6-resource-and-ip-ip-sample   fd34:5d8f:e75e:f3a2::2   ipv6-child-cidr-subnet-sample   ipam.onmetal.de/v1alpha1   NetworkCounter   referred-networkcounter-sample   Finished   
ipv6-resource-ip-sample          fd34:5d8f:e75e:f3a2::    ipv6-child-cidr-subnet-sample   ipam.onmetal.de/v1alpha1   NetworkCounter   referred-networkcounter-sample   Finished
```

IPs status is pretty simple and does not provide any additional info. 

```shell
Name:         ipv4-ip-ip-sample
Namespace:    default
API Version:  ipam.onmetal.de/v1alpha1
Kind:         Ip
Status:
  Reserved:   10.0.0.1
  State:      Finished
...
```

Examples:
- [IPv4 IP request](config/samples/ipam_v1alpha1_ipv4_ip.yaml);
- [IPv4 IP request with reference to related resource](config/samples/ipam_v1alpha1_ipv4_resource_ip.yaml);
- [IPv4 IP request with IP set explicitly](config/samples/ipam_v1alpha1_ipv4_ip_ip.yaml);
- [IPv4 IP request with reference to related resource and IP set explicitly](config/samples/ipam_v1alpha1_ipv4_resource_and_ip_ip.yaml);
- [IPv6 IP request](config/samples/ipam_v1alpha1_ipv6_ip.yaml);
- [IPv6 IP request with reference to related resource](config/samples/ipam_v1alpha1_ipv6_resource_ip.yaml);
- [IPv6 IP request with IP set explicitly](config/samples/ipam_v1alpha1_ipv6_ip_ip.yaml);
- [IPv6 IP request with reference to related resource and IP set explicitly](config/samples/ipam_v1alpha1_ipv6_resource_and_ip_ip.yaml);

## Consuming API with client

Package provides a client library written in go for programmatic interactions with API.

Clients for corresponding API versions are located in `clientset/` and act similar to [client-go](https://github.com/kubernetes/client-go).

Below there are two examples, for inbound (from the pod deployed on a cluster) and outbound (from the program running on 3rd party resources) interactions.

### Inbound example

```go
import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	apiv1alpha1 "github.com/onmetal/ipam/api/v1alpha1"
	clientv1alpha1 "github.com/onmetal/ipam/clientset"
)

func inbound() error {
	// get config from environment
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// register CRD types in local client scheme
	if err := apiv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return errors.Wrap(err, "unable to add registered types to client scheme")
	}

	// create a client from obtained configuration
	cs, err := clientset.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "unable to build clientset from config")
	}

	// get a client for particular namespace
	client := clientset.IpamV1Alpha1().Networks("default")

	// request a list of resources
	list, err := client.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "unable to get list of resources")
	}

	// print names of resources
	for _, r := range list.Items {
		fmt.Println(r.Name)
	}

	return nil
}
```

### Outbound example

```go
import (
    "context"
    "fmt"
    "path/filepath"
    
    "github.com/pkg/errors"
    "github.com/spf13/pflag"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
    
    "k8s.io/client-go/kubernetes/scheme"

    apiv1alpha1 "github.com/onmetal/ipam/api/v1alpha1"
    clientv1alpha1 "github.com/onmetal/ipam/clientset"
)

func outbound() error {
	// make default path to kubeconfig empty
	var kubeconfigDefaultPath string

	// if there is a home directory in environment,
	// alter the defalut path to ~/.kube/config
	if home := homedir.HomeDir(); home != "" {
		kubeconfigDefaultPath = filepath.Join(home, ".kube", "config")
	}

	// configure the kubeconfig CLI flag with dafault value
	kubeconfig := pflag.StringP("kubeconfig", "k", kubeconfigDefaultPath, "path to kubeconfig")
	// configure k8s namespace flag with "default" default value
	namespace := pflag.StringP("namespace", "n", "default", "k8s namespace")
	// parse flags
	pflag.Parse()

	// read in kubeconfig file and build configuration
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return errors.Wrapf(err, "unable to read kubeconfig from path %s", kubeconfig)
	}

	// register CRD types in local client scheme
	if err := apiv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return errors.Wrap(err, "unable to add registered types to client scheme")
	}

	// create a client from obtained configuration
	cs, err := clientset.NewForConfig(config)
        if err != nil {
        return errors.Wrap(err, "unable to build clientset from config")
    }

	// get a client for particular namespace
	client := clientset.IpamV1Alpha1().Networks(*namespace)
	
	// request a list of resources
	list, err := client.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "unable to get list of resources")
	}
	
	// print names of resources
	for _, r := range list.Items {
		fmt.Println(r.Name)
	}

	return nil
}
```

## Development

### Running locally without deployment 

It is possible to run operator locally during development. To do this, execute `make run`. In that case 
operator will use current context from kubeconfig to connect to k8s' apiserver.

If problem running webhook occures, use `ENABLE_WEBHOOKS=false` env var to disable them on local run.

### Adding a new API version

One should not modify API once it got to be used.

Instead, in order to introduce breaking changes to the API, a new API version should be created.

First, move the existing controller to a different file, as generator will try to put a new controller into the same location, e.g.

    mv controllers/network_controller.go controllers/network_v1alpha1_controller.go

After that, add a new API version

    operator-sdk create api --group machine --version v1alpha2 --kind Network --resource --controller

Do modifications in a new CR, add a new controller to `main.go`.

Following actions should be applied to other parts of project:
- regenerate code and configs with `make install`
- add a client to client set for the new API version
- alter Helm chart with new CRD spec

### Deprecating old APIs

Since there is no version deprecation marker available now, old APIs may be deprecated with `kustomize` patches

Describe deprecation status and deprecation warning in patch file, e.g. `crd_patch.yaml`

```
- op: add
  path: "/spec/versions/0/deprecated"
  value: true
- op: add
  path: "/spec/versions/0/deprecationWarning"
  value: "This API version is deprecated. Check documentation for migration instructions."
```

Add patch instructions to `kustomization.yaml`

```
patchesJson6902:
  - target:
      version: v1
      group: apiextensions.k8s.io
      kind: CustomResourceDefinition
      name: ipam.onmetal.de
    path: crd_patch.yaml
```

When you are ready to drop the support for the API version, give CRD a `+kubebuilder:skipversion` marker,
or just remove it completely from the code base.

This includes:
- API objects
- client
- controller
