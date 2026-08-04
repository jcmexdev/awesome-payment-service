import http from 'k6/http';
import { check } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  scenarios: {
    contention: {
      executor: 'constant-vus',
      vus: 30, // 30 virtual users running concurrently
      duration: '5m', // flood test for 5 minutes
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'], // less than 5% HTTP errors allowed
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Setup runs once, creating test accounts with sufficient balance
export function setup() {
  const accounts = [];
  const numAccounts = 3; // Keep the number of accounts low to maximize lock contention per account!
  
  for (let i = 0; i < numAccounts; i++) {
    const payload = JSON.stringify({
      user_id: `k6_user_${i}_${Date.now()}`,
      currency: 'USD',
      balance: 5000000, // 50,000.00 USD (in cents) so balance is never insufficient
    });
    
    const params = {
      headers: {
        'Content-Type': 'application/json',
      },
    };
    
    const res = http.post(`${BASE_URL}/v1/accounts`, payload, params);
    
    if (res.status === 201) {
      const respBody = JSON.parse(res.body);
      const accId = respBody.data.id || respBody.data.ID;
      if (accId) {
        accounts.push(accId);
      }
    } else {
      console.log(`Failed to create account ${i}: Status ${res.status}, body: ${res.body}`);
    }
  }
  
  if (accounts.length === 0) {
    throw new Error('Could not create any test accounts. Load test aborted.');
  }
  
  console.log(`Test accounts created successfully for stress testing: ${accounts.join(', ')}`);
  return { accounts };
}

export default function (data) {
  const accounts = data.accounts;
  // Pick an account (low number of accounts ensures VUs hit the same account concurrently)
  const accountId = accounts[Math.floor(Math.random() * accounts.length)];
  
  const payload = JSON.stringify({
    account_id: accountId,
    amount: 10, // 0.10 USD
    currency: 'USD',
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': uuidv4(), // Unique key per request to bypass cache and write to DB
    },
  };
  
  const res = http.post(`${BASE_URL}/v1/payments`, payload, params);
  
  check(res, {
    'status is 202': (r) => r.status === 202,
  });
  
  // No sleep/wait is set: VUs will hammer the API, creating intense lock contention on PostgreSQL rows!
}
