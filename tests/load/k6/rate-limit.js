import http from 'k6/http';
import { check } from 'k6';

export const options = { vus: 10, iterations: 50 };
const BASE = __ENV.API_URL || 'http://localhost:8080/api/v1';

export default function () {
  const res = http.post(`${BASE}/auth/login`, JSON.stringify({ email: 'nobody@wms.local', password: 'x' }), {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { '401 or 429': (r) => r.status === 401 || r.status === 429 });
}
