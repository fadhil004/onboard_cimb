/**
 * scenarios/transaction-service.test.js
 *
 * Performance test untuk transaction-service (port 8082).
 * Total durasi maksimal: 3 menit.
 *
 * Endpoints:
 *   POST /transfers-intrabank
 *   GET  /accounts/{id}/transactions
 *
 * Run:
 *   k6 run scenarios/transaction-service.test.js
 *   k6 run --env STAGE=smoke scenarios/transaction-service.test.js
 */

import http from 'k6/http';
import { sleep, group, fail } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { uuidv4, snapHeaders, checkOK, checkTransfer, randInt, nowISO } from '../helpers/utils.js';

// ─── Config ───────────────────────────────────────────────────────────────────
const ACCOUNT_URL     = __ENV.ACCOUNT_URL     || 'http://localhost:8081';
const TRANSACTION_URL = __ENV.TRANSACTION_URL || 'http://localhost:8082';
const STAGE           = __ENV.STAGE || 'full';

// ─── Custom metrics ───────────────────────────────────────────────────────────
const transferDuration  = new Trend('transfer_duration',          true);
const getTxDuration     = new Trend('get_transactions_duration',  true);
const transferErrors    = new Rate('transfer_errors');
const transferSuccess   = new Counter('transfer_success');
const fraudRejected     = new Counter('transfer_fraud_rejected');
const idempotencyHits   = new Counter('transfer_idempotency_hits');

// ─── Stages (total ≤ 3 menit) ────────────────────────────────────────────────
const stages = {
  smoke: [
    { duration: '30s', target: 1 },
  ],
  full: [
    { duration: '30s', target: 5  }, // ramp-up
    { duration: '90s', target: 20 }, // sustained (gRPC fan-out ke 2 service)
    { duration: '30s', target: 0  }, // ramp-down
  ],
};

export const options = {
  stages: stages[STAGE] || stages.full,
  thresholds: {
    http_req_duration:           ['p(95)<1500', 'p(99)<3000'],
    http_req_failed:             ['rate<0.03'],
    checks:                      ['rate>0.97'],
    transfer_duration:           ['p(95)<1500', 'p(99)<3000'],
    transfer_errors:             ['rate<0.03'],
    get_transactions_duration:   ['p(95)<500'],
  },
  tags: { service: 'transaction' },
};

// ─── Helper ───────────────────────────────────────────────────────────────────
function createFundedAccount(name) {
  const createRes = http.post(
    `${ACCOUNT_URL}/registration-account-creation`,
    JSON.stringify({
      partnerReferenceNo: `setup-${uuidv4()}`,
      name,
      phoneNo:     `08${randInt(100000000, 999999999)}`,
      email:       `${name.replace(/\s/g, '')}@k6test.id`,
      countryCode: 'ID',
      customerId:  `CUST-${uuidv4().substring(0, 8)}`,
      deviceInfo:  { os: 'Linux', osVersion: '5.x', model: 'CI', manufacturer: 'K6' },
    }),
    { headers: snapHeaders(uuidv4()) }
  );

  if (createRes.status !== 200) {
    console.error(`[setup] create failed: ${createRes.status} ${createRes.body}`);
    return null;
  }

  const account = JSON.parse(createRes.body);

  http.post(
    `${ACCOUNT_URL}/balance/deposit`,
    JSON.stringify({ accountNumber: account.accountNumber, amount: 999999999, remark: 'k6-setup' }),
    { headers: snapHeaders(uuidv4()) }
  );

  console.log(`[setup] ${account.accountNumber} (${account.accountId})`);
  return { id: account.accountId, accountNumber: account.accountNumber };
}

// ─── Setup: buat pool 5 akun sebelum VU mulai ────────────────────────────────
export function setup() {
  console.log('[setup] provisioning accounts...');
  const pool = [];
  for (let i = 0; i < 5; i++) {
    const acc = createFundedAccount(`K6 Account ${i + 1}`);
    if (acc) pool.push(acc);
    sleep(0.2);
  }
  if (pool.length < 2) fail(`[setup] not enough accounts: ${pool.length}`);
  console.log(`[setup] pool ready: ${pool.length} accounts`);
  return { pool };
}

// ─── Main VU ─────────────────────────────────────────────────────────────────
export default function ({ pool }) {
  if (!pool || pool.length < 2) { sleep(1); return; }

  let srcIdx = randInt(0, pool.length - 1);
  let dstIdx = randInt(0, pool.length - 1);
  while (dstIdx === srcIdx) dstIdx = randInt(0, pool.length - 1);

  const source = pool[srcIdx];
  const dest   = pool[dstIdx];

  // 1. Transfer
  group('POST /transfers-intrabank', () => {
    const externalID = uuidv4();
    const payload    = JSON.stringify({
      partnerReferenceNo:   `k6-${uuidv4()}`,
      sourceAccountNo:      source.accountNumber,
      beneficiaryAccountNo: dest.accountNumber,
      amount:               { value: String(randInt(1000, 50000)), currency: 'IDR' },
      currency:             'IDR',
      transactionDate:      nowISO(),
      remark:               `k6-vu${__VU}-iter${__ITER}`,
      feeType:              'OUR',
      customerReference:    `K6-${__VU}-${__ITER}`,
      originatorInfos: [{
        originatorCustomerNo:   source.accountNumber,
        originatorCustomerName: 'K6 Account',
        originatorBankCode:     'K6BANK',
      }],
      additionalInfo: { deviceId: `k6-vu-${__VU}`, channel: 'API' },
    });

    const res = http.post(
      `${TRANSACTION_URL}/transfers-intrabank`,
      payload,
      { headers: snapHeaders(externalID), tags: { endpoint: 'transfer' } }
    );

    transferDuration.add(res.timings.duration);

    if (res.status === 200) {
      transferSuccess.add(1);
      checkTransfer(res, 'transfer');
    } else if (res.status === 403) {
      try {
        const body = JSON.parse(res.body);
        if (body.responseCode?.includes('03') || body.responseCode?.includes('06')) {
          fraudRejected.add(1);
        }
      } catch (_) {}
    } else if (res.status === 409) {
      idempotencyHits.add(1);
    } else {
      transferErrors.add(1);
    }

    // Idempotency replay — 10% iterasi
    if (Math.random() < 0.1) {
      const replay = http.post(
        `${TRANSACTION_URL}/transfers-intrabank`,
        payload,
        { headers: snapHeaders(externalID), tags: { endpoint: 'transfer_replay' } }
      );
      if (replay.status === 200) idempotencyHits.add(1);
    }
  });

  sleep(0.5);

  // 2. Get transactions
  group('GET /accounts/{id}/transactions', () => {
    if (!source.id) return;
    const res = http.get(
      `${TRANSACTION_URL}/accounts/${source.id}/transactions`,
      { tags: { endpoint: 'get_transactions' } }
    );
    getTxDuration.add(res.timings.duration);
    checkOK(res, 'get-transactions');
  });

  sleep(1);
}

export function teardown({ pool }) {
  console.log(`[transaction-service] done. pool: ${pool?.length} accounts`);
}
