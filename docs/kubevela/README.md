
# Deploying Fern Application on k3d with KubeVela and Cloud Native PostgreSQL

This README provides detailed instructions on deploying the Fern application in a Kubernetes environment using k3d, KubeVela, and Cloud Native PostgreSQL (cnpg). The process includes setting up a k3d cluster, installing KubeVela, deploying Cloud Native PostgreSQL using Helm, and finally, deploying the Fern application as defined in `vela.yaml`. Additionally, this guide covers the installation of custom ComponentDefinitions for an enhanced gateway trait and Cloud Native PostgreSQL.

## Prerequisites

Before starting, ensure you have the following tools installed on your system:

- Docker
- k3d
- kubectl
- Helm
- KubeVela CLI

## Step 1: Create a k3d Cluster

Create a new k3d cluster with the following command:

```bash
k3d cluster create my-k3d-cluster --port "8080:8080@loadbalancer" --agents 3
```

This command will set up a new Kubernetes cluster named `my-k3d-cluster` running in Docker.

## Step 2: Install KubeVela

Install KubeVela in your k3d cluster using Helm:

```bash
helm repo add kubevela https://charts.kubevela.net/core
helm repo update
helm install --create-namespace -n vela-system kubevela kubevela/vela-core
```

Confirm the installation by checking the deployed pods:

```bash
kubectl get pods -n vela-system
```

## Step 3: Install Cloud Native PostgreSQL

Add the Cloud Native PostgreSQL Helm repository and install it:

```bash
helm repo add cnpg https://cloudnative-pg.github.io/charts
helm repo update
helm install cnpg cnpg/cloud-native-pg
```

## Step 4: Install Custom ComponentDefinitions

Before deploying the Fern application, add the following custom ComponentDefinitions:

1. **Gateway Component (gateway.cue):**
   
   This updates the existing gateway trait to support the service type LoadBalancer.

   ```bash
   kubectl apply -f gateway.cue
   ```

2. **Cloud Native PostgreSQL Component (cnpg.cue):**
   
   Introduces a new component definition for Cloud Native PostgreSQL.

   ```bash
   kubectl apply -f cnpg.cue
   ```

## Step 5: Deploy the Fern Application

Deploy your application using the provided `vela.yaml` in a namespace called fern:

```bash
kubectls creante ns fern
kubectl apply -f vela.yaml
```

## Verifying the Deployment

To check the status of your deployment, use:

```bash
kubectl get all -n fern
```

## Additional Notes

- Ensure Docker is running prior to initiating the k3d cluster.
- Customize `gateway.cue` and `cnpg.cue` according to your specific needs.
- Adjust `vela.yaml` to fit the configuration of your Fern application.

## Contributing

We welcome your contributions! Feel free to submit pull requests or open issues to enhance the documentation or deployment procedures.

---

For questions or feedback, please create an issue in this repository.

Thank you for using or contributing to this project!

