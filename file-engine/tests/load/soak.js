import http from 'k6/http';
import { check } from 'k6';

export const options = { stages: [{duration:'2m',target:5},{duration:'10m',target:5},{duration:'2m',target:0}], thresholds: { http_req_failed: ['rate<0.05'] } };
const base = __ENV.BASE_URL || 'http://localhost:8081';

export default function () {
  const headers = { Authorization: `Bearer ${__ENV.TOKEN || ''}` };
  const res = http.get(`${base}/metrics`, { headers });
  check(res, { 'metrics reachable': (r) => r.status === 200 });
}
