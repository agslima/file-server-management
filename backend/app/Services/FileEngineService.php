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
        $response = $this->client()->withHeaders($traceHeaders)->post($this->base() . '/folders', [
            'parentPath' => $path,
            'folderName' => $folderName,
            'requestedBy' => $requestedBy,
        ]);

        return $this->withStatus($response);
    }

    public function initiateUpload(array $payload, array $traceHeaders = [], string $idempotencyKey = ''): array
    {
        $request = $this->client()->withHeaders($traceHeaders);
        if ($idempotencyKey !== '') {
            $request = $request->withHeaders(['X-Idempotency-Key' => $idempotencyKey]);
        }

        return $this->withStatus($request->post($this->base() . '/uploads:initiate', $payload));
    }

    public function uploadChunk(string $uploadId, int $offset, string $content, array $traceHeaders = []): array
    {
        $request = $this->client()->withHeaders($traceHeaders)->withBody($content, 'application/octet-stream');

        return $this->withStatus($request->put($this->base() . '/uploads/' . rawurlencode($uploadId) . ':chunk?offset=' . $offset));
    }

    public function completeUpload(string $uploadId, array $traceHeaders = [], string $idempotencyKey = ''): array
    {
        $request = $this->client()->withHeaders($traceHeaders);
        if ($idempotencyKey !== '') {
            $request = $request->withHeaders(['X-Idempotency-Key' => $idempotencyKey]);
        }

        return $this->withStatus($request->post($this->base() . '/uploads/' . rawurlencode($uploadId) . ':complete'));
    }

    public function getTask(string $id, array $traceHeaders = []): array
    {
        return $this->withStatus($this->client()->withHeaders($traceHeaders)->get($this->base() . '/tasks/' . $id));
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
