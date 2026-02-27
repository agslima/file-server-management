<?php

namespace App\Services;

use App\Clients\FileEngineClient;
use Illuminate\Http\Client\Factory as HttpFactory;
use Illuminate\Http\Client\Response;

class FileEngineService
{
    private HttpFactory $http;

    private ?string $baseUrl;

    private ?string $adminBaseUrl;

    private ?string $bearerToken;

    /**
     * Initialize the service with an HTTP factory and optional API base URLs and service-level bearer token.
     *
     * @param string|null $baseUrl Optional public API base URL to override the configured default.
     * @param string|null $adminBaseUrl Optional admin API base URL to override the derived admin base.
     * @param string|null $bearerToken Optional service-level bearer token; when provided it takes precedence over forwarded Authorization headers.
     */
    public function __construct(
        HttpFactory $http,
        ?string $baseUrl = null,
        ?string $adminBaseUrl = null,
        ?string $bearerToken = null
    ) {
        $this->http = $http;
        $this->baseUrl = $baseUrl;
        $this->adminBaseUrl = $adminBaseUrl;
        $this->bearerToken = $bearerToken;
    }

    /**
     * Resolve the public API base URL for the File Engine service.
     *
     * Uses the configured instance value if provided, otherwise falls back to the
     * `services.fileengine.base_url` config key or the default `http://file-engine:8080/v1`.
     *
     * @return string The resolved base URL with any trailing slash removed.
     */
    private function base(): string
    {
        return rtrim($this->baseUrl ?? config('services.fileengine.base_url', 'http://file-engine:8080/v1'), '/');
    }

    /**
     * Resolves the admin API base URL for the File Engine service.
     *
     * If an explicit admin base URL was provided or configured, that value is returned trimmed of any trailing slash.
     * Otherwise the value is derived from the public base URL by removing a trailing `/v1` path segment (if present)
     * and combining the scheme, host, optional port, and remaining path.
     *
     * @return string The resolved admin base URL without a trailing slash.
     * @throws \RuntimeException If the public base URL is invalid or cannot be parsed.
     */
    private function adminBase(): string
    {
        $configured = trim((string) ($this->adminBaseUrl ?? config('services.fileengine.admin_base_url', '')));
        if ($configured !== '') {
            return rtrim($configured, '/');
        }

        $base = $this->base();
        $parts = parse_url($base);
        if (!is_array($parts) || !isset($parts['scheme'], $parts['host'])) {
            throw new \RuntimeException(sprintf('invalid file-engine base URL: %s', $base));
        }

        $path = $parts['path'] ?? '';
        $path = preg_replace('#/v1/?$#', '', $path) ?? $path;
        $path = rtrim($path, '/');

        $adminBase = sprintf('%s://%s', $parts['scheme'], $parts['host']);
        if (isset($parts['port'])) {
            $adminBase .= ':' . $parts['port'];
        }
        if ($path !== '') {
            $adminBase .= $path;
        }

        return $adminBase;
    }

    /**
     * Create a FileEngineClient configured with the service's public API base URL and bearer token.
     *
     * @return FileEngineClient The HTTP client configured for the public File Engine API; uses an empty bearer token when none is configured.
     */
    private function apiClient(): FileEngineClient
    {
        return new FileEngineClient($this->http, $this->base(), (string) ($this->bearerToken ?? ''));
    }

    /**
     * Create a FileEngineClient configured for admin API endpoints.
     *
     * @return FileEngineClient A client configured to target the admin API base URL and to use the service-level bearer token when present.
     */
    private function adminClient(): FileEngineClient
    {
        return new FileEngineClient($this->http, $this->adminBase(), (string) ($this->bearerToken ?? ''));
    }

