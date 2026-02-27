<?php

namespace App\Clients;

use RuntimeException;

class FileEngineException extends RuntimeException
{
    /**
     * Create a FileEngineException carrying status, code, reason, retryability, and message.
     *
     * @param int    $status    Numeric status code associated with the error (e.g., HTTP status).
     * @param string $codeValue Machine-readable error code.
     * @param string $reason    Short human-readable reason for the error.
     * @param bool   $retryable Whether the operation that caused the error is considered retryable.
     * @param string $message   Detailed message used as the exception message.
     */
    public function __construct(
        public readonly int $status,
        public readonly string $codeValue,
        public readonly string $reason,
        public readonly bool $retryable,
        string $message,
    ) {
        parent::__construct($message, $status);
    }

    /**
     * Determines whether this exception represents an error that should be retried.
     *
     * @return bool `true` if the exception is marked retryable, or the HTTP status is 429, or the HTTP status is 500 or greater; `false` otherwise.
     */
    public function isRetryable(): bool
    {
        return $this->retryable || $this->status === 429 || $this->status >= 500;
    }
}
