<?php

namespace App\Clients;

use RuntimeException;
use Throwable;

class FileEngineException extends RuntimeException
{
    public function __construct(
        public readonly int $status,
        public readonly string $codeValue,
        public readonly string $reason,
        public readonly bool $retryable,
        string $message,
        ?Throwable $previous = null,
    ) {
        parent::__construct($message, $status, $previous);
    }

    public function isRetryable(): bool
    {
        return $this->retryable || $this->status === 429 || $this->status >= 500;
    }
}
