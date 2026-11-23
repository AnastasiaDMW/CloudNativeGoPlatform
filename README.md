# CloudNativeGoPlatform

**Cloud-Native Go REST API Platform with PostgreSQL, Kubernetes, Helm, Monitoring and Distributed Tracing**

A cloud-native backend platform built with Go and PostgreSQL, deployed on Kubernetes (k3s). The project demonstrates containerization, database high availability, automated backups, infrastructure management with Helm, monitoring and observability.

## Architecture

The platform includes the following components:

| Component | Purpose |
|---|---|
| Go | REST API and business logic |
| PostgreSQL 18 | Relational database |
| CloudNativePG | PostgreSQL cluster management and replication |
| RustFS | S3-compatible object storage for database backups |
| Barman Cloud Plugin | PostgreSQL backup and WAL archiving |
| Kubernetes (k3s) | Container orchestration |
| Helm | Application deployment and configuration |
| Prometheus | Metrics collection |
| Grafana | Monitoring dashboards |
| OpenTelemetry Collector | Telemetry collection and processing |
| Elasticsearch | Log storage |
| Kibana | Log visualization and search |
| Jaeger | Distributed tracing |
| Docker | Application containerization |


## REST API

The application provides CRUD operations for users.

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/user` | Retrieve all users |
| GET | `/api/user/{id}` | Retrieve a user by ID |
| POST | `/api/newuser` | Create a user |
| PUT | `/api/user/{id}` | Update a user |
| DELETE | `/api/user/{id}` | Delete a user |
| GET | `/healthz` | Application health check |
| GET | `/readyz` | Database readiness check |
| GET | `/metrics` | Prometheus metrics |

## Requirements

For the Kubernetes deployment:

- Ubuntu 24.04 LTS
- Multipass (recommended for creating an isolated Ubuntu VM)
- Docker
- Kubernetes (k3s)
- Helm
- Git

The demonstrated environment uses an Ubuntu VM with approximately 4 CPU cores, 8 GB RAM and 40 GB disk space. Resource requirements depend on the enabled services.

## 1. Create an Ubuntu VM with Multipass

Multipass is recommended for running the project in an isolated Ubuntu environment without installing Kubernetes directly on your host operating system.

Install Multipass:

https://multipass.run/

Create a VM:

```bash
multipass launch 24.04 \
  --name k3s-proj \
  --cpus 4 \
  --memory 8G \
  --disk 40G
```

Connect to the VM:

```bash
multipass shell k3s-proj
```

## 2. Install Kubernetes (k3s)

Inside the Ubuntu VM:

```bash
curl -sfL https://get.k3s.io | sh -
```

Configure kubectl:

```bash
export KUBECONFIG=/home/ubuntu/.kube/config
```

Verify the cluster:

```bash
kubectl get nodes
kubectl get pods -A
```

## 3. Clone the repository

Inside the VM:

```bash
git clone https://github.com/AnastasiaDMW/CloudNativeGoPlatform.git
cd CloudNativeGoPlatform
```

## 4. Build the Go application

On the host machine, build the Docker image for the VM architecture.

For an Apple Silicon Mac:

```bash
docker build --platform linux/arm64 -t go-postgres:dev .
```

Save the image:

```bash
docker save -o go-postgres-dev.tar go-postgres:dev
```

Transfer it to the Multipass VM:

```bash
multipass transfer go-postgres-dev.tar k3s-proj:/home/ubuntu/
```

Inside the VM, import the image into k3s containerd:

```bash
sudo k3s ctr images import /home/ubuntu/go-postgres-dev.tar
```

## 5. Configure PostgreSQL

Before deploying the API, prepare the PostgreSQL environment:

1. Install the CloudNativePG operator.
2. Deploy a PostgreSQL cluster with a primary instance and a replica.
3. Configure the Barman Cloud Plugin.
4. Deploy RustFS and configure the S3-compatible backup bucket.
5. Verify that the Kubernetes Secret `app-db-app` exists in the application namespace.

The Go API expects the database connection string in the `uri` key of the `app-db-app` Secret.

**Do not commit database credentials or `.env` files to GitHub.**

## 6. Deploy the Go API with Helm

From the repository root:

```bash
helm lint helm/go-postgres
```

Render the Kubernetes manifests:

```bash
helm template go-postgres helm/go-postgres \
  --namespace default
```

Deploy the application:

```bash
helm upgrade --install go-postgres \
  helm/go-postgres \
  --namespace default \
  --wait
```

Check the deployment:

```bash
kubectl get pods -l app=go-postgres
kubectl get services go-postgres
helm list -n default
```

The Helm chart deploys two Go API replicas by default.

## 7. Monitoring and Observability

The project includes an observability stack:

- Prometheus collects application metrics.
- Grafana visualizes metrics through dashboards.
- OpenTelemetry Collector receives and processes telemetry.
- Elasticsearch stores application logs.
- Kibana provides log search and visualization.
- Jaeger provides distributed tracing.

Useful application metrics include:

- `http_requests_total`
- `http_errors_total`
- `http_request_duration_seconds`
- `http_requests_in_flight`

## 8. Local Development with Docker Compose

For a simpler local development environment, Docker Compose is also provided.

Start the services:

```bash
docker compose up --build -d
```

Check running containers:

```bash
docker compose ps
```

View application logs:

```bash
docker compose logs -f app
```

Stop the environment:

```bash
docker compose down
```

**Note:** The supplied Compose configuration is intended for local development and demonstration, not production use. Review its example credentials and security settings before exposing services.

## 9. Database Backup and Recovery

The Kubernetes deployment uses CloudNativePG and the Barman Cloud Plugin to manage PostgreSQL backups.

The backup architecture includes:

- Base backups stored in RustFS.
- WAL archiving.
- Scheduled backups.
- Manual backups.
- Database recovery using a backup.

The backup destination is an S3-compatible RustFS bucket.

## 10. Security

Security practices for this project include:

- Keeping credentials outside source code.
- Using Kubernetes Secrets for database connection settings.
- Excluding `.env` files and generated binaries from Git.
- Adding automated secret detection to GitHub Actions with Gitleaks.

## Learning Objectives

This project demonstrates practical experience with:

- Go backend development and REST API design.
- PostgreSQL administration and replication.
- Kubernetes workload orchestration.
- Helm charts and application deployment.
- Database backup and recovery.
- Docker image building and container management.
- Metrics, centralized logging and distributed tracing.
- Infrastructure automation and CI/CD security checks.

---

**Status:** Personal cloud-native infrastructure and backend engineering project.