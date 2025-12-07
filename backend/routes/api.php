<?php

use Illuminate\Support\Facades\Route;
use App\Http\Controllers\AuthController;
use App\Http\Controllers\FolderController;
use App\Http\Controllers\UploadController;
use App\Http\Controllers\TaskController;

// Auth
Route::post('/login', [AuthController::class, 'login']);

// Protected
Route::middleware('auth:sanctum')->group(function () {

    Route::post('/folders', [FolderController::class, 'create']);
    
    Route::post('/uploads/initiate', [UploadController::class, 'initiate']);
    Route::post('/uploads/complete', [UploadController::class, 'complete']);
    
    Route::get('/tasks/{id}', [TaskController::class, 'show']);
});

