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
        $controller = new UploadController(new FakeFileEngineService());

        $initiate = $controller->initiate(Request::create('/uploads/initiate', 'POST', [
            'path' => 'tenants/acme/reports',
            'filename' => 'a.txt',
            'mimeType' => 'text/plain',
            'requestedBy' => 'owner@example.com',
        ]));
        self::assertSame(200, $initiate->getStatusCode());
        self::assertSame('upl-1', $initiate->getData(true)['uploadId']);

        $complete = $controller->complete(Request::create('/uploads/complete', 'POST', [
            'uploadId' => 'upl-1',
        ]));
        self::assertSame(200, $complete->getStatusCode());
        self::assertSame('queued', $complete->getData(true)['status']);
    }

    public function testGetTaskEndpointReturnsTaskStatus(): void
    {
        $controller = new TaskController(new FakeFileEngineService());

        $response = $controller->show('task-123');

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
    public function __construct()
    {
        parent::__construct(new HttpFactory(), 'http://example.test/v1');
    }

    public function createFolder(string $path, string $folderName, string $requestedBy): array
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

    public function initiateUpload(array $payload, string $requestedBy): array
    {
        return [
            '_engine_http_status' => 200,
            'uploadId' => 'upl-1',
            'uploadUrl' => 'http://upload.local/upl-1',
            'requestedBy' => $requestedBy,
            'payload' => $payload,
        ];
    }

    public function completeUpload(string $uploadId): array
    {
        return [
            '_engine_http_status' => 200,
            'taskId' => 'task-complete',
            'status' => 'queued',
            'uploadId' => $uploadId,
        ];
    }

    public function getTask(string $id): array
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
    public function __construct()
    {
        parent::__construct(new HttpFactory(), 'http://example.test/v1');
    }

    public function createFolder(string $path, string $folderName, string $requestedBy): array
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
