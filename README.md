# File Server Management - Project Skeleton

This repository contains an initial project skeleton for the File Server Management application (PHP + Go hybrid).

See `docs/openapi.yaml` for the API spec.

```text

🧑‍💻 Users
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
+-------------+        +------------------+
|   Client    | -----> | gRPC / REST API  |
+-------------+        +------------------+
                                |
                                v
                     +------------------------+
                     | Authorization Layer    |
                     | (RBAC + ACL Resolver)  |
                     +------------------------+
                                |
                                v
                     +------------------------+
                     | Application Services   |
                     | (Command orchestration)|
                     +------------------------+
                                |
                                v
                     +------------------------+
                     | Task Queue (Redis)     |
                     +------------------------+
                                |
                                v
                     +------------------------+
                     | Worker Process         |
                     | (Filesystem execution)|
                     +------------------------+
                                |
                                v
                     +------------------------+
                     | Filesystem (LocalFS)   |
                     +------------------------+
```

```mermaid
flowchart LR A["Cliente (Frontend)"] --> B["API Gateway / Backend"]; B --> C["Validação de
Autenticação"]; C --> D{"Rota da API?"}; D -->|"Criar Recurso"| E["Controller: Create"]; D
-->|"Atualizar Recurso"| F["Controller: Update"]; D -->|"Consultar Dados"| G["Controller:
Read"]; D -->|"Excluir Recurso"| H["Controller: Delete"]; E --> I["Service Layer"]; F -->
I; G --> I; H --> I; I --> J["Repository / ORM"]; J --> K["Banco de Dados"]; K --> J; J -->
I; I --> B; B --> A;
```


```text
project-root/
├─ frontend/               # React / Next.js
│  ├─ components/
│  ├─ pages/
│  └─ services/
│  └─ tests/
|
├─ backend/                # Laravel API
│  ├─ app/
│  │   ├─ Http/Controllers/
│  │   ├─ Services/       # Chama Go File Engine
│  │   └─ Policies/
│  ├─ config/
│  ├─ database/migrations/
│  ├─ routes/
│  └─ tests/
|
├── file-engine/         # Go service
│   ├── api
│   │   └── proto
│   ├── build
│   │   └── docker
│   ├── cmd                 # Entrypoint
│   │   ├── file-engine
│   │   ├── gateway
│   │   ├── migrate
│   │   ├── server
│   │   └── worker
│   ├── db
│   │   ├── migrations
│   │   └── queries
│   ├── docs
│   ├── internal
│   │   ├── adapters
│   │   │   ├── fs
│   │   │   │   └── local
│   │   │   ├── queue
│   │   │   │   └── redisq
│   │   │   └── storage
│   │   │       ├── gcs
│   │   │       ├── local
│   │   │       └── s3
│   │   ├── app
│   │   │   └── tasks
│   │   ├── auth
│   │   ├── config
│   │   ├── di
│   │   ├── filesystem
│   │   ├── fs
│   │   ├── handlers
│   │   ├── logger
│   │   ├── server
│   │   ├── storage
│   │   └── worker
│   ├── pkg
│   │   └── generated
│   ├── proto
│   ├── scripts
│   │   └── scripts
|
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



