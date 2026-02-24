<?php

namespace App\Clients;

use Illuminate\Http\Client\Factory as HttpFactory;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Http\Client\Response;

class FileEngineClient
{
    public function __construct(
        private readonly HttpFactory $http,
        private readonly string $baseUrl,
        private readonly string $bearerToken = '',
    ) {
    }

    private function request(): PendingRequest
    {
        if ($this->bearerToken !== '') {
            return $this->http->withToken($this->bearerToken);
        }

        return $this->http->withHeaders([]);
    }

    public function post(string $path, array $payload = [], array $headers = []): Response
    {
        return $this->request()->withHeaders($headers)->post(rtrim($this->baseUrl, '/') . '/' . ltrim($path, '/'), $payload);
    }

    public function get(string $path, array $headers = []): Response
    {
        return $this->request()->withHeaders($headers)->get(rtrim($this->baseUrl, '/') . '/' . ltrim($path, '/'));
    }

    public function putRaw(string $path, string $content, array $headers = []): Response
    {
        return $this->request()->withHeaders($headers)->withBody($content, 'application/octet-stream')->put(rtrim($this->baseUrl, '/') . '/' . ltrim($path, '/'));
    }
}
