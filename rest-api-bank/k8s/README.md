# Kubernetes Deployment — microservices-bank

## Struktur direktori

```
k8s/
├── deploy.sh                        ← script deploy utama
├── namespace.yaml
├── config/
│   ├── configmaps.yaml              ← env config semua service
│   └── secrets.yaml                 ← kredensial DB (base64)
├── infrastructure/
│   ├── postgres.yaml                ← 3 database (account, transaction, notification)
│   ├── redis.yaml
│   └── kafka.yaml                   ← Kafka KRaft + Kafka UI
├── services/
│   ├── account-service.yaml         ← HTTP :30081 + gRPC :50051
│   ├── fraud-detection-service.yaml ← gRPC only (ClusterIP)
│   ├── transaction-service.yaml     ← HTTP :30082
│   └── notification-service.yaml    ← HTTP :30083
└── observability/
    └── observability.yaml           ← Prometheus, Loki, Tempo, Grafana, Fluent Bit
```

## Deploy

```bash

# 1. Start minikube
minikube start --cpus=4 --memory=6144 --disk-size=20g --driver=docker
minikube status
# 2. Arahkan docker ke minikube (biar image kebuild lgsung ke dalam cluster)
minikube docker-env --shell powershell | Invoke-Expression
# 3. Build semua Docker image
docker build -t account-service:latest -f account-service/Dockerfile .
docker build -t transaction-service:latest -f transaction-service/Dockerfile .
docker build -t notification-service:latest -f notification-service/Dockerfile .
docker build -t fraud-detection-service:latest ./fraud-detection-service

docker images
# 4. Apply namespace
kubectl apply -f k8s/namespace.yaml
# 5. Config (secrets dan configmap)
kubectl apply -f k8s/config/secrets.yaml
kubectl apply -f k8s/config/configmaps.yaml
# 6. Infrastructure
kubectl apply -f k8s/infrastructure/redis.yaml
kubectl apply -f k8s/infrastructure/postgres.yaml
kubectl apply -f k8s/infrastructure/kafka.yaml
# 7. Tunggu semua DB & Redis ready
kubectl rollout status deployment/redis -n bank
kubectl rollout status deployment/account-db -n bank
kubectl rollout status deployment/transaction-db -n bank
kubectl rollout status deployment/notification-db -n bank
# 8. Tunggu kafka (biasanya lama)
kubectl rollout status deployment/kafka -n bank --timeout=180s
# kalau stuck
kubectl get pods -n bank
kubectl logs <nama-pod-kafka> -n bank
# 9. Observability
kubectl apply -f k8s/observability/observability.yaml
# 10. Deploy core services (sesuai urutan)
kubectl apply -f k8s/services/fraud-detection-service.yaml
kubectl apply -f k8s/services/account-service.yaml
# 11. Tunggu ready
kubectl rollout status deployment/fraud-detection-service -n bank
kubectl rollout status deployment/account-service -n bank
# 12. Deploy sisa
kubectl apply -f k8s/services/transaction-service.yaml
kubectl apply -f k8s/services/notification-service.yaml
# 13. Tunggu lagi
kubectl rollout status deployment/transaction-service -n bank
kubectl rollout status deployment/notification-service -n bank
# 14. Cek semua pod
kubectl get pods -n bank
# 15. Akses service
minikube ip

```

1. Start Minikube (4 CPU, 6GB RAM)
2. Build semua Docker image ke daemon Minikube
3. Apply manifest sesuai urutan dependency
4. Tunggu setiap layer siap sebelum lanjut
5. Print URL akses

## Port mapping (NodePort)

| Service              | NodePort | Akses                         |
| -------------------- | -------- | ----------------------------- |
| account-service HTTP | 30081    | `http://$(minikube ip):30081` |
| account-service gRPC | 50051    | via ClusterIP (internal)      |
| transaction-service  | 30082    | `http://$(minikube ip):30082` |
| notification-service | 30083    | `http://$(minikube ip):30083` |
| kafka-ui             | 30090    | `http://$(minikube ip):30090` |
| prometheus           | 30090    | `http://$(minikube ip):30090` |
| grafana              | 30300    | `http://$(minikube ip):30300` |

## Jalankan k6 setelah deploy

```bash
MINIKUBE_IP=$(minikube ip)

k6 run \
  --env ACCOUNT_URL=http://${MINIKUBE_IP}:30081 \
  --env TRANSACTION_URL=http://${MINIKUBE_IP}:30082 \
  k6-tests/scenarios/full-suite.test.js
```

## Perintah umum

```bash
# Lihat semua pod
kubectl get pods -n bank

# Lihat logs service tertentu
kubectl logs -n bank deployment/account-service -f
kubectl logs -n bank deployment/fraud-detection-service -f

# Masuk ke pod
kubectl exec -it -n bank deployment/account-service -- sh

# Restart deployment (misal setelah rebuild image)
eval $(minikube docker-env)
docker build -t account-service:latest -f account-service/Dockerfile .
kubectl rollout restart deployment/account-service -n bank

# Hapus semua (cleanup)
kubectl delete namespace bank
minikube stop
```

## Catatan penting

**`imagePullPolicy: Never`** — semua service memakai image lokal yang di-build
ke dalam daemon Docker Minikube. Jangan lupa `eval $(minikube docker-env)`
sebelum `docker build`, kalau tidak image tidak akan ditemukan saat pod start.

**Urutan dependency:**

```
postgres + redis + kafka
    ↓
fraud-detection-service + account-service
    ↓
transaction-service + notification-service
```

**Kafka membutuhkan waktu** ~60-90 detik untuk siap. Script sudah tunggu dengan
timeout 180 detik. Kalau timeout, cek dengan `kubectl logs -n bank deployment/kafka`.
