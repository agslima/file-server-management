<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\Services\FileEngineService;

class TaskController extends Controller
{
    public function __construct(private FileEngineService $engine) {}

    public function show(string $id)
    {
        return $this->engine->getTask($id);
    }
}
