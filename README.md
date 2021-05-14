# jx-secret-postrenderer

[![Documentation](https://godoc.org/github.com/jenkins-x-plugins/jx-secret-postrenderer?status.svg)](https://pkg.go.dev/mod/github.com/jenkins-x-plugins/jx-secret-postrenderer)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x-plugins/jx-secret-postrenderer)](https://goreportcard.com/report/github.com/jenkins-x-plugins/jx-secret-postrenderer)
[![Releases](https://img.shields.io/github/release-pre/jenkins-x-plugins/jx-secret-postrenderer.svg)](https://github.com/jenkins-x-plugins/jx-secret-postrenderer/releases)
[![LICENSE](https://img.shields.io/github/license/jenkins-x-plugins/jx-secret-postrenderer.svg)](https://github.com/jenkins-x-plugins/jx-secret-postrenderer/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)
    
## Overview

`jx-secret-postrenderer` is a [helm post renderer](https://helm.sh/docs/topics/advanced/#post-rendering) for working with [helm](https://helm.sh/) and [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets).

This post renderer lets you use helm charts which contain `Secret` resources and have those secrets managed by [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets) without having to modify your charts.

## How it works

When you use this helm post renderer:

* `Secret` resources are automatically translated to `ExternalSecret` resources on the fly when you use `helm install / helm update / helm template`
*  [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets) then populates any `Secret` resources from the underlying external secret store (e.g. vault or your cloud providers secret store)
* any missing secret values in your external secret store are initially populated from any generated secret values from your helm charts - or you can use a Secret schema to generate values from templates or generators

After installing a chart things should generally look and feel the same other than:

* you can modify secrets at any time in your secret store (e.g. vault or your cloud providers secret store) and [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets) will replicate the state to the kubernetes `Secret` resources
* you don't need to pre-create the vault or cloud provider secret store values by hand first which is error prone
    

## Editing secrets

Once you have installed your chart with this post renderer so that [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets) manages the secret data you can edit your secret values by:

* modify the secret using the vault / cloud provider secret store CLI
* use the [jx secret edit -i](https://jenkins-x.io/v3/develop/reference/jx/secret/edit/) command line [like this example](https://jenkins-x.io/v3/admin/setup/secrets/#edit-secrets)
  

## GitOps Integration
                                                                     
If you convert your helm charts into YAML via something like:

```bash 
helm template --post-renderer "jx-secret-postrenderer"  myname mychart
```

Then the results have no secret values in them - so you are free to check in all of that YAML safely into your git repository! 

This also avoids accidentally checking secrets into git.

We have found [this approach to GitOps to be a massive benefit](https://jenkins-x.io/v3/develop/faq/general/#why-does-jenkins-x-use-helmfile-template) on the [Jenkins X](https://jenkins-x.io/) project - as it makes it super easy to reason about how a particular kubernetes resource has changed over time via git history.


## Background

We initially tried to use sealed secrets or [sops](https://github.com/mozilla/sops) with [Jenkins X](https://jenkins-x.io/) and found them to be clumsy for UX developers to work with.

So in version 3.x of [Jenkins X](https://jenkins-x.io/) we [switched](https://jenkins-x.io/v3/admin/setup/secrets/) to using [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets) to handle population of `Secrets` from the underlying secret store (vault or your cloud providers secret store).

Once you move to [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets) things are much simpler; but it means converting lots of helm charts that use `Secret` to switch them to use `ExternalSecret` instead.

So we automated the translation of the `Secret` resources in any helm chart to use `ExternalSecret` resources and pre-populate any missing external secret store values with values genearted from the charts.

What this post renderer does is bring this capability from [Jenkins X](https://jenkins-x.io/) to anyone who is not using Jenkins X but is using [helm](https://helm.sh/) and [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets)



## Installation

You need to download the [jx-secret-postrenderer binary](https://github.com/jenkins-x-plugins/jx-secret-postrenderer/releases) binary for your platform and add it to your `$PATH`

You also need to make sure your kubernetes cluster has the [kubernetes external secrets helm chart installed](https://github.com/external-secrets/kubernetes-external-secrets#install-with-helm).
 
If you are working with vault you'll need to set the `$VAULT_ADDR` to point at your vault server.

If its inside the cluster you may need to [run a port forward](https://jenkins-x.io/v3/admin/setup/secrets/vault/#using-vault).
 

## User Guide

To use the [helm postrenderer](https://helm.sh/docs/topics/advanced/#post-rendering) you just append ` --post-renderer jx-secret-postrenderer` to any `helm install / helm upgrade / helm template` command.

e.g. to install the RabbitMQ helm chart, setting up the necessary secrets in your secret store, creating the `ExternalSecret` resource which will then populate the `Secret` run the following:

```bash 
helm install --post-renderer jx-secret-postrenderer rmq bitnami/rabbitmq
```

This will then convert the `Secret` resource inside the chart into an `ExternalSecret` resource and pre-populate your particular external secret store (which defaults to vault).

You should be able to see the external secret via 

```bash 
kubectl get externalsecrets 
```

### Helmfile support

If you use [helmfile](https://github.com/roboll/helmfile) then you can pass in the necessary command line arguments to [helm](https://helm.sh/) via something like:


```bash 
helmfile sync --args "--post-renderer jx-secret-postrenderer"
```


## Configuration
 
The following environment variables can be used to configure the helm post renderer

| Variable | Description |
| --- | --- |
| JX_DIR | the directory look in for the `.jx/secret/mapping/secret-mappings.yaml` file which is used to configure the secret mapping such as which kind of back end to use |
| JX_NO_POPULATE | set to `true` if you want to disable the pre-create and pre-population of the external store values and just translate the `Secret` resources to `ExternalSecret` |
             

### Secret Mapping

As you can see above you can point the `$JX_DIR` at a folder which contains a file `.jx/secret/mapping/secret-mappings.yaml` that [defines the Secret Mapping configuration](https://github.com/jenkins-x-plugins/jx-secret/blob/main/docs/mapping.md#secret.jenkins-x.io/v1alpha1.SecretMapping)

These files are automatically included in any Jenkins X environment. Here's a few examples:

| Example | Description |
| --- | --- |
| [alicloud](https://github.com/jx3-gitops-repositories/jx3-alicloud/blob/main/.jx/secret/mapping/secret-mappings.yaml) | for using alicloud secrets manager |
| [gsm](https://github.com/jx3-gitops-repositories/jx3-gke-gsm/blob/main/.jx/secret/mapping/secret-mappings.yaml) | for using Google Secrets Manager
| [vault](https://github.com/jx3-gitops-repositories/jx3-gke-vault/blob/main/.jx/secret/mapping/secret-mappings.yaml) | for using Hashicorp Vault |
| [local](https://github.com/jx3-gitops-repositories/jx3-kubernetes/blob/main/.jx/secret/mapping/secret-mappings.yaml) | for local secrets (i.e. no secret store) | 
