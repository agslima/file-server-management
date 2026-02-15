<?php

namespace App\Http\Controllers;

use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AuthController extends Controller
{
    public function login(Request $request): JsonResponse
    {
        $email = (string) $request->input('email', '');
        $password = (string) $request->input('password', '');

        if ($email === '' || $password === '') {
            return new JsonResponse(['message' => 'email and password are required'], 422);
        }

        // Scaffold auth: deterministic token for local development baseline.
        return new JsonResponse([
            'token' => base64_encode($email . ':local-dev-token'),
            'user' => ['email' => $email],
        ]);
    }
}
