<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;

class FileEngineService
{
    private string $base;

    public function __construct()
    {
        $this->base = config('services.fileengine.base_url');
    }

    public function createFolder(string $path, string $folderName, string $user)
    {
        $res = Http::post("$this->base/folders", [
            'path' => $path,
            'folderName' => $folderName,
            'createdBy' => $user
        ]);

        return $res->json();
    }

    public function initiateUpload(array $data, string $user)
    {
        $data['createdBy'] = $user;

        $res = Http::post("$this->base/uploads/initiate", $data);
        return $res->json();
    }

    public function completeUpload(string $uploadId)
    {
        $res = Http::post("$this->base/uploads/complete", [
            'uploadId' => $uploadId
        ]);

        return $res->json();
    }

    public function getTask(string $id)
    {
        return Http::get("$this->base/tasks/$id")->json();
    }
}
