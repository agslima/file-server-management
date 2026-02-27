<?php

namespace App\Clients;

use Illuminate\Http\Client\Factory as HttpFactory;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Http\Client\Response;
use Illuminate\Support\Arr;

class FileEngineClient
{
    /**
     * Create a FileEngineClient configured with a base URL and optional bearer token.
     *
     * @param HttpFactory $http HTTP factory used to create HTTP requests.
     * @param string $baseUrl Base URL used as the prefix for all requests.
     * @param string $bearerToken Optional bearer token that will be sent as a `Bearer` Authorization header when provided.
     */
    public function __construct(
        private readonly HttpFactory $http,
        private readonly string $baseUrl,
        private readonly string $bearerToken = '',
    ) {
    }

    /**
     * Create a PendingRequest configured for this client.
     *
     * @return PendingRequest A PendingRequest configured with the bearer token header if one is set; otherwise a PendingRequest with no additional headers.
     */
    private function request(): PendingRequest
    {
        if ($this->bearerToken !== '') {
            return $this->http->withToken($this->bearerToken);
        }

        return $this->http->withHeaders([]);
    }

    /**
     * Send a POST request to the client's base URL combined with the given path.
     *
     * @param string $path Relative path to append to the base URL (leading/trailing slashes are handled).
     * @param array $payload Associative array of request payload to include in the POST body.
     * @param array $headers Additional HTTP headers to include with the request.
     * @return Response The HTTP response returned by the client.
     */
    public function post(string $path, array $payload = [], array $headers = []): Response
    {
        return $this->request()->withHeaders($headers)->post(rtrim($this->baseUrl, '/') . '/' . ltrim($path, '/'), $payload);
    }

    /**
     * Performs an HTTP GET request to the client's base URL combined with the given path.
     *
     * @param string $path The request path relative to the client's base URL.
     * @param array $headers Additional headers to include with the request.
     * @return Response The HTTP response.
     */
    public function get(string $path, array $headers = []): Response
    {
        return $this->request()->withHeaders($headers)->get(rtrim($this->baseUrl, '/') . '/' . ltrim($path, '/'));
    }

    /**
     * Uploads raw binary content to the given path using an HTTP PUT and sends it as `application/octet-stream`.
     *
     * @param string $path The target path relative to the client's base URL.
     * @param string $content The raw body content to upload.
     * @param array $headers Additional HTTP headers to include with the request.
     * @return Response The HTTP response from the PUT request.
     */
    public function putRaw(string $path, string $content, array $headers = []): Response
    {
        return $this->request()->withHeaders($headers)->withBody($content, 'application/octet-stream')->put(rtrim($this->baseUrl, '/') . '/' . ltrim($path, '/'));
    }

    /**
     * Send a POST request to the specified path and throw a FileEngineException if the response indicates an error.
     *
     * @param string $path Relative path to append to the client's base URL.
     * @param array $payload Associative array to send as the request body.
     * @param array $headers Additional headers to include with the request.
     * @return Response The successful HTTP response.
     * @throws \App\Clients\FileEngineException If the response status is 300 or greater.
     */
    public function postOrThrow(string $path, array $payload = [], array $headers = []): Response
    {
        return $this->throwIfError($this->post($path, $payload, $headers));
    }

    /**
     * Perform a GET to the given path on the client's base URL and throw on HTTP error.
     *
     * @param string $path Relative path to append to the base URL.
     * @param array $headers Additional headers to include with the request.
     * @return Response The successful HTTP response (status < 300).
     * @throws \App\Clients\FileEngineException If the response status is 300 or greater; the exception contains status, code, reason, retryable, and message extracted from the response.
     */
    public function getOrThrow(string $path, array $headers = []): Response
    {
        return $this->throwIfError($this->get($path, $headers));
    }

    /**
     * Performs a PUT of raw content to the specified path and throws on HTTP errors.
     *
     * @param string $path Path relative to the configured base URL.
     * @param string $content Raw request body to send.
     * @param array<string,string> $headers Additional headers to include in the request.
     * @return Response The successful HTTP response.
     * @throws FileEngineException If the HTTP response status is 300 or greater.
     */
    public function putRawOrThrow(string $path, string $content, array $headers = []): Response
    {
        return $this->throwIfError($this->putRaw($path, $content, $headers));
    }

    /**
     * Validate an HTTP response and throw a FileEngineException for error responses.
     *
     * Extracts error details from the response JSON (falling back to defaults or the raw body)
     * and throws FileEngineException when the HTTP status is 300 or greater.
     *
     * @param Response $response The HTTP response to evaluate.
     * @return Response The original response when the status code is less than 300.
     * @throws FileEngineException Thrown when the response status is 300 or greater; the exception is constructed with the response status, an error code value, a reason, a retryable flag, and a message.
     */
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
