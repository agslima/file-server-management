<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
use App\Support\TraceHeaders;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class FolderController extends Controller
{
    private FileEngineService $engine;

    /**
     * Create a new FolderController instance and set the file engine service.
     *
     * @param FileEngineService $engine The file engine service used to perform folder operations.
     */
    public function __construct(FileEngineService $engine)
    {
        $this->engine = $engine;
    }

    /**
     * Create a folder at the specified path using the file engine and return the engine's response.
     *
     * Expects the HTTP request to include 'path' and 'folderName' (both required). Optionally accepts
     * 'requestedBy'; if omitted the authenticated user's email is used or 'system' as a fallback.
     *
     * @param \Illuminate\Http\Request $request Incoming request containing 'path', 'folderName', and optionally 'requestedBy'.
     * @return \Illuminate\Http\JsonResponse JSON payload returned by the file engine (the internal `_engine_http_status` key is removed) and an HTTP status code taken from `_engine_http_status` (defaults to 200). If 'path' or 'folderName' is missing, returns a 422 response with `{"message":"path and folderName are required"}`.
     */
    public function create(Request $request): JsonResponse
    {
        $path = (string) $request->input('path', '');
        $folderName = (string) $request->input('folderName', '');
        if ($path === '' || $folderName === '') {
            return new JsonResponse(['message' => 'path and folderName are required'], 422);
        }

        $requestedBy = (string) ($request->input('requestedBy') ?? optional($request->user())->email ?? 'system');

        $payload = $this->engine->createFolder($path, $folderName, $requestedBy, TraceHeaders::fromRequest($request, true));
        $status = (int) ($payload['_engine_http_status'] ?? 200);
        unset($payload['_engine_http_status']);

        return new JsonResponse($payload, $status);
    }
}
