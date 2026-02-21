<?php

namespace App\Support;

use Illuminate\Http\Request;

final class TraceHeaders
{
    /**
     * @return array<string,string>
     */
    public static function fromRequest(Request $request): array
    {
        $requestId = trim((string) $request->header('X-Request-Id', ''));
        if ($requestId === '') {
            $requestId = sprintf('be-%d', (int) (microtime(true) * 1000000));
        }

        $correlationId = trim((string) $request->header('X-Correlation-Id', ''));
        if ($correlationId === '') {
            $correlationId = $requestId;
        }

        $headers = [
            'X-Request-Id' => $requestId,
            'X-Correlation-Id' => $correlationId,
        ];

        self::maybeAttach($headers, 'traceparent', $request);
        self::maybeAttach($headers, 'tracestate', $request);
        self::maybeAttach($headers, 'baggage', $request);

        return $headers;
    }

    /**
     * @param array<string,string> $headers
     */
    private static function maybeAttach(array &$headers, string $headerName, Request $request): void
    {
        $value = trim((string) $request->header($headerName, ''));
        if ($value !== '') {
            $headers[$headerName] = $value;
        }
    }
}
