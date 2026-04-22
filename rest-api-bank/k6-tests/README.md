# k6 Performance Tests — microservices-bank

Performance & load testing suite menggunakan [k6](https://k6.io) untuk:
- **account-service** (port `8081`)
- **transaction-service** (port `8082`)

---

## Struktur

```
k6-tests/
├── helpers/
│   ├── utils.js          # SNAP headers, UUID, check helpers, randInt
│   └── thresholds.js     # Reusable SLO threshold definitions
├── scenarios/
│   ├── account-service.test.js    # Test isolated account-service
│   ├── transaction-service.test.js # Test isolated transaction-service
│   └── full-suite.test.js         # End-to-end: kedua service + spike scenario
└── README.md
```

---

## Prasyarat

1. **Semua service harus berjalan** via docker compose:
   ```bash
   docker compose up -d
   ```

2. **Install k6** (pilih salah satu):
   ```bash
   # macOS
   brew install k6

   # Linux (Debian/Ubuntu)
   sudo gpg -k
   sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
       --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
   echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
       | sudo tee /etc/apt/sources.list.d/k6.list
   sudo apt-get update && sudo apt-get install k6

   # Docker
   docker run --rm -i grafana/k6 run - <scenarios/account-service.test.js
   ```

---

## Cara Menjalankan

### Quick smoke test (1 VU, 30s)
```bash
# Account service saja
k6 run --env STAGE=smoke scenarios/account-service.test.js

# Transaction service saja
k6 run --env STAGE=smoke scenarios/transaction-service.test.js

# Full suite
k6 run --env STAGE=smoke scenarios/full-suite.test.js
```

### Full load test
```bash
# Account service
k6 run scenarios/account-service.test.js

# Transaction service
k6 run scenarios/transaction-service.test.js

# Full suite (kedua service + spike scenario)
k6 run scenarios/full-suite.test.js
```

### Ramp-only (untuk profiling bertahap)
```bash
k6 run --env STAGE=ramp scenarios/full-suite.test.js
```

### Custom URL (jika port berbeda)
```bash
k6 run \
  --env ACCOUNT_URL=http://localhost:8081 \
  --env TRANSACTION_URL=http://localhost:8082 \
  scenarios/full-suite.test.js
```

### Simpan hasil ke JSON untuk analisis
```bash
mkdir -p results
k6 run --out json=results/full-suite.json scenarios/full-suite.test.js
```

### Kirim metrics ke Prometheus/Grafana (sudah ada di docker-compose)
```bash
k6 run \
  --out experimental-prometheus-rw \
  --env K6_PROMETHEUS_RW_SERVER_URL=http://localhost:9090/api/v1/write \
  scenarios/full-suite.test.js
```

---

## Skenario Load

### `account-service.test.js`

| Phase     | VUs    | Durasi | Tujuan                        |
|-----------|--------|--------|-------------------------------|
| Smoke     | 1      | 30s    | Sanity check semua endpoint   |
| Ramp-up   | 0→50   | 1m     | Temukan breaking point        |
| Sustained | 50     | 3m     | Ukur performa steady-state    |
| Spike     | 50→200 | 30s    | Simulasi traffic burst        |
| Recovery  | 200→10 | 1m     | Verify graceful recovery      |

**Endpoints yang diuji:**
- `POST /registration-account-creation`
- `GET  /accounts`
- `GET  /accounts/{id}`
- `POST /balance/deposit`
- `POST /balance/withdraw`

---

### `transaction-service.test.js`

| Phase     | VUs     | Durasi | Tujuan                              |
|-----------|---------|--------|-------------------------------------|
| Smoke     | 1       | 30s    | Verify setup + 1 transfer cycle     |
| Ramp-up   | 0→30    | 1m     | Gradual load                        |
| Sustained | 30      | 3m     | Ukur p95/p99 under realistic load   |
| Spike     | 30→150  | 30s    | Stress gRPC fan-out ke 2 service    |
| Recovery  | 150→5   | 1m     | Verify graceful degradation         |

**Endpoints yang diuji:**
- `POST /transfers-intrabank` (SNAP + idempotency replay test)
- `GET  /accounts/{id}/transactions`

**Special:** 10% iterasi melakukan **idempotency replay** (kirim ulang request yang sama) untuk memverifikasi Redis cache berfungsi.

---

### `full-suite.test.js` (Recommended)

Menjalankan 3 scenario secara paralel:

| Scenario      | Executor | Peak VUs | Catatan                              |
|---------------|----------|----------|--------------------------------------|
| account_ops   | ramping  | 60       | CRUD account + deposit/withdraw      |
| transfer_ops  | ramping  | 80       | Transfer + get transactions          |
| spike_test    | ramping  | 200      | Burst 200 VU selama 30s (menit ke-5) |

---

## SLO / Thresholds

| Metric                   | SLO                      |
|--------------------------|--------------------------|
| `http_req_duration` p95  | < 1500ms                 |
| `http_req_duration` p99  | < 3000ms                 |
| `http_req_failed`        | < 3%                     |
| `checks`                 | > 97%                    |
| `account_ops_duration` p95 | < 500ms               |
| `transfer_duration` p95  | < 1500ms (2 gRPC hops)   |
| `transfer_errors`        | < 3%                     |

---

## Metrics Kustom

| Metric                       | Type    | Deskripsi                              |
|------------------------------|---------|----------------------------------------|
| `account_create_duration`    | Trend   | Latency create account                 |
| `account_deposit_duration`   | Trend   | Latency deposit                        |
| `account_withdraw_duration`  | Trend   | Latency withdraw                       |
| `account_get_duration`       | Trend   | Latency GET account by ID              |
| `transfer_duration`          | Trend   | Latency transfer (end-to-end)          |
| `transfer_success`           | Counter | Jumlah transfer berhasil               |
| `transfer_errors`            | Rate    | Rate error transfer (bukan 200/403/409)|
| `transfer_fraud_rejected`    | Counter | Transfer ditolak fraud detection       |
| `transfer_idempotency_hits`  | Counter | Idempotency cache hit                  |
| `account_errors`             | Rate    | Rate error account ops                 |

---

## Catatan

- **403 Forbidden** pada transfer = expected jika fraud detection menolak atau balance habis — **bukan test failure**.
- **409 Conflict** = idempotency duplicate — **bukan test failure**.
- Setup membuat 8 akun dengan saldo `999.999.999` masing-masing agar transfer tidak kering.
- Fraud rule default: `RULE_MAX_AMOUNT=1000000` — transfer di atas 1 juta akan diblok. Test menggunakan amount `1000–100000` agar aman.
- Velocity rule: max 5 transfer ke destinasi yang sama dalam 5 menit. Pool 8 akun meminimalkan collision ini.
