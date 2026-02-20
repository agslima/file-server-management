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

        $traceparent = trim((string) $request->header('traceparent', ''));
        if ($traceparent !== '') {
            $headers['traceparent'] = $traceparent;
        }

        $baggage = trim((string) $request->header('baggage', ''));
        if ($baggage !== '') {
            $headers['baggage'] = $baggage;
        }

        return $headers;
    }
}

