Worker integration with LocalFs:

- The worker connects to Redis, pops tasks from 'tasks' list, and processes using FSProcessor.
- Configure environment variables:
  - REDIS_ADDR (default: localhost:6379)
  - FILE_BASE_ROOT (default: /mnt/files)
- Example task (JSON pushed to Redis 'tasks'):
  {"id":"tsk1","type":"create_folder","params":{"path":"projects/demo","folder":"newfolder"}}