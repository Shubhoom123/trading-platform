# Kubernetes manifests

Runs the full platform on a local cluster (k3d or minikube). Manifests are
numbered so a plain `kubectl apply` in order brings the stack up cleanly:

```sh
# 1. Build the app images (from the repo root)
docker build -f matching-engine-cpp/Dockerfile -t trading/matching-engine:latest .
docker build -f api-service-java/Dockerfile     -t trading/api-service:latest .
docker build -f gateway-go/Dockerfile           -t trading/gateway:latest .

# 2. Make the images available to the cluster
#    k3d:      k3d image import trading/matching-engine:latest trading/api-service:latest trading/gateway:latest -c <cluster>
#    minikube: minikube image load trading/matching-engine:latest trading/api-service:latest trading/gateway:latest

# 3. Apply everything (namespace + config first, then infra, apps, observability)
kubectl apply -f infra/k8s/

# 4. Optional: load the Grafana dashboard as a ConfigMap
kubectl -n trading create configmap grafana-dashboards \
  --from-file=infra/grafana/dashboards/trading-platform.json
```

## Layout

| File | Contents |
| --- | --- |
| `00-namespace-and-config.yaml` | `trading` namespace, shared ConfigMap, demo Secret |
| `10-infra.yaml` | Postgres, Redis, Redpanda + a topic-creation Job |
| `20-apps.yaml` | Deployments + Services for engine, API, gateway |
| `30-observability.yaml` | Prometheus (inline scrape config) + Grafana |

## Notes

- Wiring (broker address, topic names, service URLs) lives once in
  `platform-config`; secrets (`JWT_SECRET`, DB password) in `platform-secrets`.
  The demo Secret is committed for convenience — replace it with a sealed or
  external secret in any real cluster.
- Postgres uses an `emptyDir`; swap in a PVC to persist across pod restarts.
- The gateway and Grafana are `NodePort` Services so you can reach them from the
  host; everything else is cluster-internal.
- Reach the gateway: `kubectl -n trading port-forward svc/gateway 8090:8090`.
  Grafana: `kubectl -n trading port-forward svc/grafana 3000:3000`.
