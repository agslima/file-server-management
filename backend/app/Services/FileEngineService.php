<?php

namespace App\Services;

use Illuminate\Http\Client\Factory as HttpFactory;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Http\Client\Response;

class FileEngineService
{
    public function __construct(
        private readonly HttpFactory $http,
        private readonly ?string $baseUrl = null,
        private readonly ?string $bearerToken = null,
    ) {
    }

    private function base(): string
    {
        return rtrim($this->baseUrl ?? config('services.fileengine.base_url', 'http://file-engine:8080/v1'), '/');
    }

    private function client(): PendingRequest
    {
        if (($this->bearerToken ?? '') !== '') {
            return $this->http->withToken($this->bearerToken);
        }

        return $this->http;
    }

    public function createFolder(string $path, string $folderName, string $requestedBy, array $traceHeaders = []): array
    {
        $response = $this->requestWithTraceHeaders($traceHeaders)->post($this->base() . '/folders', [
            'parentPath' => $path,
            'folderName' => $folderName,
            'requestedBy' => $requestedBy,
        ]);

        return $this->withStatus($response);
    }

    public function initiateUpload(array $payload, array $traceHeaders = [], string $idempotencyKey = ''): array
    {
        $request = $this->requestWithTraceHeaders($traceHeaders);
        if ($idempotencyKey !== '') {
            $request = $request->withHeaders(['X-Idempotency-Key' => $idempotencyKey]);
        }

        return $this->withStatus($request->post($this->base() . '/uploads:initiate', $payload));
    }

    public function uploadChunk(string $uploadId, int $offset, string $content, array $traceHeaders = []): array
    {
        $request = $this->requestWithTraceHeaders($traceHeaders)->withBody($content, 'application/octet-stream');

        return $this->withStatus($request->put($this->base() . '/uploads/' . rawurlencode($uploadId) . ':chunk?offset=' . $offset));
    }

    public function completeUpload(string $uploadId, array $traceHeaders = [], string $idempotencyKey = ''): array
    {
        $request = $this->requestWithTraceHeaders($traceHeaders);
        if ($idempotencyKey !== '') {
            $request = $request->withHeaders(['X-Idempotency-Key' => $idempotencyKey]);
        }

        return $this->withStatus($request->post($this->base() . '/uploads/' . rawurlencode($uploadId) . ':complete'));
    }


    public function moveObject(string $sourcePath, string $destinationPath, array $traceHeaders = []): array
    {
        return $this->withStatus($this->requestWithTraceHeaders($traceHeaders)->post($this->base() . '/objects:move', [
            'source_path' => $sourcePath,
            'destination_path' => $destinationPath,
        ]));
    }

    public function deleteObject(string $path, array $traceHeaders = []): array
    {
        return $this->withStatus($this->requestWithTraceHeaders($traceHeaders)->post($this->base() . '/objects:delete', [
            'path' => $path,
        ]));
    }

    public function restoreQuarantinedObject(string $path, bool $forceReprocess, array $traceHeaders = []): array
    {
        $adminBase = preg_replace('#/v1$#', '', $this->base()) ?: $this->base();

        return $this->withStatus($this->requestWithTraceHeaders($traceHeaders)->post($adminBase . '/admin/v1/quarantine:restore', [
            'path' => $path,
            'force_reprocess' => $forceReprocess,
        ]));
    }

    public function getTask(string $id, array $traceHeaders = []): array
    {
        return $this->withStatus($this->requestWithTraceHeaders($traceHeaders)->get($this->base() . '/tasks/' . $id));
    }

    /**
     * @param array<string,mixed> $traceHeaders
     */
    private function requestWithTraceHeaders(array $traceHeaders): PendingRequest
    {
        $allowed = [
            'X-Request-Id',
            'X-Correlation-Id',
            'traceparent',
            'tracestate',
            'baggage',
            'Authorization',
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

        return $this->client()->withHeaders($headers);
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
