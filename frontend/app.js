const state = {
  token: '',
  latestTaskId: '',
  dlq: [],
};

const $ = (id) => document.getElementById(id);
const out = (label, data) => {
  const payload = typeof data === 'string' ? data : JSON.stringify(data, null, 2);
  $("output").textContent = `[${new Date().toISOString()}] ${label}\n${payload}`;
};

const backend = () => $("backendBase").value.replace(/\/$/, '');
const engine = () => $("engineBase").value.replace(/\/$/, '');
const authHeaders = () => (state.token ? { Authorization: `Bearer ${state.token}` } : {});

async function request(url, options = {}) {
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
  });
  const text = await res.text();
  const body = text ? safeJson(text) : null;
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}: ${text}`);
  }
  return body;
}

function safeJson(value) {
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

async function login() {
  const payload = { email: $("username").value, password: $("password").value };
  const data = await request(`${backend()}/login`, { method: 'POST', body: JSON.stringify(payload) });
  state.token = data.access_token || data.token || '';
  $("loginStatus").textContent = state.token ? 'Authenticated.' : 'Login response has no token.';
  out('Login result', data);
}

function tenantPath(raw) {
  const tenant = $("tenant").value;
  return raw.replace('/tenants/acme', `/tenants/${tenant}`);
}

async function createFolder() {
  const payload = { path: tenantPath($("folderPath").value) };
  const data = await request(`${backend()}/folders`, {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: authHeaders(),
  });
  state.latestTaskId = data.task_id || data.id || state.latestTaskId;
  out('Create folder', data);
}

async function getTaskStatus() {
  if (!state.latestTaskId) throw new Error('No task id yet. Create folder first.');
  const data = await request(`${backend()}/tasks/${state.latestTaskId}`, { headers: authHeaders() });
  out('Task status', data);
}

async function runUploadFlow() {
  const path = tenantPath($("uploadPath").value);
  const init = await request(`${backend()}/uploads/initiate`, {
    method: 'POST',
    body: JSON.stringify({ path }),
    headers: authHeaders(),
  });
  const uploadId = init.upload_id;
  await request(`${backend()}/uploads/${uploadId}/chunk`, {
    method: 'PUT',
    body: $("uploadBody").value,
    headers: authHeaders(),
  });
  const complete = await request(`${backend()}/uploads/${uploadId}/complete`, {
    method: 'POST',
    body: JSON.stringify({}),
    headers: authHeaders(),
  });
  out('Upload flow complete', { init, complete });
}

async function mutate(endpoint, payload, label) {
  const data = await request(`${backend()}${endpoint}`, {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: authHeaders(),
  });
  out(label, data);
}

async function listDlq() {
  const data = await request(`${engine()}/admin/v1/scan-dlq`, { headers: authHeaders() });
  state.dlq = data.entries || [];
  out('DLQ list', data);
}

async function retryDlq() {
  if (!state.dlq.length) throw new Error('No DLQ entries loaded. Click list first.');
  const data = await request(`${engine()}/admin/v1/scan-dlq`, {
    method: 'POST',
    body: JSON.stringify({ id: state.dlq[0].id }),
    headers: authHeaders(),
  });
  out('DLQ retry', data);
}

async function cleanupQuarantine() {
  const data = await request(`${engine()}/admin/v1/quarantine:cleanup?ttl_seconds=3600`, {
    method: 'POST',
    body: JSON.stringify({}),
    headers: authHeaders(),
  });
  out('Quarantine cleanup', data);
}

async function effectivePolicy() {
  const tenant = $("policyTenant").value;
  const data = await request(`${engine()}/admin/v1/governance:effective?tenant_id=${encodeURIComponent(tenant)}`, {
    headers: authHeaders(),
  });
  out('Effective policy', data);
}

async function driftStatus() {
  const data = await request(`${engine()}/admin/v1/governance:drift-check`, {
    method: 'POST',
    body: JSON.stringify({}),
    headers: authHeaders(),
  });
  out('Drift status', data);
}

async function evidencePack() {
  const tenant = $("policyTenant").value;
  const data = await request(`${engine()}/admin/tenants/${encodeURIComponent(tenant)}/evidence`, {
    headers: authHeaders(),
  });
  out('Evidence pack pointer', data);
}

const wire = (id, fn) => {
  $(id).addEventListener('click', async () => {
    try {
      await fn();
    } catch (error) {
      out(`Error in ${id}`, String(error));
    }
  });
};

wire('loginBtn', login);
wire('createFolderBtn', createFolder);
wire('taskStatusBtn', getTaskStatus);
wire('uploadBtn', runUploadFlow);
wire('moveBtn', () =>
  mutate('/objects/move', {
    source_path: tenantPath($("moveSource").value),
    destination_path: tenantPath($("moveDestination").value),
  }, 'Move object')
);
wire('deleteBtn', () => mutate('/objects/delete', { path: tenantPath($("deletePath").value) }, 'Delete object'));
wire('restoreBtn', () =>
  mutate('/objects/restore', { path: tenantPath($("restorePath").value), force_reprocess: false }, 'Restore object')
);
wire('dlqBtn', listDlq);
wire('retryDlqBtn', retryDlq);
wire('cleanupBtn', cleanupQuarantine);
wire('effectivePolicyBtn', effectivePolicy);
wire('driftBtn', driftStatus);
wire('evidenceBtn', evidencePack);
