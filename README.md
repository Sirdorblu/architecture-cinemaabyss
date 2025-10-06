# CinemaAbyss — services with Kafka, K8s, Ingress, and GHCR CI/CD


---

## What we deployed

### Services
- **Proxy (Go)** — proxy that routes traffic to monolith and the new microservices.  
  Path: `src/microservices/proxy`  
  Health: `GET /health`
- **Events Service (Go + kafka-go)** — accepts events and produces to Kafka.  
  Path: `src/microservices/events`  
  Endpoints:  
  - `POST /api/events/movie`  
  - `POST /api/events/user`  
  - `POST /api/events/payment`  
  Health: `GET /api/events/health`
- **Movies Service (Go)** — standalone microservice for the **/api/movies** domain.  
  Path: `src/microservices/movies`  
  Health: `GET /api/movies/health`
- **Monolith (Go)** — existing application.  
  Path: `src/monolith`  
  Health: `GET /api/users` (used as a probe)

### Data Layer
- **PostgreSQL** StatefulSet (with an init script)  
  Manifests: `src/kubernetes/postgres*.yaml`  
  Credentials are read from `cinemaabyss-config` (ConfigMap) and `cinemaabyss-secrets` (Secret).
- **Kafka + Zookeeper** for event streaming  
  Manifests: `src/kubernetes/kafka/kafka.yaml`  
  Topics created: `movie-events`, `user-events`, `payment-events`.

### Kubernetes Manifests
All manifests live in `src/kubernetes/`:
- `*-service.yaml` — Deployments + Services for **monolith**, **movies-service**, **events-service**, **proxy-service**.
- `kafka/` — Zookeeper + Kafka (StatefulSets + Services).
- `postgres*.yaml` — Postgres (StatefulSet + Service + init scripts).
- `ingress.yaml` — ingress routing to services.
- `configmap.yaml` / `secret.yaml` — DB settings and secrets.

> **Image pulls from GHCR**: each Deployment uses an image from `ghcr.io/sirdorblu/*`. Add
> ```yaml
> spec:
>   template:
>     spec:
>       imagePullSecrets:
>         - name: ghcr-creds  
> ```

---

## CI/CD (GitHub Actions, self‑hosted runner)

Workflow: `.github/workflows/docker-build-push.yml`

### Build job
- Logs in to GHCR using `GHCR_USERNAME` and `GHCR_TOKEN` repository secrets.
- Builds and pushes images with tags `latest` and `${{ github.sha }}` for:
  - `monolith`
  - `movies-service`
  - `events-service`
  - `proxy-service`

### Deploy job
- Ensures namespace exists and `kubectl apply -R -f src/kubernetes/` is run.
- Rolls Deployments to the new image tag based on `${{ github.sha }}`.
- Waits for rollout to complete.

> Push to the **`cinema`** branch triggers the pipeline and redeploys the cluster.

---

## Ingress and external access

### Ingress (inside Minikube)
`src/kubernetes/ingress.yaml` routes based on **Host** `213.183.51.80.nip.io`:
- `/api/movies` → `movies-service:8081`
- `/api/events` → `events-service:8082`
- `/` → `proxy-service:80`

### Host NGINX (outside cluster) → Minikube
On your host you configured NGINX to forward to Minikube with **Host** set to `213.183.51.80.nip.io` so the ingress rule matches:

```nginx
server {
    listen 80;
    server_name 213.183.51.80 213.183.51.80.nip.io;

    location / {
        proxy_set_header Host 213.183.51.80.nip.io;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_pass http://$MINI;   # $MINI = minikube ip
    }
}
```

> Result: you can open **http://213.183.51.80** in a browser/curl and requests will hit your ingress and then the services.

---

## How to verify everything works

### 1) Quick health checks (from your laptop/host)
```
## all proofs you can find in folder **proof_screens_for_HW** 
```


---

## Configuration details

### Image pull credentials
Create a pull secret for GHCR and reference it in **each** Deployment:

```bash
kubectl -n cinemaabyss create secret docker-registry ghcr-creds \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN" \
  --docker-email=""
```

Then in the Deployment spec:
```yaml
spec:
  template:
    spec:
      imagePullSecrets:
        - name: ghcr-creds
```
Then you can use helm like 

```bash
helm upgrade --install cinemaabyss ./helm -n cinemaabyss
```

Or use GitHub action by pushing changes or start it manualy and choose a method - helm or manifest
