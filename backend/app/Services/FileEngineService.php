<?php

namespace App\Services;

use Illuminate\Http\Client\Factory as HttpFactory;

class FileEngineService
{
    public function __construct(
        private readonly HttpFactory $http,
        private readonly ?string $baseUrl = null,
    ) {
    }

    private function base(): string
    {
        return rtrim($this->baseUrl ?? config('services.fileengine.base_url', 'http://file-engine:8080/v1'), '/');
    }

    public function createFolder(string $path, string $folderName, string $requestedBy): array
    {
        return $this->http->post($this->base() . '/folders', [
            'path' => $path,
            'folderName' => $folderName,
            'createdBy' => $requestedBy,
        ])->json();
    }

    public function initiateUpload(array $payload, string $requestedBy): array
    {
        $payload['createdBy'] = $requestedBy;
        return $this->http->post($this->base() . '/uploads/initiate', $payload)->json();
    }

    public function completeUpload(string $uploadId): array
    {
        return $this->http->post($this->base() . '/uploads/complete', [
            'uploadId' => $uploadId,
        ])->json();
    }

    public function getTask(string $id): array
    {
        return $this->http->get($this->base() . '/tasks/' . $id)->json();
    }
}
