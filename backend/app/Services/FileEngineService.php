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

    private function base(): string
    {
        return rtrim($this->baseUrl ?? config('services.fileengine.base_url', 'http://file-engine:8080/v1'), '/');
    }

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

    private function apiClient(): FileEngineClient
    {
        return new FileEngineClient($this->http, $this->base(), (string) ($this->bearerToken ?? ''));
    }

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

    public function initiateUpload(array $payload, array $traceHeaders = [], string $idempotencyKey = ''): array
    {
        $headers = $this->filteredTraceHeaders($traceHeaders);
        if ($idempotencyKey !== '') {
            $headers['X-Idempotency-Key'] = $idempotencyKey;
        }

        return $this->withStatus($this->apiClient()->post('/uploads:initiate', $payload, $headers));
    }

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
     * Filters forwarded trace headers.
     *
     * Forwarded keys: `X-Request-Id`, `X-Correlation-Id`, `traceparent`, `tracestate`, `baggage`.
     * `Authorization` is forwarded only when no service-level bearer token is configured,
     * so configured service credentials always take precedence.
     *
     * @param array<string,mixed> $traceHeaders
     * @return array<string,string>
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
