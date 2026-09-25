<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
use App\Support\TraceHeaders;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class TaskController extends Controller
{
    private FileEngineService $engine;

    public function __construct(FileEngineService $engine)
    {
        $this->engine = $engine;
    }

    public function show(Request $request, string $id): JsonResponse
    {
        $payload = $this->engine->getTask($id, TraceHeaders::fromRequest($request, true));
        $status = (int) ($payload['_engine_http_status'] ?? 200);
        unset($payload['_engine_http_status']);

        return new JsonResponse($payload, $status);
    }
}
