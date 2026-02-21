<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
use App\Support\TraceHeaders;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class UploadController extends Controller
{
    public function __construct(private readonly FileEngineService $engine)
    {
    }

    public function initiate(Request $request): JsonResponse
    {
        $path = (string) $request->input('path', '');
        if ($path === '') {
            return new JsonResponse(['message' => 'path is required'], 422);
        }

        $payload = $this->engine->initiateUpload(
            ['path' => $path],
            TraceHeaders::fromRequest($request),
            (string) $request->header('X-Idempotency-Key', '')
        );

        return $this->fromEnginePayload($payload);
    }

    public function uploadChunk(Request $request, string $uploadId): JsonResponse
    {
        if ($uploadId === '') {
            return new JsonResponse(['message' => 'uploadId is required'], 422);
        }

        $offset = (int) $request->input('offset', 0);
        $payload = (string) $request->getContent();

        $response = $this->engine->uploadChunk(
            $uploadId,
            $offset,
            $payload,
            TraceHeaders::fromRequest($request)
        );

        return $this->fromEnginePayload($response);
    }

    public function complete(Request $request, string $uploadId): JsonResponse
    {
        if ($uploadId === '') {
            return new JsonResponse(['message' => 'uploadId is required'], 422);
        }

        $payload = $this->engine->completeUpload(
            $uploadId,
            TraceHeaders::fromRequest($request),
            (string) $request->header('X-Idempotency-Key', '')
        );

        return $this->fromEnginePayload($payload);
    }

    private function fromEnginePayload(array $payload): JsonResponse
    {
        $status = (int) ($payload['_engine_http_status'] ?? 200);
        unset($payload['_engine_http_status']);

        return new JsonResponse($payload, $status);
    }
}
