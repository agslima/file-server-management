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

/**
 * Perform an HTTP request with a JSON Content-Type header and parse the response body when possible.
 *
 * @param {string} url - The request URL.
 * @param {RequestInit} [options] - Fetch options; provided headers are merged with `Content-Type: application/json`.
 * @returns {any|null} The parsed JSON value if the response contains valid JSON, the raw response text if parsing fails, or `null` for empty responses.
 * @throws {Error} If the response has a non-OK HTTP status; the error message includes status, status text, and response body.
 */
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

/**
 * Parse a JSON string and fall back to the original value when parsing fails.
 * @param {*} value - The value to parse as JSON.
 * @returns {*} The parsed value if `value` is valid JSON, otherwise the original `value`.
 */
function safeJson(value) {
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

/**
 * Authenticate using credentials from the username and password inputs and persist the received token.
 *
 * Sends the collected credentials to the backend login endpoint, saves the returned token to `state.token`,
 * updates the "loginStatus" element to reflect authentication state, and emits the raw login response via `out`.
 */
async function login() {
  const payload = { username: $("username").value, password: $("password").value };
  const data = await request(`${backend()}/login`, { method: 'POST', body: JSON.stringify(payload) });
  state.token = data.access_token || data.token || '';
  $("loginStatus").textContent = state.token ? 'Authenticated.' : 'Login response has no token.';
  out('Login result', data);
}

/**
 * Normalize a tenant path by substituting the default tenant segment with the selected tenant.
 * @param {string} raw - The original path that includes the default segment `/tenants/acme`.
 * @returns {string} The path with `/tenants/acme` replaced by `/tenants/{selectedTenant}` where `{selectedTenant}` is the current value of the `tenant` input.
 */
function tenantPath(raw) {
  const tenant = $("tenant").value;
  return raw.replace('/tenants/acme', `/tenants/${tenant}`);
}

/**
 * Create a folder on the backend using the path from the folderPath input.
 *
 * Sends a POST to the backend /folders endpoint with a payload containing the
 * tenant-normalized path, updates state.latestTaskId from `task_id` or `id`
 * in the response when present, and emits the response via `out` with the
 * label "Create folder".
 */
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

/**
 * Fetches the status of the most recently created task and displays it in the output area.
 *
 * @throws {Error} If there is no latest task id available (create a task first).
 */
async function getTaskStatus() {
  if (!state.latestTaskId) throw new Error('No task id yet. Create folder first.');
  const data = await request(`${backend()}/tasks/${state.latestTaskId}`, { headers: authHeaders() });
  out('Task status', data);
}

/**
 * Initiates an upload for the path specified in the uploadPath input, uploads the content from the uploadBody input, and completes the upload.
 *
 * Performs the backend upload initiation, sends a single chunk containing the upload body, and marks the upload complete; outputs the initiation and completion responses via the application's output helper.
 */
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

/**
 * Send a JSON POST to a backend endpoint and print the response with a label.
 * @param {string} endpoint - Backend-relative path (appended to the configured backend base URL).
 * @param {any} payload - Value to serialize as the JSON request body.
 * @param {string} label - Short label used when outputting the response.
 */
async function mutate(endpoint, payload, label) {
  const data = await request(`${backend()}${endpoint}`, {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: authHeaders(),
  });
  out(label, data);
}

/**
 * Fetches dead-letter queue entries from the engine admin API, stores them in state.dlq, and prints the response under the "DLQ list" label.
 *
 * The stored value will be an array (empty if the response has no `entries` field).
 */
async function listDlq() {
  const data = await request(`${engine()}/admin/v1/scan-dlq`, { headers: authHeaders() });
  state.dlq = data.entries || [];
  out('DLQ list', data);
}

/**
 * Retry the first entry currently loaded in the dead-letter queue by invoking the engine admin retry endpoint.
 *
 * Posts the `id` of the first entry in `state.dlq` to the engine's /admin/v1/scan-dlq endpoint and emits the response
 * to the UI output area with the label "DLQ retry".
 *
 * @throws {Error} If no DLQ entries are loaded in `state.dlq`.
 */
async function retryDlq() {
  if (!state.dlq.length) throw new Error('No DLQ entries loaded. Click list first.');
  const data = await request(`${engine()}/admin/v1/scan-dlq`, {
    method: 'POST',
    body: JSON.stringify({ id: state.dlq[0].id }),
    headers: authHeaders(),
  });
  out('DLQ retry', data);
}

/**
 * Initiates a quarantine cleanup on the engine with a TTL of 3600 seconds.
 *
 * Sends a POST to the engine admin quarantine cleanup endpoint and emits the response to the UI output.
 * @returns {any} The parsed response body from the cleanup request, or `null` if the response has no body.
 */
async function cleanupQuarantine() {
  const data = await request(`${engine()}/admin/v1/quarantine:cleanup?ttl_seconds=3600`, {
    method: 'POST',
    body: JSON.stringify({}),
    headers: authHeaders(),
  });
  out('Quarantine cleanup', data);
}

/**
 * Fetches and outputs the effective governance policy for the selected tenant.
 *
 * Reads the tenant identifier from the `policyTenant` input, requests the engine
 * admin effective-policy endpoint for that tenant using auth headers, and writes
 * the response to the output pane labeled "Effective policy".
 */
async function effectivePolicy() {
  const tenant = $("policyTenant").value;
  const data = await request(`${engine()}/admin/v1/governance:effective?tenant_id=${encodeURIComponent(tenant)}`, {
    headers: authHeaders(),
  });
  out('Effective policy', data);
}

/**
 * Requests governance drift status from the engine and outputs the response.
 *
 * Sends a drift-check request to the engine admin governance endpoint and writes the returned data to the output area.
 */
async function driftStatus() {
  const data = await request(`${engine()}/admin/v1/governance:drift-check`, {
    method: 'POST',
    body: JSON.stringify({}),
    headers: authHeaders(),
  });
  out('Drift status', data);
}

/**
 * Fetches the evidence pack pointer for the selected tenant and writes the response to the output.
 *
 * Reads the tenant identifier from the `policyTenant` input, requests the tenant's evidence pack
 * pointer from the engine admin API, and emits the returned data via the application's output handler.
 */
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
