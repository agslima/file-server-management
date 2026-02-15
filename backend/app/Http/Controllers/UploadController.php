<?php

namespace App\Http\Controllers;

use App\Services\FileEngineService;
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
        $filename = (string) $request->input('filename', '');
        $mimeType = (string) $request->input('mimeType', '');

        if ($path === '' || $filename === '' || $mimeType === '') {
            return new JsonResponse(['message' => 'path, filename and mimeType are required'], 422);
        }

        $requestedBy = (string) ($request->input('requestedBy') ?? optional($request->user())->email ?? 'system');

        return new JsonResponse($this->engine->initiateUpload([
            'path' => $path,
            'filename' => $filename,
            'mimeType' => $mimeType,
        ], $requestedBy));
    }

    public function complete(Request $request): JsonResponse
    {
        $uploadId = (string) $request->input('uploadId', '');
        if ($uploadId === '') {
            return new JsonResponse(['message' => 'uploadId is required'], 422);
        }

        return new JsonResponse($this->engine->completeUpload($uploadId));
    }
}
