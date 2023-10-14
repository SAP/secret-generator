# Instructions for local development

Prerequisite: K8s cluster (kind, minikube) with cert-manager installed.

1. Deploy webhook definitions and according objects:
   ```bash
   # replace HOST_IP below with a non-loopback interface address of your desktop
   HOST_IP=1.2.3.4 envsubst < .local/k8s-resources.yaml | kubectl apply -f -
   ```

2. Extract the TLS server keypair:
   ```bash
   .local/getcerts.sh
   ```

Afterwards (if using vscode) it should be possible to start the operator with the included launch configuration.
