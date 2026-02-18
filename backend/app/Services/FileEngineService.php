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

    public function createFolder(string $path, string $folderName, string $requestedBy): array
    {
        $response = $this->client()->post($this->base() . '/folders', [
            'parentPath' => $path,
            'folderName' => $folderName,
            'requestedBy' => $requestedBy,
        ]);

        return $this->withStatus($response);
    }

    public function initiateUpload(array $payload, string $requestedBy): array
    {
        $payload['createdBy'] = $requestedBy;
        return $this->withStatus($this->client()->post($this->base() . '/uploads/initiate', $payload));
    }

    public function completeUpload(string $uploadId): array
    {
        return $this->withStatus($this->client()->post($this->base() . '/uploads/complete', [
            'uploadId' => $uploadId,
        ]));
    }

    public function getTask(string $id): array
    {
        return $this->withStatus($this->client()->get($this->base() . '/tasks/' . $id));
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
