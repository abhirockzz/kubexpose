# kubexpose: Kubernetes Operator to expose a deployment to the internet using `ngrok`

*work in progress*

An Operator example based on [Expose Kubernetes services with ngrok](https://medium.com/@abhishek1987/expose-kubernetes-services-with-ngrok-65280142dab4) - uses vanilla Kubernetes APIs.

> Thanks to [Extending Kubernetes: Create Controllers for Core and Custom Resources](https://medium.com/@trstringer/create-kubernetes-controllers-for-core-and-custom-resources-62fc35ad64a3)

## Pre-requisites

1. Make sure that the `config` file in `$HOME/.kube/config` points to the Kubernetes cluster against which you want to test this out

> this could be anything like minikube, K8s on Docker for Mac, GKE, AKS etc.

2. You have Go installed

## Steps

Clone the repo and change to directory

    git clone https://github.com/abhirockzz/kubexpose
    cd kubexpose
    
Build the operator binary

    go build -o kubexpose-operator

Seed the `Kubexpose` CRD (Custom Resource Definition)

    kubectl apply -f crd/kubexpose.yaml

Start operator

    ./kubexpose-operator

## Test drive

Create `nginx` Deployment and Service for testing

    kubectl create -f app/app.yaml 

Create `kubexpose` resource

    kubectl apply -f custom-resource/kubexpose-cr.yaml 

You should see a new Deployment named `kubexpose-nginx` - check logs to confirm. Now search for the Pod

    kubectl exec $(kubectl get pods -l=app=kubexpose-nginx -o=jsonpath='{.items[0].metadata.name}') -- curl http://localhost:4040/api/tunnels

You will get a JSON output such as

    {"tunnels":[{"name":"command_line","uri":"/api/tunnels/command_line","public_url":"https://af596a5c.ngrok.io","proto":"https","config":{"addr":"http://nginx-service-1:80","inspect":true},"metrics":{"conns":{"count":0,"gauge":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0},"http":{"count":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0}}},{"name":"command_line (http)","uri":"/api/tunnels/command_line%20%28http%29","public_url":"http://af596a5c.ngrok.io","proto":"http","config":{"addr":"http://nginx-service-1:80","inspect":true},"metrics":{"conns":{"count":0,"gauge":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0},"http":{"count":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0}}}],"uri":"/api/tunnels"}

Copy the value of `public_url` (`https://af596a5c.ngrok.io` in this case) and paste in in a browser - you should see the nginx landing page.