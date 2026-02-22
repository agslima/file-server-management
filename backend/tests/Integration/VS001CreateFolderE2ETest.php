<?php

declare(strict_types=1);

namespace Tests\Integration;

use PHPUnit\Framework\TestCase;

final class VS001CreateFolderE2ETest extends TestCase
{
    public function testBackendForwardsCreateFolderAndPollsUntilSuccess(): void
    {
        $backendUrl = (string) (getenv('BACKEND_URL') ?: 'http://localhost:8081');
        if (!$this->isHealthy($backendUrl . '/healthz')) {
            self::markTestSkipped('backend is not reachable; start docker compose stack for VS-001 E2E validation');
        }
        if (!$this->isFileEngineServiceRunning()) {
            self::markTestSkipped('file-engine service is not running in docker compose; start full stack for VS-001 E2E validation');
        }

        $parentPath = 'tenants/acme';
        $folderName = 'phpunit-vs001-' . time();

        $create = $this->requestJson('POST', $backendUrl . '/folders', [
            'path' => $parentPath,
            'folderName' => $folderName,
            'requestedBy' => 'phpunit-vs001@example.com',
        ]);

        self::assertSame(200, $create['status']);
        self::assertNotEmpty($create['json']['taskId'] ?? null);

        $taskId = (string) $create['json']['taskId'];
        $deadline = time() + 90;
        $taskStatus = 'queued';

        while (time() < $deadline) {
            $task = $this->requestJson('GET', $backendUrl . '/tasks/' . rawurlencode($taskId));
            self::assertSame(200, $task['status']);
            $taskStatus = (string) ($task['json']['status'] ?? '');
            if ($taskStatus === 'success') {
                break;
            }
            self::assertNotSame('failed', $taskStatus, 'task transitioned to failed unexpectedly');
            usleep(500000);
        }

        self::assertSame('success', $taskStatus);

        $targetPath = escapeshellarg('/mnt/files/' . $parentPath . '/' . $folderName);
        $folderCheck = trim((string) shell_exec(sprintf(
            '(docker compose exec -T file-engine test -d %1$s || docker-compose exec -T file-engine test -d %1$s) >/dev/null 2>&1 && echo true || echo false',
            $targetPath
        )));

        self::assertSame('true', $folderCheck);
    }

    private function isHealthy(string $url): bool
    {
        $result = $this->requestJson('GET', $url);

        return $result['status'] === 200;
    }

    private function isFileEngineServiceRunning(): bool
    {
        $output = shell_exec(
            'docker compose ps --services --status running 2>/dev/null || docker-compose ps --services --filter "status=running" 2>/dev/null'
        );
        if (!is_string($output) || trim($output) === '') {
            return false;
        }
        $services = preg_split('/\R+/', trim($output));
        if (!is_array($services)) {
            return false;
        }
        return in_array('file-engine', $services, true);
    }

    /**
     * @return array{status:int, json:array<string,mixed>}
     */
    private function requestJson(string $method, string $url, ?array $payload = null): array
    {
        $headers = "Content-Type: application/json\r\n";
        $options = [
            'http' => [
                'method' => $method,
                'header' => $headers,
                'ignore_errors' => true,
                'timeout' => 10,
            ],
        ];

        if ($payload !== null) {
            $options['http']['content'] = json_encode($payload, JSON_THROW_ON_ERROR);
        }

        $context = stream_context_create($options);
        $body = @file_get_contents($url, false, $context);
        $rawHeaders = $http_response_header ?? [];
        $statusLine = is_array($rawHeaders) && isset($rawHeaders[0]) ? (string) $rawHeaders[0] : '';
        preg_match('/\s(\d{3})\s/', $statusLine, $matches);
        $status = isset($matches[1]) ? (int) $matches[1] : 0;

        $json = [];
        if (is_string($body) && $body !== '') {
            $decoded = json_decode($body, true);
            if (is_array($decoded)) {
                $json = $decoded;
            }
        }

        return [
            'status' => $status,
            'json' => $json,
        ];
    }
}
