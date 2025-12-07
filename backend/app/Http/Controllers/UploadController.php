<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\Services\FileEngineService;

class UploadController extends Controller
{
    public function __construct(private FileEngineService $engine) {}

    public function initiate(Request $request)
    {
        $validated = $request->validate([
            'path' => 'required',
            'filename' => 'required',
            'mimeType' => 'required'
        ]);

        return $this->engine->initiateUpload(
            $validated,
            $request->user()->email
        );
    }

    public function complete(Request $request)
    {
        $validated = $request->validate([
            'uploadId' => 'required|string'
        ]);

        return $this->engine->completeUpload(
            $validated['uploadId']
        );
    }
}
