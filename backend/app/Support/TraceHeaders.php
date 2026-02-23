<?php

namespace App\Support;

use Illuminate\Http\Request;

final class TraceHeaders
{
    /**
     * Build an associative array of trace-related headers extracted from the given HTTP request.
     *
     * Ensures an X-Request-Id is present (generates a be-<microseconds> fallback when missing)
     * and sets X-Correlation-Id to the request id when absent. Also includes request headers
     * `traceparent`, `tracestate`, `baggage`, and `Authorization` when they are present and non-empty.
     *
     * @param Request $request The incoming HTTP request to read headers from.
     * @return array<string,string> Header names mapped to their string values.
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
        self::maybeAttach($headers, 'Authorization', $request);

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
