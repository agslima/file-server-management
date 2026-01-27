package services

import (
    "context"

    "github.com/example/file-engine/internal/adapters/queue/redisq"
)

type FileService struct {
    queue *redisq.RedisQueue
}

func NewFileService(q *redisq.RedisQueue) *FileService {
    return &FileService{queue: q}
}

func (s *FileService) QueueCreateFolder(
    ctx context.Context,
    taskID, parent, name, user string,
) error {

    payload := redisq.TaskPayload{
        ID:   taskID,
        Type: "create_folder",
        Params: map[string]string{
            "parent": parent,
            "name":   name,
            "user":   user,
        },
    }

    return s.Queue(ctx, payload)
}
