<?php

declare(strict_types=1);

use App\Http\Controllers\FolderController;
use App\Http\Controllers\TaskController;
use App\Services\FileEngineService;
use Illuminate\Http\Client\Factory as HttpFactory;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

require __DIR__ . '/vendor/autoload.php';

$request = Request::capture();
$method = strtoupper($request->method());
$path = '/' . ltrim($request->path(), '/');

$service = new FileEngineService(
    new HttpFactory(),
    getenv('FILE_ENGINE_URL') ?: 'http://file-engine:8080/v1',
    getenv('FILE_ENGINE_BEARER_TOKEN') ?: null,
);

if ($method === 'GET' && $path === '/healthz') {
    respond(new JsonResponse(['status' => 'ok'], 200));
}

if ($method === 'POST' && $path === '/folders') {
    $controller = new FolderController($service);
    respond($controller->create($request));
}

if ($method === 'GET' && preg_match('#^/tasks/([^/]+)$#', $path, $matches) === 1) {
    $controller = new TaskController($service);
    respond($controller->show($request, $matches[1]));
}

respond(new JsonResponse(['message' => 'Not Found'], 404));

function respond(JsonResponse $response): void
{
    http_response_code($response->getStatusCode());
    header('Content-Type: application/json');

    foreach ($response->headers->allPreserveCaseWithoutCookies() as $name => $values) {
        foreach ($values as $value) {
            header($name . ': ' . $value, false);
        }
    }

    echo $response->getContent();
    exit;
}
