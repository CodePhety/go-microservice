# Kubernetes manifests

This directory contains a minimal Kubernetes deployment for the user service.

## Deploy

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

## Notes

- The deployment uses a readiness probe and a liveness probe.
- Prometheus scraping is enabled via annotations.
- The service is exposed internally via a ClusterIP service.
- This is intentionally simple and suitable for interview practice.
