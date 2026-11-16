import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 2,
  iterations: 10,
  thresholds: { http_req_failed: ['rate<0.05'] },
};

const BASE = __ENV.API_URL || 'http://localhost:8080/api/v1';

function login(email, password) {
  const res = http.post(`${BASE}/auth/login`, JSON.stringify({ email, password }), { headers: { 'Content-Type': 'application/json' } });
  check(res, { 'login ok': (r) => r.status === 200 });
  return res.json('data.token');
}

export default function () {
  const admin = login(__ENV.ADMIN_EMAIL || 'warehouse@wms.local', __ENV.ADMIN_PASSWORD || 'Warehouse123!');
  const dept = login(__ENV.DEPT_EMAIL || 'staff@wms.local', __ENV.DEPT_PASSWORD || 'Staff123!');
  const auth = (t) => ({ headers: { Authorization: `Bearer ${t}`, 'Content-Type': 'application/json' } });
  const products = http.get(`${BASE}/products?page_size=1`, auth(dept));
  check(products, { 'products ok': (r) => r.status === 200 });
  const pid = products.json('data.0.id');
  if (!pid) { sleep(1); return; }
  const key = `k6-${__VU}-${__ITER}-${Date.now()}`;
  const create = http.post(`${BASE}/stock-requests`, JSON.stringify({ purpose: 'k6 load test request', items: [{ product_id: pid, requested_qty: 1 }] }), { headers: { ...auth(dept).headers, 'Idempotency-Key': key } });
  check(create, { 'request created or handled': (r) => r.status === 201 || r.status === 409 || r.status === 422 });
  sleep(1);
}