    /**
     * Creates a folder under the specified parent path in the File Engine.
     *
     * @param string $path Parent path where the folder will be created.
     * @param string $folderName Name of the folder to create.
     * @param string $requestedBy Identifier of the actor requesting the creation.
     * @param string[] $traceHeaders Optional trace headers (e.g. X-Request-Id, traceparent). `Authorization` is forwarded only when no service-level bearer token is configured.
     * @return array The response payload as an associative array; includes `_engine_http_status` with the HTTP status code.
     */
    public function createFolder(string $path, string $folderName, string $requestedBy, array $traceHeaders = []): array
    {
        $response = $this->apiClient()->post('/folders', [
            'parentPath' => $path,
            'folderName' => $folderName,
            'requestedBy' => $requestedBy,
        ], $this->filteredTraceHeaders($traceHeaders));

        return $this->withStatus($response);
    }

    /**
     * Initiates an upload session with the File Engine.
     *
     * @param array $payload Request payload for the upload initiation.
     * @param array $traceHeaders Optional trace headers to forward (allowed keys: `X-Request-Id`, `X-Correlation-Id`, `traceparent`, `tracestate`, `baggage`).
     * @param string $idempotencyKey Optional idempotency key sent as `X-Idempotency-Key` to make the request idempotent.
     * @return array The response payload augmented with `_engine_http_status` containing the HTTP status code.
     */
    public function initiateUpload(array $payload, array $traceHeaders = [], string $idempotencyKey = ''): array
    {
        $headers = $this->filteredTraceHeaders($traceHeaders);
        if ($idempotencyKey !== '') {
            $headers['X-Idempotency-Key'] = $idempotencyKey;
        }

        return $this->withStatus($this->apiClient()->post('/uploads:initiate', $payload, $headers));
    }

    /**
     * Uploads a chunk of data for an in-progress upload session.
     *
     * @param string $uploadId The identifier of the upload session.
     * @param int $offset The byte offset at which this chunk should be written.
     * @param string $content The raw chunk data to upload (may contain binary data).
     * @param array $traceHeaders Optional trace headers to forward (filtered by the service).
     * @return array The response payload from the file engine augmented with `_engine_http_status` containing the HTTP status code.
     */
    public function uploadChunk(string $uploadId, int $offset, string $content, array $traceHeaders = []): array
    {
        return $this->withStatus($this->apiClient()->putRaw('/uploads/' . rawurlencode($uploadId) . ':chunk?offset=' . $offset, $content, $this->filteredTraceHeaders($traceHeaders)));
    }

    /**
     * Completes an in-progress upload session for the given upload ID.
     *
     * @param string $uploadId The upload session identifier.
     * @param array $traceHeaders Optional trace headers to include (allowed: `X-Request-Id`, `X-Correlation-Id`, `traceparent`, `tracestate`, `baggage`). Caller `Authorization` is forwarded only when no service-level bearer token is configured; otherwise the configured token takes precedence.
     * @param string $idempotencyKey Optional idempotency key to make the request idempotent.
     * @return array The response payload as an associative array, augmented with an `_engine_http_status` key containing the HTTP status code.
     */
    public function completeUpload(string $uploadId, array $traceHeaders = [], string $idempotencyKey = ''): array
    {
        $headers = $this->filteredTraceHeaders($traceHeaders);
        if ($idempotencyKey !== '') {
            $headers['X-Idempotency-Key'] = $idempotencyKey;
        }

        return $this->withStatus($this->apiClient()->post('/uploads/' . rawurlencode($uploadId) . ':complete', [], $headers));
    }


    /**
     * Moves an object from a source path to a destination path in the remote file engine.
     *
     * @param string $sourcePath The source object's path.
     * @param string $destinationPath The destination path for the object.
     * @param array $traceHeaders Optional trace or authentication headers to forward with the request.
     * @return array The file-engine response payload with an added '_engine_http_status' key containing the HTTP status code.
     */
    public function moveObject(string $sourcePath, string $destinationPath, array $traceHeaders = []): array
    {
        return $this->withStatus($this->apiClient()->post('/objects:move', [
            'source_path' => $sourcePath,
            'destination_path' => $destinationPath,
        ], $this->filteredTraceHeaders($traceHeaders)));
    }

