<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
use App\Support\TraceHeaders;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class ObjectMutationController extends Controller
{
    public function __construct(private readonly FileEngineService $engine)
    {
    }

    public function move(Request $request): JsonResponse
    {
        $sourcePath = (string) $request->input('sourcePath', '');
        $destinationPath = (string) $request->input('destinationPath', '');
        if ($sourcePath === '' || $destinationPath === '') {
            return new JsonResponse(['message' => 'sourcePath and destinationPath are required'], 422);
        }

        return $this->fromEnginePayload($this->engine->moveObject($sourcePath, $destinationPath, TraceHeaders::fromRequest($request)));
    }

    public function delete(Request $request): JsonResponse
    {
        $path = (string) $request->input('path', '');
        if ($path === '') {
            return new JsonResponse(['message' => 'path is required'], 422);
        }

        return $this->fromEnginePayload($this->engine->deleteObject($path, TraceHeaders::fromRequest($request)));
    }

    public function restore(Request $request): JsonResponse
    {
        $path = (string) $request->input('path', '');
        if ($path === '') {
            return new JsonResponse(['message' => 'path is required'], 422);
        }

        return $this->fromEnginePayload($this->engine->restoreQuarantinedObject(
            $path,
            (bool) $request->boolean('forceReprocess', false),
            TraceHeaders::fromRequest($request)
        ));
    }

    private function fromEnginePayload(array $payload): JsonResponse
    {
        $status = (int) ($payload['_engine_http_status'] ?? 200);
        unset($payload['_engine_http_status']);

        return new JsonResponse($payload, $status);
    }
}
