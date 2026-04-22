/**
 * scenarios/account-service.test.js
 *
 * Performance test untuk account-service (port 8081).
 * Total durasi maksimal: 3 menit.
 *
 * Endpoints:
 *   POST /registration-account-creation
 *   GET  /accounts
 *   GET  /accounts/{id}
 *   POST /balance/deposit
 *   POST /balance/withdraw
 *
 * Run:
 *   k6 run scenarios/account-service.test.js
 *   k6 run --env STAGE=smoke scenarios/account-service.test.js
 */

import http from 'k6/http';
import { sleep, group } from 'k6';
import { Trend, Rate } from 'k6/metrics';
import { uuidv4, snapHeaders, checkOK, checkCreated, randInt } from '../helpers/utils.js';

// ─── Config ───────────────────────────────────────────────────────────────────
const BASE_URL = __ENV.ACCOUNT_URL || 'http://localhost:8081';
const STAGE    = __ENV.STAGE || 'full';

// ─── Custom metrics ───────────────────────────────────────────────────────────
const createDuration   = new Trend('account_create_duration',   true);
const depositDuration  = new Trend('account_deposit_duration',  true);
const withdrawDuration = new Trend('account_withdraw_duration', true);
const getDuration      = new Trend('account_get_duration',      true);
const getAllDuration    = new Trend('account_get_all_duration',  true);
const createErrors     = new Rate('account_create_errors');
const balanceErrors    = new Rate('account_balance_errors');

// ─── Stages (total ≤ 3 menit) ────────────────────────────────────────────────
const stages = {
  // 30s — 1 VU, sanity only
  smoke: [
    { duration: '30s', target: 1 },
  ],
  // 3 menit: ramp → sustain → ramp down
  full: [
    { duration: '30s', target: 10 }, // ramp-up
    { duration: '90s', target: 30 }, // sustained load
    { duration: '30s', target: 0  }, // ramp-down
  ],
};

export const options = {
  stages: stages[STAGE] || stages.full,
  thresholds: {
    http_req_duration:        ['p(95)<500',  'p(99)<1000'],
    http_req_failed:          ['rate<0.02'],
    checks:                   ['rate>0.97'],
    account_create_duration:  ['p(95)<600'],
    account_deposit_duration: ['p(95)<400'],
    account_withdraw_duration:['p(95)<400'],
    account_get_duration:     ['p(95)<300'],
    account_get_all_duration: ['p(95)<500'],
  },
  tags: { service: 'account' },
};

// ─── VU-local state ───────────────────────────────────────────────────────────
let vuAccountID     = null;
let vuAccountNumber = null;

// ─── Setup ────────────────────────────────────────────────────────────────────
export function setup() {
  const res = http.post(
    `${BASE_URL}/registration-account-creation`,
    JSON.stringify({
      partnerReferenceNo: `setup-${uuidv4()}`,
      name:        'K6 Setup Account',
      phoneNo:     `08${randInt(100000000, 999999999)}`,
      email:       'k6setup@test.com',
      countryCode: 'ID',
      customerId:  'CUST-SETUP',
      deviceInfo:  { os: 'Linux', osVersion: '5.x', model: 'CI', manufacturer: 'K6' },
    }),
    { headers: snapHeaders(uuidv4()) }
  );

  if (res.status === 200) {
    const body = JSON.parse(res.body);
    return { seedAccountID: body.accountId || null, seedAccountNumber: body.accountNumber || null };
  }
  return { seedAccountID: null, seedAccountNumber: null };
}

// ─── Main VU ─────────────────────────────────────────────────────────────────
export default function (data) {

  // 1. Create account — hanya sekali per VU
  group('POST /registration-account-creation', () => {
    if (vuAccountID !== null) return;

    const res = http.post(
      `${BASE_URL}/registration-account-creation`,
      JSON.stringify({
        partnerReferenceNo: `ref-${uuidv4()}`,
        name:        `K6 User ${__VU}`,
        phoneNo:     `08${randInt(100000000, 999999999)}`,
        email:       `vu${__VU}@k6test.id`,
        countryCode: 'ID',
        customerId:  `CUST-${__VU}`,
        deviceInfo:  { os: 'Android', osVersion: '13', model: 'Pixel 7', manufacturer: 'Google' },
        additionalInfo: { channel: 'mobile' },
      }),
      { headers: snapHeaders(uuidv4()), tags: { endpoint: 'create_account' } }
    );

    createDuration.add(res.timings.duration);
    createErrors.add(!checkCreated(res, 'create-account'));

    if (res.status === 200) {
      try {
        const body    = JSON.parse(res.body);
        vuAccountID     = body.accountId;
        vuAccountNumber = body.accountNumber;
      } catch (_) {}
    }
  });

  sleep(0.3);

  // 2. Deposit
  group('POST /balance/deposit', () => {
    if (!vuAccountNumber) return;
    const res = http.post(
      `${BASE_URL}/balance/deposit`,
      JSON.stringify({ accountNumber: vuAccountNumber, amount: randInt(100000, 5000000), remark: `k6-vu${__VU}` }),
      { headers: snapHeaders(uuidv4()), tags: { endpoint: 'deposit' } }
    );
    depositDuration.add(res.timings.duration);
    balanceErrors.add(!checkCreated(res, 'deposit'));
  });

  sleep(0.2);

  // 3. Get by ID
  group('GET /accounts/{id}', () => {
    if (!vuAccountID) return;
    const res = http.get(
      `${BASE_URL}/accounts/${vuAccountID}`,
      { tags: { endpoint: 'get_by_id' } }
    );
    getDuration.add(res.timings.duration);
    checkOK(res, 'get-by-id');
  });

  sleep(0.2);

  // 4. Get all
  group('GET /accounts', () => {
    const res = http.get(`${BASE_URL}/accounts`, { tags: { endpoint: 'get_all' } });
    getAllDuration.add(res.timings.duration);
    checkOK(res, 'get-all');
  });

  sleep(0.2);

  // 5. Withdraw (amount kecil agar tidak habis)
  group('POST /balance/withdraw', () => {
    if (!vuAccountNumber) return;
    const res = http.post(
      `${BASE_URL}/balance/withdraw`,
      JSON.stringify({ accountNumber: vuAccountNumber, amount: randInt(1000, 5000), remark: `k6-vu${__VU}` }),
      { headers: snapHeaders(uuidv4()), tags: { endpoint: 'withdraw' } }
    );
    withdrawDuration.add(res.timings.duration);
    // 403 = insufficient funds — expected, bukan failure
    balanceErrors.add(![200, 403].includes(res.status));
  });

  sleep(1);
}

export function teardown(data) {
  console.log(`[account-service] done. seed: ${data.seedAccountNumber || 'N/A'}`);
}