    /**
     * Deletes an object at the given path in the File Engine.
     *
     * @param string $path The filesystem-like path of the object to delete.
     * @param array $traceHeaders Optional trace-related headers (e.g. `X-Request-Id`, `X-Correlation-Id`, `traceparent`, `tracestate`, `baggage`) to include on the request. Caller `Authorization` is forwarded only when no service-level bearer token is configured.
     * @return array The parsed response payload merged into an array and augmented with `_engine_http_status` containing the HTTP status code.
     */
    public function deleteObject(string $path, array $traceHeaders = []): array
    {
        return $this->withStatus($this->apiClient()->post('/objects:delete', [
            'path' => $path,
        ], $this->filteredTraceHeaders($traceHeaders)));
    }

    /**
     * Restores a quarantined object identified by its storage path.
     *
     * @param string $path The storage path of the quarantined object to restore.
     * @param bool $forceReprocess If true, force the object to be reprocessed after restoration.
     * @param array $traceHeaders Optional trace-related headers to include with the request.
     * @return array The response payload as an associative array, augmented with `_engine_http_status`.
     */
    public function restoreQuarantinedObject(string $path, bool $forceReprocess, array $traceHeaders = []): array
    {
        return $this->withStatus($this->adminClient()->post('/admin/v1/quarantine:restore', [
            'path' => $path,
            'force_reprocess' => $forceReprocess,
        ], $this->filteredTraceHeaders($traceHeaders)));
    }

    /**
     * Retrieve a task by its identifier from the File Engine service.
     *
     * @param string $id The task identifier.
     * @param array $traceHeaders Optional trace-related headers to include (e.g. `X-Request-Id`, `X-Correlation-Id`, `traceparent`, `tracestate`, `baggage`). Caller `Authorization` is forwarded only when no service-level bearer token is configured.
     * @return array The parsed JSON response augmented with an `_engine_http_status` entry containing the HTTP status code.
     */
    public function getTask(string $id, array $traceHeaders = []): array
    {
        return $this->withStatus($this->apiClient()->get('/tasks/' . $id, $this->filteredTraceHeaders($traceHeaders)));
    }


    /**
     * Sanitizes and selects trace headers to forward to the File Engine service.
     *
     * Only the following keys are forwarded when present as non-empty strings:
     * `X-Request-Id`, `X-Correlation-Id`, `traceparent`, `tracestate`, `baggage`.
     * `Authorization` is forwarded only when no service-level bearer token is configured.
     * All returned values are trimmed strings; empty or non-string inputs are omitted.
     *
     * @param array<string,mixed> $traceHeaders Raw headers to filter.
     * @return array<string,string> Filtered headers with trimmed string values.
     */
    private function filteredTraceHeaders(array $traceHeaders): array
    {
        $allowed = [
            'X-Request-Id',
            'X-Correlation-Id',
            'traceparent',
            'tracestate',
            'baggage',
        ];

        $headers = [];
        foreach ($allowed as $name) {
            $value = $traceHeaders[$name] ?? null;
            if (!is_string($value)) {
                continue;
            }
            $clean = trim($value);
            if ($clean === '') {
                continue;
            }
            $headers[$name] = $clean;
        }

        if (($this->bearerToken ?? '') === '') {
            $authorization = $traceHeaders['Authorization'] ?? null;
            if (is_string($authorization) && trim($authorization) !== '') {
                $headers['Authorization'] = trim($authorization);
            }
        }

        return $headers;
    }
    /**
     * Extracts the JSON payload from the HTTP response and attaches the response status code.
     *
     * @param Response $response The HTTP response to extract the payload from.
     * @return array The response payload as an associative array with an added `_engine_http_status` key containing the HTTP status code.
     */
    private function withStatus(Response $response): array
    {
        $payload = $response->json();
        if (!is_array($payload)) {
            $payload = [];
        }

        $payload['_engine_http_status'] = $response->status();

        return $payload;
    }
}
