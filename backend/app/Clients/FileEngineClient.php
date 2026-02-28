<?php

namespace App\Clients;

use Illuminate\Http\Client\Factory as HttpFactory;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Http\Client\Response;
use Illuminate\Support\Arr;
use Throwable;

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

    public function postOrThrow(string $path, array $payload = [], array $headers = []): Response
    {
        return $this->invokeOrThrow('POST', fn () => $this->post($path, $payload, $headers));
    }

    public function getOrThrow(string $path, array $headers = []): Response
    {
        return $this->invokeOrThrow('GET', fn () => $this->get($path, $headers));
    }

    public function putRawOrThrow(string $path, string $content, array $headers = []): Response
    {
        return $this->invokeOrThrow('PUT', fn () => $this->putRaw($path, $content, $headers));
    }

    /**
     * @param callable(): Response $call
     */
    private function invokeOrThrow(string $method, callable $call): Response
    {
        try {
            return $this->throwIfError($call());
        } catch (FileEngineException $exception) {
            throw $exception;
        } catch (Throwable $exception) {
            throw new FileEngineException(
                0,
                'TRANSPORT_ERROR',
                'transport_error',
                true,
                "file-engine {$method} request failed: {$exception->getMessage()}",
                $exception,
            );
        }
    }

    private function throwIfError(Response $response): Response
    {
        if ($response->status() < 300) {
            return $response;
        }

        $payload = $response->json();
        $codeValue = (string) Arr::get($payload, 'error.code', 'HTTP_ERROR');
        $reason = (string) Arr::get($payload, 'error.reason', 'http_error');
        $retryable = (bool) Arr::get($payload, 'error.retryable', false);
        $message = (string) Arr::get($payload, 'error.message', $response->body());

        throw new FileEngineException($response->status(), $codeValue, $reason, $retryable, $message);
    }
}
