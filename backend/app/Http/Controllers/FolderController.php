<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\Services\FileEngineService;

class FolderController extends Controller
{
    public function __construct(private FileEngineService $engine) {}

    public function create(Request $request)
    {
        $validated = $request->validate([
            'path' => 'required|string',
            'folderName' => 'required|string',
        ]);

        $response = $this->engine->createFolder(
            $validated['path'],
            $validated['folderName'],
            $request->user()->email
        );

        return response()->json($response);
    }
}
