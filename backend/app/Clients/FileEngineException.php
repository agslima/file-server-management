<?php

namespace App\Clients;

use RuntimeException;

class FileEngineException extends RuntimeException
{
    public function __construct(
        public readonly int $status,
        public readonly string $codeValue,
        public readonly string $reason,
        public readonly bool $retryable,
        string $message,
    ) {
        parent::__construct($message, $status);
    }

    public function isRetryable(): bool
    {
        return $this->retryable || $this->status === 429 || $this->status >= 500;
    }
}
