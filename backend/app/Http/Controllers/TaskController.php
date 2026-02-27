<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
use App\Support\TraceHeaders;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class TaskController extends Controller
{
    private FileEngineService $engine;

    /**
     * Create a new TaskController with the given FileEngineService.
     *
     * @param FileEngineService $engine The engine used to retrieve task data.
     */
    public function __construct(FileEngineService $engine)
    {
        $this->engine = $engine;
    }

    /**
     * Return the task payload as a JSON response.
     *
     * Retrieves the task data for the given identifier and returns it as JSON.
     * If the payload contains an `_engine_http_status` key its integer value is used
     * as the response HTTP status; that key is removed from the returned payload.
     *
     * @param Request $request Incoming HTTP request (used to propagate trace headers).
     * @param string $id The task identifier to retrieve.
     * @return JsonResponse JSON response containing the task payload; status reflects `_engine_http_status` when present, otherwise 200.
     */
    public function show(Request $request, string $id): JsonResponse
    {
        $payload = $this->engine->getTask($id, TraceHeaders::fromRequest($request, true));
        $status = (int) ($payload['_engine_http_status'] ?? 200);
        unset($payload['_engine_http_status']);

        return new JsonResponse($payload, $status);
    }
}
