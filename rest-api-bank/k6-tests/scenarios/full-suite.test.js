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

import http from 'k6/http';
import { sleep, group } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';
import { uuidv4, snapHeaders, checkOK, checkCreated, checkTransfer, randInt, pick, nowISO } from '../helpers/utils.js';

// ─── Config ───────────────────────────────────────────────────────────────────
const ACCOUNT_URL     = __ENV.ACCOUNT_URL     || 'http://localhost:8081';
const TRANSACTION_URL = __ENV.TRANSACTION_URL || 'http://localhost:8082';
const STAGE           = __ENV.STAGE || 'full';

// ─── Custom metrics ───────────────────────────────────────────────────────────
const accountOpsDuration = new Trend('account_ops_duration', true);
const transferDuration   = new Trend('transfer_duration',    true);
const transferErrors     = new Rate('transfer_errors');
const transferSuccess    = new Counter('transfer_success');
const accountErrors      = new Rate('account_errors');

// ─── Scenario presets (total ≤ 3 menit per scenario) ─────────────────────────
const presets = {
  smoke: {
    account_ops: {
      executor: 'constant-vus', vus: 1, duration: '30s',
      gracefulStop: '5s', exec: 'accountOps',
    },
    transfer_ops: {
      executor: 'constant-vus', vus: 1, duration: '30s',
      startTime: '5s', gracefulStop: '5s', exec: 'transferOps',
    },
  },

  full: {
    // account-service: ramp 0→20 → sustain 90s → ramp down (total ~3m)
    account_ops: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 20 }, // ramp-up
        { duration: '90s', target: 20 }, // sustained
        { duration: '30s', target: 0  }, // ramp-down
      ],
      gracefulRampDown: '10s',
      exec: 'accountOps',
    },

    // transaction-service: mulai 10s setelah setup selesai, total ~3m
    transfer_ops: {
      executor: 'ramping-vus',
      startVUs: 0,
      startTime: '10s',
      stages: [
        { duration: '30s', target: 10 }, // ramp-up
        { duration: '90s', target: 10 }, // sustained — gRPC fan-out ke 2 service
        { duration: '30s', target: 0  }, // ramp-down
      ],
      gracefulRampDown: '10s',
      exec: 'transferOps',
    },
  },
};

export const options = {
  scenarios: presets[STAGE] || presets.full,
  thresholds: {
    http_req_duration:    ['p(95)<1500', 'p(99)<3000'],
    http_req_failed:      ['rate<0.03'],
    checks:               ['rate>0.97'],
    account_ops_duration: ['p(95)<500',  'p(99)<1000'],
    transfer_duration:    ['p(95)<1500', 'p(99)<3000'],
    transfer_errors:      ['rate<0.03'],
    account_errors:       ['rate<0.02'],
  },
};

// ─── Setup: buat pool akun sebelum VU mulai ───────────────────────────────────
function createAndFund(name) {
  const res = http.post(
    `${ACCOUNT_URL}/registration-account-creation`,
    JSON.stringify({
      partnerReferenceNo: `setup-${uuidv4()}`,
      name,
      phoneNo:     `08${randInt(100000000, 999999999)}`,
      email:       `${name.replace(/\s+/g, '').toLowerCase()}@k6.id`,
      countryCode: 'ID',
      customerId:  `CUST-${uuidv4().substring(0, 8)}`,
      deviceInfo:  { os: 'Linux', osVersion: '5.x', model: 'CI', manufacturer: 'K6' },
    }),
    { headers: snapHeaders(uuidv4()) }
  );
  if (res.status !== 200) return null;

  const account = JSON.parse(res.body);
  http.post(
    `${ACCOUNT_URL}/balance/deposit`,
    JSON.stringify({ accountNumber: account.accountNumber, amount: 999999999, remark: 'k6-setup' }),
    { headers: snapHeaders(uuidv4()) }
  );
  console.log(`  → ${account.accountNumber}`);
  return { id: account.accountId, accountNumber: account.accountNumber };
}

