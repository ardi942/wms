import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 1,
  iterations: 5,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<1000'],
  },
};

const BASE = __ENV.API_URL || 'http://localhost:8080/api/v1';

export default function () {
  let res = http.get(`${BASE.replace(/\/api\/v1$/, '')}/healthz`);
  check(res, { 'health 200': (r) => r.status === 200 });
  res = http.post(`${BASE}/auth/login`, JSON.stringify({
    email: __ENV.E2E_EMAIL || 'warehouse@wms.local',
    password: __ENV.E2E_PASSWORD || 'Warehouse123!',
  }), { headers: { 'Content-Type': 'application/json' } });
  check(res, { 'login 200': (r) => r.status === 200 });
  const token = res.json('data.token');
  const auth = { headers: { Authorization: `Bearer ${token}` } };
  check(http.get(`${BASE}/products`, auth), { 'products 200': (r) => r.status === 200 });
  check(http.get(`${BASE}/inventory`, auth), { 'inventory 200': (r) => r.status === 200 });
  sleep(1);
}
