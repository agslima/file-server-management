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
        return new JsonResponse($this->engine->getTask($id));
    }
}
