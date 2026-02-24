import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = { vus: 2, duration: '20s', thresholds: { http_req_duration: ['p(95)<1200'] } };
const base = __ENV.BASE_URL || 'http://localhost:8081';

export default function () {
  const headers = { Authorization: `Bearer ${__ENV.TOKEN || ''}` };
  check(http.get(`${base}/v1/objects:download?path=/tenants/acme/docs/a.txt`, { headers }), { 'download status ok-ish': (r) => [200,404].includes(r.status) });
  const init = http.post(`${base}/v1/uploads:initiate`, JSON.stringify({ path: '/tenants/acme/docs/load.txt' }), { headers: { ...headers, 'Content-Type': 'application/json' } });
  check(init, { 'init accepted': (r) => [200,429].includes(r.status) });
  sleep(1);
}
