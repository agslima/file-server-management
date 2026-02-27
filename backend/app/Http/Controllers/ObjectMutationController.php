<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
use App\Support\TraceHeaders;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class ObjectMutationController extends Controller
{
    /**
     * Initialize the controller with the file engine service.
     *
     * The provided FileEngineService is stored on the controller for use by object mutation endpoints.
     */
    public function __construct(private readonly FileEngineService $engine)
    {
    }

    /**
     * Moves an object from one storage path to another using the configured file engine.
     *
     * @param Request $request HTTP request; must include 'sourcePath' and 'destinationPath' inputs.
     * @return JsonResponse The engine response payload as JSON. If required inputs are missing, returns a JSON error with HTTP status 422.
     */
    public function move(Request $request): JsonResponse
    {
        $sourcePath = (string) ($request->input('sourcePath') ?? $request->input('source_path', ''));
        $destinationPath = (string) ($request->input('destinationPath') ?? $request->input('destination_path', ''));
        if ($sourcePath === '' || $destinationPath === '') {
            return new JsonResponse(['message' => 'sourcePath and destinationPath are required'], 422);
        }

        return $this->fromEnginePayload($this->engine->moveObject($sourcePath, $destinationPath, TraceHeaders::fromRequest($request, true)));
    }

    /**
     * Deletes the object at the provided storage path.
     *
     * Expects the request to include a 'path' input; returns a JSON response built from the file engine's payload.
     *
     * @param Request $request The incoming HTTP request containing the 'path' parameter.
     * @return JsonResponse A JSON response containing the engine's payload; the HTTP status is taken from `_engine_http_status` in the payload or 200 if absent.
     */
    public function delete(Request $request): JsonResponse
    {
        $path = (string) $request->input('path', '');
        if ($path === '') {
            return new JsonResponse(['message' => 'path is required'], 422);
        }

        return $this->fromEnginePayload($this->engine->deleteObject($path, TraceHeaders::fromRequest($request, true)));
    }

    /**
     * Restore a quarantined object specified by the request path.
     *
     * @param Request $request HTTP request containing:
     *                        - `path` (string): required path of the object to restore.
     *                        - `forceReprocess` or `force_reprocess` (bool, optional): whether to force reprocessing; defaults to false.
     * @return JsonResponse A JSON response containing the engine payload. If the engine payload includes `_engine_http_status` that status will be used; if `path` is missing returns a 422 response with `['message' => 'path is required']`.
     */
    public function restore(Request $request): JsonResponse
    {
        $path = (string) $request->input('path', '');
        if ($path === '') {
            return new JsonResponse(['message' => 'path is required'], 422);
        }

        return $this->fromEnginePayload($this->engine->restoreQuarantinedObject(
            $path,
            $request->boolean('forceReprocess', $request->boolean('force_reprocess', false)),
            TraceHeaders::fromRequest($request, true)
        ));
    }

    /**
     * Build a JsonResponse from an engine payload, using an optional `_engine_http_status` entry as the response status.
     *
     * Removes the `_engine_http_status` key from the returned payload if present.
     *
     * @param array $payload Engine response payload which may include `_engine_http_status`.
     * @return JsonResponse A JsonResponse containing the payload (with `_engine_http_status` removed) and the HTTP status extracted from that key or 200 if absent.
     */
    private function fromEnginePayload(array $payload): JsonResponse
    {
        $status = (int) ($payload['_engine_http_status'] ?? 200);
        unset($payload['_engine_http_status']);

        return new JsonResponse($payload, $status);
    }
}
