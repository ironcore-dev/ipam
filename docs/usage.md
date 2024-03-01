# Usage

## Resources

IPAM process is held by 3 main resources: Networks, Subnets and IPs.
There is also a supplicant Network Counter resource that handles unique network IP accounting and acquisition.

All resources are sharing similar concepts in status representation. 
Every resource has a `state`, that may have `Processing`, `Finished` or `Failed` value.
If resource has a `Failed` state, `Message` should contain an explanation why it wasn't processed correctly.
If resource has been processed successfully, i.e. precessing has been `Finished`, `Reserved` field will have a 
corresponding operation result, ID for Network, CIDR for Subnet or IP addres for IP.

## Networks

Network is a top level resource that identifies unique address space.
It means that 2 different Networks may have clashing address zones, and, on the other hand, it is not possible to have 
matching addresses at the same subnet. 

Main responsibilities of the Network resource are to handle address space integrity, account child subnets and supply 
networks with valid unique IDs corresponding to network technology.

Network is able to handle both IPv4 and IPv6 address spaces simultaneously.

A proper Network CR should be formed following the rules below. 

```yaml
apiVersion: ipam.ironcore.dev/v1alpha1
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
API Version:  ipam.ironcore.dev/v1alpha1
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
API Version:  ipam.ironcore.dev/v1alpha1
Kind:         NetworkCounter
Spec:
  Vacant:
    Exact:  100
    Begin:  102
    End:    16777215
```

Examples:
- [empty network](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_network.yaml);
- [network with VXLAN ID request](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_vxlan_network.yaml);
- [network with GENEVE ID request](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_geneve_network.yaml);
- [network with MPLS ID request](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_mpls_network.yaml).

## Subnets 

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
apiVersion: ipam.ironcore.dev/v1alpha1
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
  # ParentSubnet refers to the parent network at the same namespace
  # Optional
  # Object
  # Should refer an existing subnet resource
  parentSubnet:
    name: "ipv4-parent-cidr-subnet-sample"
  # Network refers to the parent network at the same namespace
  # Required
  # Object
  # Should refer an existing network resource
  network:
    name: network-sample
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
  # Consumer is a reference to k8s resource IP would be bound to
  # Optional
  # Object with string fields
  consumer:
     apiVersion: ipam.ironcore.dev/v1alpha1
     kind: SampleReource
     name: sample-resorce-name
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
API Version:  ipam.ironcore.dev/v1alpha1
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
- [IPv4 parent (top level) subnet](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv4_parent_cidr_subnet.yaml);
- [IPv4 child subnet with CIDR set explicitly](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv4_child_cidr_subnet.yaml);
- [IPv4 child subnet with CIDR requested by network prefix bits](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv4_child_bits_subnet.yaml);
- [IPv4 child subnet with CIDR requested by address capacity](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv4_child_capacity_subnet.yaml);
- [IPv6 parent (top level) subnet](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv6_parent_cidr_subnet.yaml);
- [IPv6 child subnet with CIDR set explicitly](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv6_child_cidr_subnet.yaml);
- [IPv6 child subnet with CIDR requested by network prefix bits](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv6_child_bits_subnet.yaml);
- [IPv6 child subnet with CIDR requested by address capacity](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv6_child_capacity_subnet.yaml);

## IPs

IPs are basically individual addresses, and the may be also represented in a form of /32 or /128 CIDRs for IPv4 and IPv6
correspondingly.

IPs are always booked on specified Subnet as CIDRs, reducing their capacity.

IPs may or may not point to resource they are assigned to. 

```yaml
apiVersion: ipam.ironcore.dev/v1alpha1
kind: Ip
metadata:
  name: ip-sample
spec:
  # Subnet is a reference to subnet where IP should be reserved
  # Required
  # Object
  # Should refer to an existing subnet at the same namespace
  subnet:
    name: ipv4-child-cidr-subnet-sample
  # Consumer is a reference to k8s resource IP would be bound to
  # Optional
  # Object with string fields
  consumer:
    apiVersion: ipam.ironcore.dev/v1alpha1
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
NAME                             IP                       SUBNET                          RESOURCE GROUP               RESOURCE KIND    RESOURCE NAME                    STATE      MESSAGE
ipv4-ip-ip-sample                10.0.0.1                 ipv4-child-cidr-subnet-sample                                                                                  Finished   
ipv4-ip-sample                   10.0.0.3                 ipv4-child-cidr-subnet-sample                                                                                  Finished   
ipv4-resource-and-ip-ip-sample   10.0.0.2                 ipv4-child-cidr-subnet-sample   ipam.ironcore.dev/v1alpha1   NetworkCounter   referred-networkcounter-sample   Finished   
ipv4-resource-ip-sample          10.0.0.0                 ipv4-child-cidr-subnet-sample   ipam.ironcore.dev/v1alpha1   NetworkCounter   referred-networkcounter-sample   Finished   
ipv6-ip-ip-sample                fd34:5d8f:e75e:f3a2::1   ipv6-child-cidr-subnet-sample                                                                                  Finished   
ipv6-ip-sample                   fd34:5d8f:e75e:f3a2::3   ipv6-child-cidr-subnet-sample                                                                                  Finished   
ipv6-resource-and-ip-ip-sample   fd34:5d8f:e75e:f3a2::2   ipv6-child-cidr-subnet-sample   ipam.ironcore.dev/v1alpha1   NetworkCounter   referred-networkcounter-sample   Finished   
ipv6-resource-ip-sample          fd34:5d8f:e75e:f3a2::    ipv6-child-cidr-subnet-sample   ipam.ironcore.dev/v1alpha1   NetworkCounter   referred-networkcounter-sample   Finished
```

IPs status is pretty simple and does not provide any additional info. 

```shell
Name:         ipv4-ip-ip-sample
Namespace:    default
API Version:  ipam.ironcore.dev/v1alpha1
Kind:         Ip
Status:
  Reserved:   10.0.0.1
  State:      Finished
...
```

Examples:
- [IPv4 IP request](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv4_ip.yaml);
- [IPv4 IP request with reference to related resource](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv4_resource_ip.yaml);
- [IPv4 IP request with IP set explicitly](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv4_ip_ip.yaml);
- [IPv4 IP request with reference to related resource and IP set explicitly](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv4_resource_and_ip_ip.yaml);
- [IPv6 IP request](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv6_ip.yaml);
- [IPv6 IP request with reference to related resource](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv6_resource_ip.yaml);
- [IPv6 IP request with IP set explicitly](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv6_ip_ip.yaml);
- [IPv6 IP request with reference to related resource and IP set explicitly](https://github.com/ironcore-dev/ipam/blob/main/config/samples/ipam_v1alpha1_ipv6_resource_and_ip_ip.yaml);
