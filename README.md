# azdevops-agent-operator
Kubernetes operator for the Azure DevOps self-hosted pipe-line agent. The operator adds an extra layer of configuration on top of the default images like: azure devops pool settings and additional proxy settings.

# init.sh
```bash
#!/bin/bash

# docker and github repo username
export USERNAME='bartvanbenthem'
# image and bundle version
export VERSION=0.10.30
# operator repo and name
export OPERATOR_NAME='azdevops-agent-operator'
export OPERATOR_GROUP='azdevops'
export OPERATOR_KIND='Agent'
export OPERATOR_DOMAIN='gofound.nl'

cd $OPERATOR_NAME
# scaffold operator
operator-sdk init --domain $OPERATOR_DOMAIN --repo github.com/$USERNAME/$OPERATOR_NAME --skip-go-version-check
operator-sdk create api --group $OPERATOR_GROUP --version v1alpha1 --kind $OPERATOR_KIND --resource --controller

# always run make after changing *_types.go and *_controller.go
go get all
go mod tidy
make generate
make manifests

#######################################################
# Build the operator
make docker-build docker-push IMG=docker.io/$USERNAME/$OPERATOR_NAME:v$VERSION

#######################################################
# test and deploy the operator
make deploy IMG=docker.io/$USERNAME/$OPERATOR_NAME:v$VERSION
kubectl create ns test
kubectl -n test apply -f ../azdevops_v1alpha1_agent.yaml
kubectl -n test get agent agent-sample -o yaml
kubectl -n test get pods
kubectl -n test get secret agent-sample -o yaml
# check operator logs
sudo cat /var/log/containers/azdevops-agent-operator-controller-manager-
kubectl -n azdevops-agent-operator-system logs azdevops-agent-operator-controller-manager-

# cleanup test deployment
kubectl -n test delete -f ../azdevops_v1alpha1_agent.yaml
kubectl delete ns test
make undeploy

#######################################################
#######################################################
# Operator Lifecycle Manager - create and install bundle
operator-sdk olm install
operator-sdk olm status

# set env vars for creating the bundle
export IMG=docker.io/$USERNAME/$OPERATOR_NAME:v$VERSION
export BUNDLE_IMG=docker.io/$USERNAME/$OPERATOR_NAME-bundle:v$VERSION

# make the olm bundle
make bundle
# build and push the bundle image:
make bundle-build bundle-push
# verify bundle
operator-sdk bundle validate docker.io/$USERNAME/$OPERATOR_NAME-bundle:v$VERSION

# installing the bundle
operator-sdk run bundle docker.io/$USERNAME/$OPERATOR_NAME-bundle:v$VERSION --timeout 15m

# apply custom resource example
kubectl create ns test
kubectl -n test apply -f config/samples/azdevops_v1alpha1_agent.yaml
kubectl -n test get agent agent-sample

#######################################################
#######################################################
# Cleaning up your cluster
k -n test get operators
operator-sdk cleanup adoagent-operator
# If your bundle is in a bad state, you can clean up all the OLM and OLM-dependent 
# resources by running the following command:
operator-sdk olm uninstall


#######################################################
# Redeploying from scratch
operator-sdk olm install

operator-sdk run bundle docker.io/$USERNAME/$OPERATOR_NAME-bundle:v1.0.0
kubectl apply -f config/sample/ado_v1alpha1_agent.yaml

```

# Agent Sample
```yaml
apiVersion: azdevops.gofound.nl/v1alpha1
kind: Agent
metadata:
  name: agent-sample
spec:
  size: 2
  image: # image: bartvanbenthem/agent:v0.0.1
  pool:
    url: https://dev.azure.com/ProjectName
    token: exampleo4m6uekbfpodresprxcsa3fx4xduvkzvmojx
    poolName: operator-sh
    agentName: agent-sample
    workDir:
  proxy:
    httpProxy: http://proxy_server:port
    httpsProxy: https://proxy_server:port
    ftpProxy: http://proxy_server:port
    noProxy:
  mtuValue:
```