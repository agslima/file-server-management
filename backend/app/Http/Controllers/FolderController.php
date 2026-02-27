<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
use App\Support\TraceHeaders;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class FolderController extends Controller
{
    private FileEngineService $engine;

    public function __construct(FileEngineService $engine)
    {
        $this->engine = $engine;
    }

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
