# Delcom Blog — Backend (Golang Gin) · SK P9

REST API platform blog multi-tier (Publik · Penulis · Admin) dengan **AI Blog Generator**
yang terintegrasi **Search Engine** + OpenAI Chat Completions.
Arsitektur *Clean Architecture & Feature-First*, Golang + Gin + GORM + PostgreSQL 17.

- Port: **8081** · Swagger UI: http://localhost:8081/swagger/index.html
- Akun bawaan (seeder): `admin / admin` (admin) dan `member / member` (member)

## Menjalankan

```bash
psql -U postgres -c "CREATE DATABASE pabwe_p9_blog;"
cp .env.example .env        # sesuaikan bila perlu
go mod tidy                 # mengunduh dependensi & membuat go.sum
go run cmd/server/main.go
```

## Pengujian (100% statement coverage)

```bash
go test -v -cover ./internal/...
```

## Build binary

```bash
go build -o server.exe cmd/server/main.go   # Windows
go build -o server cmd/server/main.go       # Linux / macOS
```

`.env` dibaca dari direktori aktif, tempatkan di samping binary.

## Format respons

```json
{ "status": "success|fail|error", "message": "…", "data": { } }
```
