import http from 'k6/http';
import { check, sleep } from 'k6';

const base = __ENV.BASE_URL || 'http://localhost:8080';
if (!__ENV.TOKEN) {
  throw new Error('TOKEN is required. Set TOKEN in the environment before running k6.');
}

const token = __ENV.TOKEN;
const headers = { Authorization: `Bearer ${token}` };

export const options = {
  vus: 4,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
    'http_req_duration{operation:list}': ['p(95)<250'],
    'http_req_duration{operation:download}': ['p(95)<300'],
    'http_req_duration{operation:upload_complete}': ['p(95)<1200'],
    'http_req_duration{operation:mutation}': ['p(95)<800'],
    'http_req_duration{operation:task_status}': ['p(95)<250'],
  },
};

function asJSON(response) {
  try {
    return response.json();
  } catch (_err) {
    return {};
  }
}

export default function () {
  const tenantPath = `/tenants/acme/load-${__VU}-${__ITER}`;

  const listRes = http.get(`${base}/v1/objects?prefix=${encodeURIComponent('/tenants/acme')}`, {
    headers,
    tags: { operation: 'list' },
    responseCallback: http.expectedStatuses(200, 404),
  });
  check(listRes, { 'list status is 200 or 404': (r) => r.status === 200 || r.status === 404 });

  const createRes = http.post(
    `${base}/v1/folders`,
    JSON.stringify({ parent_path: tenantPath, folder_name: 'budget-smoke', requested_by: 'k6-smoke' }),
    {
      headers: { ...headers, 'Content-Type': 'application/json' },
      tags: { operation: 'mutation' },
    }
  );
  check(createRes, { 'create folder accepted': (r) => r.status === 200 });

  const createBody = asJSON(createRes);
  if (createBody.task_id) {
    const taskRes = http.get(`${base}/v1/tasks/${createBody.task_id}`, {
      headers,
      tags: { operation: 'task_status' },
    });
    check(taskRes, { 'task status reachable': (r) => r.status === 200 });
  }

  const initRes = http.post(
    `${base}/v1/uploads:initiate`,
    JSON.stringify({ path: `${tenantPath}/smoke.txt` }),
    {
      headers: { ...headers, 'Content-Type': 'application/json', 'X-Idempotency-Key': `k6-init-${__VU}-${__ITER}` },
      tags: { operation: 'mutation' },
    }
  );
  check(initRes, { 'upload initiate accepted': (r) => r.status === 200 });

  const initBody = asJSON(initRes);
  if (initBody.upload_id) {
    const chunkRes = http.put(`${base}/v1/uploads/${initBody.upload_id}:chunk?offset=0`, 'budget-smoke', {
      headers: { ...headers, 'Content-Type': 'application/octet-stream' },
      tags: { operation: 'mutation' },
    });
    check(chunkRes, { 'upload chunk accepted': (r) => r.status === 202 });

    const completeRes = http.post(`${base}/v1/uploads/${initBody.upload_id}:complete`, null, {
      headers: { ...headers, 'X-Idempotency-Key': `k6-complete-${__VU}-${__ITER}` },
      tags: { operation: 'upload_complete' },
    });
    check(completeRes, { 'upload complete accepted': (r) => r.status === 200 });
  }

  const dlRes = http.get(`${base}/v1/objects:download?path=${encodeURIComponent('/tenants/acme/docs/a.txt')}`, {
    headers,
    tags: { operation: 'download' },
    responseCallback: http.expectedStatuses(200, 404),
  });
  check(dlRes, { 'download status is 200 or 404': (r) => r.status === 200 || r.status === 404 });

  sleep(0.25);
}
