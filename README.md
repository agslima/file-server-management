# File Server Management - Project Skeleton

This repository contains an initial project skeleton for the File Server Management application (PHP + Go hybrid).

See `docs/openapi.yaml` for the API spec.

```text

🧑‍💻 Usuário
                 │
                 ▼
      ┌─────────────────────┐
      │    Frontend (Web)   │
      │ React / Next.js     │
      └─────────────────────┘
                 │ REST/GraphQL
                 ▼
     ┌─────────────────────────┐
     │   API Gateway (Laravel) │
     │  Autenticação, RBAC     │
     └─────────────────────────┘
                 │ Event/REST/gRPC
                 ▼
     ┌─────────────────────────┐
     │  File Engine (Go)       │
     │ Manipulação real:       │
     │ SMB / SFTP / NFS        │
     │ Execuções concorrentes  │
     └─────────────────────────┘
                 │
                 ▼
     ┌─────────────────────────┐
     │ File Server Híbrido     │
     │ (Local / Cloud / AD)    │
     └─────────────────────────┘
```
```text
project-root/
├─ frontend/               # React / Next.js
│  ├─ components/
│  ├─ pages/
│  └─ services/
├─ backend/                # Laravel API
│  ├─ app/
│  │   ├─ Http/Controllers/
│  │   ├─ Services/       # Chama Go File Engine
│  │   └─ Policies/
│  ├─ database/migrations/
│  ├─ routes/
│  └─ tests/
├─ file-engine-go/         # Go service
│  ├─ cmd/                 # Entrypoint
│  ├─ internal/
│  │   ├─ filesystem/
│  │   ├─ validators/
│  │   └─ uploader/
│  └─ pkg/
├─ docker/                 # Dockerfiles + Compose
└─ docs/
```
```Yaml
Usuário → Frontend: Solicita nova pasta
Frontend → API Laravel: POST /folders
API Laravel → Validator: Verifica regras de nome
API Laravel → Queue (Redis/Kafka): Envia tarefa
File Engine Go → File Server: Cria pasta no caminho correto
File Engine Go → API Laravel: Retorna status
API Laravel → Audit Log DB: Registra ação
API Laravel → Frontend: Notifica sucesso/erro
```



