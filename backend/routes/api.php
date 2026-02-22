<?php

use App\Http\Controllers\AuthController;
use App\Http\Controllers\FolderController;
use App\Http\Controllers\TaskController;
use App\Http\Controllers\ObjectMutationController;
use App\Http\Controllers\UploadController;
use Illuminate\Support\Facades\Route;

Route::post('/login', [AuthController::class, 'login']);

Route::post('/folders', [FolderController::class, 'create']);
Route::post('/uploads/initiate', [UploadController::class, 'initiate']);
Route::put('/uploads/{uploadId}/chunk', [UploadController::class, 'uploadChunk']);
Route::post('/uploads/{uploadId}/complete', [UploadController::class, 'complete']);
Route::post('/objects/move', [ObjectMutationController::class, 'move']);
Route::post('/objects/delete', [ObjectMutationController::class, 'delete']);
Route::post('/objects/restore', [ObjectMutationController::class, 'restore']);
Route::get('/tasks/{id}', [TaskController::class, 'show']);
