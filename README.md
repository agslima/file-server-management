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
