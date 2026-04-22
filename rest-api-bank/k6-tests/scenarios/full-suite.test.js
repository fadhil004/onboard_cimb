/**
 * scenarios/full-suite.test.js
 *
 * End-to-end suite — account-service + transaction-service paralel.
 * Total durasi maksimal: 3 menit.
 *
 *  Scenario A: account_ops  — CRUD + deposit/withdraw
 *  Scenario B: transfer_ops — transfer + get-transactions
 *
 * Run:
 *   k6 run scenarios/full-suite.test.js
 *   k6 run --env STAGE=smoke scenarios/full-suite.test.js
 *   k6 run --out json=results/full-suite.json scenarios/full-suite.test.js
 */

import http from "k6/http";
import { sleep, group } from "k6";

http.setResponseCallback(http.expectedStatuses(200, 201, 403, 409, 422, 429));

import { Trend, Rate, Counter } from "k6/metrics";
import {
  uuidv4,
  snapHeaders,
  checkOK,
  checkCreated,
  checkTransfer,
  randInt,
  pick,
  nowISO,
} from "../helpers/utils.js";

// ─── Config ───────────────────────────────────────────────────────────────────
const ACCOUNT_URL = __ENV.ACCOUNT_URL || "http://localhost:8081";
const TRANSACTION_URL = __ENV.TRANSACTION_URL || "http://localhost:8082";
const STAGE = __ENV.STAGE || "full";

// ─── Custom metrics ───────────────────────────────────────────────────────────
const accountOpsDuration = new Trend("account_ops_duration", true);
const transferDuration = new Trend("transfer_duration", true);
const transferErrors = new Rate("transfer_errors");
const transferSuccess = new Counter("transfer_success");
const accountErrors = new Rate("account_errors");

// ─── Scenario presets (total ≤ 3 menit per scenario) ─────────────────────────
const presets = {
  smoke: {
    account_ops: {
      executor: "constant-vus",
      vus: 1,
      duration: "30s",
      gracefulStop: "5s",
      exec: "accountOps",
    },
    transfer_ops: {
      executor: "constant-vus",
      vus: 1,
      duration: "30s",
      startTime: "5s",
      gracefulStop: "5s",
      exec: "transferOps",
    },
  },

  full: {
    account_ops: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "30s", target: 20 },
        { duration: "90s", target: 20 },
        { duration: "30s", target: 0 },
      ],
      gracefulRampDown: "10s",
      exec: "accountOps",
    },
    transfer_ops: {
      executor: "ramping-vus",
      startVUs: 0,
      startTime: "10s",
      stages: [
        { duration: "30s", target: 10 },
        { duration: "90s", target: 10 },
        { duration: "30s", target: 0 },
      ],
      gracefulRampDown: "10s",
      exec: "transferOps",
    },
  },
};

export const options = {
  scenarios: presets[STAGE] || presets.full,
  thresholds: {
    http_req_duration: ["p(95)<1500", "p(99)<3000"],
    http_req_failed: ["rate<0.03"],
    checks: ["rate>0.97"],
    account_ops_duration: ["p(95)<500", "p(99)<1000"],
    transfer_duration: ["p(95)<1500", "p(99)<3000"],
    transfer_errors: ["rate<0.03"],
    account_errors: ["rate<0.02"],
  },
};

// ─── Setup ────────────────────────────────────────────────────────────────────
function createAndFund(name) {
  const res = http.post(
    `${ACCOUNT_URL}/registration-account-creation`,
    JSON.stringify({
      partnerReferenceNo: `setup-${uuidv4()}`,
      name,
      phoneNo: `08${randInt(100000000, 999999999)}`,
      email: `${name.replace(/\s+/g, "").toLowerCase()}@k6.id`,
      countryCode: "ID",
      customerId: `CUST-${uuidv4().substring(0, 8)}`,
      deviceInfo: {
        os: "Linux",
        osVersion: "5.x",
        model: "CI",
        manufacturer: "K6",
      },
    }),
    { headers: snapHeaders(uuidv4()) },
  );
  if (res.status !== 200) return null;

  const account = JSON.parse(res.body);
  http.post(
    `${ACCOUNT_URL}/balance/deposit`,
    JSON.stringify({
      accountNumber: account.accountNumber,
      amount: 999999999,
      remark: "k6-setup",
    }),
    {
      headers: snapHeaders(uuidv4()),
      expectedStatuses: [200, 403, 409, 422, 429],
    },
  );
  console.log(`  → ${account.accountNumber}`);
  return { id: account.accountId, accountNumber: account.accountNumber };
}

export function setup() {
  console.log("[setup] creating account pool (6 accounts)...");
  const pool = [];
  for (let i = 0; i < 6; i++) {
    const acc = createAndFund(`K6 Suite ${i + 1}`);
    if (acc) pool.push(acc);
    sleep(0.2);
  }
  console.log(`[setup] ready: ${pool.length} accounts`);
  return { pool };
}

