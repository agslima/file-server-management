<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
use Illuminate\Http\JsonResponse;

class TaskController extends Controller
{
    public function __construct(private readonly FileEngineService $engine)
    {
    }

    public function show(string $id): JsonResponse
    {
        $payload = $this->engine->getTask($id);
        $status = (int) ($payload['_engine_http_status'] ?? 200);
        unset($payload['_engine_http_status']);

        return new JsonResponse($payload, $status);
    }
}