export function setup() {
  console.log('[setup] creating account pool (6 accounts)...');
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
  if (!acc) { sleep(1); return; }

  group('GET /accounts', () => {
    const res = http.get(`${ACCOUNT_URL}/accounts`, { tags: { endpoint: 'get_all' } });
    accountOpsDuration.add(res.timings.duration);
    accountErrors.add(!checkOK(res, 'get-all'));
  });

  sleep(0.2);

  group('GET /accounts/{id}', () => {
    const res = http.get(`${ACCOUNT_URL}/accounts/${acc.id}`, { tags: { endpoint: 'get_by_id' } });
    accountOpsDuration.add(res.timings.duration);
    accountErrors.add(!checkOK(res, 'get-by-id'));
  });

  sleep(0.2);

  group('POST /balance/deposit', () => {
    const res = http.post(
      `${ACCOUNT_URL}/balance/deposit`,
      JSON.stringify({ accountNumber: acc.accountNumber, amount: randInt(10000, 500000), remark: 'k6' }),
      { headers: snapHeaders(uuidv4()), tags: { endpoint: 'deposit' } }
    );
    accountOpsDuration.add(res.timings.duration);
    accountErrors.add(res.status !== 200);
  });

  sleep(0.2);

  group('POST /balance/withdraw', () => {
    const res = http.post(
      `${ACCOUNT_URL}/balance/withdraw`,
      JSON.stringify({ accountNumber: acc.accountNumber, amount: randInt(1000, 5000), remark: 'k6' }),
      { headers: snapHeaders(uuidv4()), tags: { endpoint: 'withdraw' } }
    );
    accountOpsDuration.add(res.timings.duration);
    // 403 = insufficient funds — expected
    accountErrors.add(![200, 403].includes(res.status));
  });

  sleep(1);
}

// ─── Executor B: transferOps ──────────────────────────────────────────────────
export function transferOps({ pool }) {
  if (!pool || pool.length < 2) { sleep(1); return; }

  let srcIdx = randInt(0, pool.length - 1);
  let dstIdx = randInt(0, pool.length - 1);
  while (dstIdx === srcIdx) dstIdx = randInt(0, pool.length - 1);

  const src = pool[srcIdx];
  const dst = pool[dstIdx];

  group('POST /transfers-intrabank', () => {
    const externalID = uuidv4();
    const res = http.post(
      `${TRANSACTION_URL}/transfers-intrabank`,
      JSON.stringify({
        partnerReferenceNo:   `k6-${uuidv4()}`,
        sourceAccountNo:      src.accountNumber,
        beneficiaryAccountNo: dst.accountNumber,
        amount:               { value: String(randInt(1000, 50000)), currency: 'IDR' },
        currency:             'IDR',
        transactionDate:      nowISO(),
        remark:               `k6-suite-vu${__VU}`,
        feeType:              'OUR',
        customerReference:    `K6-${__VU}`,
        originatorInfos: [{
          originatorCustomerNo:   src.accountNumber,
          originatorCustomerName: 'K6 Suite',
          originatorBankCode:     'K6BANK',
        }],
        additionalInfo: { deviceId: `k6-${__VU}`, channel: 'API' },
      }),
      { headers: snapHeaders(externalID), tags: { endpoint: 'transfer' } }
    );

    transferDuration.add(res.timings.duration);

    if (res.status === 200) {
      transferSuccess.add(1);
      checkTransfer(res, 'transfer');
    } else if ([403, 409].includes(res.status)) {
      // business error — bukan failure
    } else {
      transferErrors.add(1);
    }
  });

  sleep(0.5);

  group('GET /accounts/{id}/transactions', () => {
    const res = http.get(
      `${TRANSACTION_URL}/accounts/${src.id}/transactions`,
      { tags: { endpoint: 'get_transactions' } }
    );
    checkOK(res, 'get-transactions');
  });

  sleep(1);
}

export function teardown({ pool }) {
  console.log(`[full-suite] done. pool: ${pool?.length} accounts`);
}
