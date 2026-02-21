<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Support\TraceHeaders;
use Illuminate\Http\Request;
use PHPUnit\Framework\TestCase;

final class TraceHeadersTest extends TestCase
{
    public function testTraceHeadersFallbackToGeneratedRequestId(): void
    {
        $request = Request::create('/folders', 'POST');

        $headers = TraceHeaders::fromRequest($request);

        self::assertArrayHasKey('X-Request-Id', $headers);
        self::assertArrayHasKey('X-Correlation-Id', $headers);
        self::assertSame($headers['X-Request-Id'], $headers['X-Correlation-Id']);
        self::assertArrayNotHasKey('tracestate', $headers);
    }

    public function testTraceHeadersPreserveInboundTracingHeaders(): void
    {
        $request = Request::create('/folders', 'POST', [], [], [], [
            'HTTP_X_REQUEST_ID' => 'req-1',
            'HTTP_X_CORRELATION_ID' => 'corr-1',
            'HTTP_TRACEPARENT' => '00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01',
            'HTTP_TRACESTATE' => 'vendor=value',
            'HTTP_BAGGAGE' => 'tenant=acme',
        ]);

        $headers = TraceHeaders::fromRequest($request);

        self::assertSame('req-1', $headers['X-Request-Id']);
        self::assertSame('corr-1', $headers['X-Correlation-Id']);
        self::assertSame('00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01', $headers['traceparent']);
        self::assertSame('vendor=value', $headers['tracestate']);
        self::assertSame('tenant=acme', $headers['baggage']);
    }
}