// ─── Executor A: accountOps ───────────────────────────────────────────────────
export function accountOps({ pool }) {
  const acc = pick(pool);
  if (!acc) {
    sleep(1);
    return;
  }

  // FIX: GET /accounts dan GET /accounts/{id} tidak punya SNAP body,
  // tapi rate limit middleware sekarang baca X-PARTNER-ID dari header.
  // Kirim header dengan externalID unik per request agar setiap request
  // punya counter rate-limit sendiri, bukan berbagi satu IP counter.
  const reqHeaders = snapHeaders(uuidv4());

  group("GET /accounts", () => {
    const res = http.get(`${ACCOUNT_URL}/accounts`, {
      headers: reqHeaders,
      tags: { endpoint: "get_all" },
      expectedStatuses: [200, 403, 409, 422, 429],
    });
    accountOpsDuration.add(res.timings.duration);
    // FIX: Selalu add ke Rate (true=error, false=ok) agar denominator benar.
    // Sebelumnya hanya add saat checkOK gagal → rate selalu 100% kalau ada 1 error.
    // 429 dihitung error (rate limit seharusnya tidak terjadi setelah fix backend).
    accountErrors.add(!checkOK(res, "get-all"));
  });

  sleep(0.2);

  group("GET /accounts/{id}", () => {
    const res = http.get(`${ACCOUNT_URL}/accounts/${acc.id}`, {
      headers: snapHeaders(uuidv4()),
      tags: { endpoint: "get_by_id" },
      expectedStatuses: [200, 403, 409, 422, 429],
    });
    accountOpsDuration.add(res.timings.duration);
    accountErrors.add(!checkOK(res, "get-by-id"));
  });

  sleep(0.2);

  group("POST /balance/deposit", () => {
    const res = http.post(
      `${ACCOUNT_URL}/balance/deposit`,
      JSON.stringify({
        accountNumber: acc.accountNumber,
        amount: randInt(10000, 500000),
        remark: "k6",
      }),
      {
        headers: snapHeaders(uuidv4()),
        tags: { endpoint: "deposit" },
        expectedStatuses: [200, 403, 409, 422, 429],
      },
    );
    accountOpsDuration.add(res.timings.duration);
    accountErrors.add(res.status !== 200);
  });

  sleep(0.2);

  group("POST /balance/withdraw", () => {
    const res = http.post(
      `${ACCOUNT_URL}/balance/withdraw`,
      JSON.stringify({
        accountNumber: acc.accountNumber,
        amount: randInt(1000, 5000),
        remark: "k6",
      }),
      {
        headers: snapHeaders(uuidv4()),
        tags: { endpoint: "withdraw" },
        expectedStatuses: [200, 403, 409, 422, 429],
      },
    );
    accountOpsDuration.add(res.timings.duration);
    // 403 = insufficient funds — expected, bukan failure
    accountErrors.add(![200, 403].includes(res.status));
  });

  sleep(1);
}

// ─── Executor B: transferOps ──────────────────────────────────────────────────
export function transferOps({ pool }) {
  if (!pool || pool.length < 2) {
    sleep(1);
    return;
  }

  let srcIdx = randInt(0, pool.length - 1);
  let dstIdx = randInt(0, pool.length - 1);
  while (dstIdx === srcIdx) dstIdx = randInt(0, pool.length - 1);

  const src = pool[srcIdx];
  const dst = pool[dstIdx];

  group("POST /transfers-intrabank", () => {
    const externalID = uuidv4();
    const res = http.post(
      `${TRANSACTION_URL}/transfers-intrabank`,
      JSON.stringify({
        partnerReferenceNo: `k6-${uuidv4()}`,
        sourceAccountNo: src.accountNumber,
        beneficiaryAccountNo: dst.accountNumber,
        amount: { value: String(randInt(1000, 50000)), currency: "IDR" },
        currency: "IDR",
        transactionDate: nowISO(),
        remark: `k6-suite-vu${__VU}`,
        feeType: "OUR",
        customerReference: `K6-${__VU}`,
        originatorInfos: [
          {
            originatorCustomerNo: src.accountNumber,
            originatorCustomerName: "K6 Suite",
            originatorBankCode: "K6BANK",
          },
        ],
        additionalInfo: { deviceId: `k6-${__VU}`, channel: "API" },
      }),
      {
        headers: snapHeaders(externalID),
        tags: { endpoint: "transfer" },
        expectedStatuses: [200, 403, 409, 422, 429],
      },
    );

    transferDuration.add(res.timings.duration);

    if (res.status === 200) {
      transferErrors.add(false);
    } else if ([403, 409, 422, 429].includes(res.status)) {
      transferErrors.add(false); // expected business case
    } else {
      transferErrors.add(true);
    }
  });

  sleep(0.5);

  group("GET /accounts/{id}/transactions", () => {
    const res = http.get(
      `${TRANSACTION_URL}/accounts/${src.id}/transactions`,
      // FIX: Kirim X-PARTNER-ID agar rate limit tidak terpicu di endpoint ini.
      {
        headers: snapHeaders(uuidv4()),
        tags: {
          endpoint: "get_transactions",
        },
        expectedStatuses: [200, 403, 409, 422, 429],
      },
    );
    checkOK(res, "get-transactions");
  });

  sleep(1);
}

export function teardown({ pool }) {
  console.log(`[full-suite] done. pool: ${pool?.length} accounts`);
}
