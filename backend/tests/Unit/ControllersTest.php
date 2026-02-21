<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Http\Controllers\AuthController;
use App\Http\Controllers\FolderController;
use App\Http\Controllers\TaskController;
use App\Http\Controllers\UploadController;
use App\Services\FileEngineService;
use Illuminate\Http\Client\Factory as HttpFactory;
use Illuminate\Http\Request;
use PHPUnit\Framework\TestCase;

final class ControllersTest extends TestCase
{
    public function testCreateFolderReturnsTaskPayload(): void
    {
        $controller = new FolderController(new FakeFileEngineService());
        $request = Request::create('/folders', 'POST', [
            'path' => 'tenants/acme',
            'folderName' => 'reports',
            'requestedBy' => 'owner@example.com',
        ]);

        $response = $controller->create($request);

        self::assertSame(200, $response->getStatusCode());
        self::assertSame('queued', $response->getData(true)['status']);
    }

    public function testCreateFolderValidationFailsWithoutRequiredFields(): void
    {
        $controller = new FolderController(new FakeFileEngineService());
        $request = Request::create('/folders', 'POST', []);

        $response = $controller->create($request);

        self::assertSame(422, $response->getStatusCode());
    }

    public function testCreateFolderPropagatesEnginePermissionDeniedAuthoritatively(): void
    {
        $controller = new FolderController(new DeniedFileEngineService());
        $request = Request::create('/folders', 'POST', [
            'path' => 'tenants/beta',
            'folderName' => 'secret',
            'requestedBy' => 'owner@example.com',
        ]);

        $response = $controller->create($request);

        self::assertSame(403, $response->getStatusCode());
        self::assertSame('tenant access denied', $response->getData(true)['message']);
    }

    public function testInitiateAndCompleteUploadEndpoints(): void
    {
        $engine = new FakeFileEngineService();
        $controller = new UploadController($engine);

        $initRequest = Request::create('/uploads/initiate', 'POST', [
            'path' => 'tenants/acme/reports',
            'filename' => 'a.txt',
            'mimeType' => 'text/plain',
            'requestedBy' => 'owner@example.com',
        ]);
        $initRequest->headers->set('X-Request-Id', 'req-upload-test');
        $initiate = $controller->initiate($initRequest);
        self::assertSame(200, $initiate->getStatusCode());
        self::assertSame('upl-1', $initiate->getData(true)['upload_id']);

        $chunkRequest = Request::create('/uploads/upl-1/chunk?offset=0', 'PUT', [], [], [], [], 'hello');
        $chunkRequest->headers->set('X-Request-Id', 'req-upload-test');
        $chunk = $controller->uploadChunk($chunkRequest, 'upl-1');
        self::assertSame(202, $chunk->getStatusCode());

        $completeRequest = Request::create('/uploads/upl-1/complete', 'POST');
        $completeRequest->headers->set('X-Request-Id', 'req-upload-test');
        $complete = $controller->complete($completeRequest, 'upl-1');
        self::assertSame(200, $complete->getStatusCode());
        self::assertSame('completed', $complete->getData(true)['status']);
        self::assertSame('req-upload-test', $engine->lastRequestId);
    }

    public function testGetTaskEndpointReturnsTaskStatus(): void
    {
        $controller = new TaskController(new FakeFileEngineService());

        $response = $controller->show(Request::create('/tasks/task-123', 'GET'), 'task-123');

        self::assertSame(200, $response->getStatusCode());
        self::assertSame('task-123', $response->getData(true)['taskId']);
    }

    public function testAuthLoginReturnsToken(): void
    {
        $controller = new AuthController();
        $response = $controller->login(Request::create('/login', 'POST', [
            'email' => 'owner@example.com',
            'password' => 'secret',
        ]));

        self::assertSame(200, $response->getStatusCode());
        $payload = $response->getData(true);
        self::assertArrayHasKey('token', $payload);
        self::assertSame('owner@example.com', $payload['user']['email']);
    }
}

final class FakeFileEngineService extends FileEngineService
{
    public string $lastRequestId = '';

    public function __construct()
    {
        parent::__construct(new HttpFactory(), 'http://example.test/v1');
    }

    public function createFolder(string $path, string $folderName, string $requestedBy, array $traceHeaders = []): array
    {
        return [
            '_engine_http_status' => 200,
            'taskId' => 'task-123',
            'status' => 'queued',
            'path' => $path,
            'folderName' => $folderName,
            'requestedBy' => $requestedBy,
        ];
    }

    public function initiateUpload(array $payload, array $traceHeaders = [], string $idempotencyKey = ""): array
    {
        $this->lastRequestId = (string) ($traceHeaders['X-Request-Id'] ?? '');
        return [
            '_engine_http_status' => 200,
            'upload_id' => 'upl-1',
            'upload_url' => 'http://upload.local/upl-1:chunk',
            'staging_token' => 'upl-1',
            'payload' => $payload,
        ];
    }


    public function uploadChunk(string $uploadId, int $offset, string $content, array $traceHeaders = []): array
    {
        $this->lastRequestId = (string) ($traceHeaders['X-Request-Id'] ?? '');
        return [
            '_engine_http_status' => 202,
            'upload_id' => $uploadId,
            'offset' => $offset,
            'bytes' => strlen($content),
        ];
    }

    public function completeUpload(string $uploadId, array $traceHeaders = [], string $idempotencyKey = ""): array
    {
        $this->lastRequestId = (string) ($traceHeaders['X-Request-Id'] ?? '');
        return [
            '_engine_http_status' => 200,
            'upload_id' => $uploadId,
            'status' => 'completed',
        ];
    }

    public function getTask(string $id, array $traceHeaders = []): array
    {
        return [
            '_engine_http_status' => 200,
            'taskId' => $id,
            'status' => 'success',
            'message' => 'done',
        ];
    }
}

final class DeniedFileEngineService extends FileEngineService
{
    public string $lastRequestId = '';

    public function __construct()
    {
        parent::__construct(new HttpFactory(), 'http://example.test/v1');
    }

    public function createFolder(string $path, string $folderName, string $requestedBy, array $traceHeaders = []): array
    {
        return [
            '_engine_http_status' => 403,
            'message' => 'tenant access denied',
            'path' => $path,
            'folderName' => $folderName,
            'requestedBy' => $requestedBy,
        ];
    }
}
