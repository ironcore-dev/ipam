#!/usr/bin/env bash
#// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
#// SPDX-License-Identifier: Apache-2.0

update_settings(k8s_upsert_timeout_secs=60)  # on first tilt up, often can take longer than 30 seconds

settings = {
    "allowed_contexts": [
        "kind-ipam"
    ],
    "kubectl": "./bin/kubectl",
    "cert_manager_version": "v1.15.3",
}

kubectl = settings.get("kubectl")

if "allowed_contexts" in settings:
    allow_k8s_contexts(settings.get("allowed_contexts"))

def deploy_cert_manager():
    version = settings.get("cert_manager_version")
    print("Installing cert-manager")
    local("{} apply -f https://github.com/cert-manager/cert-manager/releases/download/{}/cert-manager.yaml".format(kubectl, version), quiet=True, echo_off=True)

    print("Waiting for cert-manager to start")
    local("{} wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager".format(kubectl), quiet=True, echo_off=True)
    local("{} wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager-cainjector".format(kubectl), quiet=True, echo_off=True)
    local("{} wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager-webhook".format(kubectl), quiet=True, echo_off=True)

def waitforsystem():
    print("Waiting for ipam-operator to start")
    local("{} wait --for=condition=ready --timeout=300s -n ipam-system pod --all".format(kubectl), quiet=False, echo_off=True)

##############################
# Actual work happens here
##############################

deploy_cert_manager()

docker_build('ironcore-dev/ipam', '.')

yaml = kustomize('./config/default')

k8s_yaml(yaml)

k8s_yaml('./config/samples/ipam_v1alpha1_network.yaml')
k8s_resource(
    objects=['network-sample:network'],
    new_name='network-sample',
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

k8s_yaml('./config/samples/ipam_v1alpha1_ipv4_child_cidr_subnet.yaml')
k8s_resource(
    objects=['ipv4-child-cidr-subnet-sample:subnet'],
    new_name='ipv4-child-cidr-subnet-sample',
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

k8s_yaml('./config/samples/ipam_v1alpha1_ipv4_ip.yaml')
k8s_resource(
    objects=['ipv4-ip-sample:ip'],
    new_name='ipv4-ip-sample',
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)
