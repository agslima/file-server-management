import http from 'k6/http';
import { check, sleep } from 'k6';

const base = __ENV.BASE_URL || 'http://localhost:8080';
const token = __ENV.TOKEN || __ENV.TOKEN_JWT;
if (!token) {
  throw new Error('TOKEN/TOKEN_JWT is required. Set TOKEN_JWT (or TOKEN) in the environment before running k6.');
}
export const options = {
  stages: [
    { duration: '2m', target: 10 },
    { duration: '10m', target: 10 },
    { duration: '2m', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.02'],
    checks: ['rate>0.98'],
    'http_req_duration{operation:download}': ['p(95)<350'],
    'http_req_duration{operation:upload_complete}': ['p(95)<1500'],
    'http_req_duration{operation:mutation}': ['p(95)<900'],
  },
};

export default function () {
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

  const createRes = http.post(
    `${base}/v1/folders`,
    JSON.stringify({ parent_path: '/tenants/acme/soak', folder_name: `iter-${__VU}-${__ITER}`, requested_by: 'k6-soak' }),
    { headers, tags: { operation: 'mutation' } }
  );
  check(createRes, { 'create folder accepted': (r) => r.status === 200 });

  const initRes = http.post(
    `${base}/v1/uploads:initiate`,
    JSON.stringify({ path: `/tenants/acme/soak/iter-${__VU}-${__ITER}.txt` }),
    { headers: { ...headers, 'X-Idempotency-Key': `soak-init-${__VU}-${__ITER}` }, tags: { operation: 'mutation' } }
  );
  check(initRes, { 'initiate accepted': (r) => r.status === 200 });

  if (initRes.status === 200) {
    const uploadId = initRes.json('upload_id');
    if (!uploadId || typeof uploadId !== 'string' || uploadId.trim() === '') {
      check(initRes, { 'upload id present': () => false });
      sleep(0.5);
      return;
    }

    const chunkRes = http.put(`${base}/v1/uploads/${uploadId}:chunk?offset=0`, 'soak', {
      headers: { Authorization: `Bearer ${token}` },
      tags: { operation: 'mutation' },
    });
    check(chunkRes, { 'chunk accepted': (r) => r.status === 202 });

    const completeRes = http.post(`${base}/v1/uploads/${uploadId}:complete`, null, {
      headers: { Authorization: `Bearer ${token}`, 'X-Idempotency-Key': `soak-complete-${__VU}-${__ITER}` },
      tags: { operation: 'upload_complete' },
    });
    check(completeRes, { 'complete accepted': (r) => r.status === 200 });
  }

  const dlRes = http.get(`${base}/v1/objects:download?path=${encodeURIComponent('/tenants/acme/docs/a.txt')}`, {
    headers: { Authorization: `Bearer ${token}` },
    tags: { operation: 'download' },
    responseCallback: http.expectedStatuses(200, 404),
  });
  check(dlRes, { 'download status ok': (r) => r.status === 200 || r.status === 404 });

  sleep(0.5);
}
