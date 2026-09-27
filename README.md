# KDA Auth Service

REST API autentikasi berbasis Go dan Echo dengan dukungan login email/password, Google OAuth2, JWT, PostgreSQL, Redis, health/readiness check, Swagger UI, Docker, dan Kubernetes.

## Fitur

- Registrasi user menggunakan email dan password
- Password hashing dengan bcrypt
- Login email/password dan penerbitan JWT HS256
- Login Google menggunakan OAuth2 authorization-code flow
- Auto-register user saat pertama kali login melalui Google
- Penyimpanan session token dan revocation logout di Redis
- PostgreSQL sebagai penyimpanan data user
- Liveness dan readiness endpoint
- Swagger UI dengan contoh request/response sesuai implementasi
- Unit test untuk handler, service, repository, middleware, OAuth2, JWT, response, dan validation

## Teknologi

| Komponen | Teknologi |
|---|---|
| Bahasa | Go 1.27 |
| HTTP framework | Echo v4 |
| Database | PostgreSQL 15 |
| ORM | GORM |
| Cache/session | Redis 7 |
| Authentication | JWT HS256 dan Google OAuth2 |
| Validation | go-playground/validator |
| API documentation | Swaggo dan Swagger UI |
| Deployment | Docker Compose dan Kubernetes |

## Arsitektur

Proyek menggunakan pendekatan hexagonal. Handler HTTP bergantung pada service melalui port, sedangkan service mengakses PostgreSQL dan Redis melalui repository interface.

```mermaid
flowchart LR
    Client --> Echo[Echo handlers]
    Echo --> AuthService[Auth service]
    AuthService --> UserPort[User repository port]
    AuthService --> TokenPort[Token repository port]
    UserPort --> PostgreSQL
    TokenPort --> Redis
    Echo --> GoogleAuth[Google authorization endpoint]
    Echo --> GoogleToken[Google token endpoint]
    AuthService --> GoogleInfo[Google tokeninfo endpoint]
```

Struktur direktori:

```text
cmd/api/                         application bootstrap
docs/                            Swagger source dan generated contract
internal/core/domain/            entity dan request/response model
internal/core/ports/             inbound/outbound interfaces
internal/core/services/          authentication business rules
internal/adapters/handlers/      routes, HTTP handlers, middleware
internal/adapters/repositories/  PostgreSQL dan Redis adapters
pkg/config/                      PostgreSQL, Redis, dan OAuth configuration
pkg/jwt/                         JWT generation
pkg/response/                    standard response envelope
pkg/utils/                       validation error formatter
migrations/                      SQL migrations
deploy/                          Docker dan Kubernetes manifests
```

## Prasyarat

- Go 1.27+
- PostgreSQL 15+
- Redis 7+
- Docker dan Docker Compose, jika memakai container
- [`golang-migrate`](https://github.com/golang-migrate/migrate) untuk perintah migration
- [`swag`](https://github.com/swaggo/swag) jika ingin generate ulang Swagger

Instalasi tool development:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/swaggo/swag/cmd/swag@v1.16.6
```

## Konfigurasi environment

Salin file contoh:

```bash
cp .env.example .env
```

| Variable | Wajib | Keterangan |
|---|---:|---|
| `PORT` | Tidak | Port HTTP; default aplikasi `8080` |
| `DB_HOST` | Ya | Host PostgreSQL |
| `DB_PORT` | Ya | Port PostgreSQL |
| `DB_USER` | Ya | Username PostgreSQL |
| `DB_PASSWORD` | Ya | Password PostgreSQL |
| `DB_NAME` | Ya | Nama database |
| `REDIS_HOST` | Ya | Alamat Redis dalam format `host:port` |
| `REDIS_PASSWORD` | Tidak | Password Redis; kosong jika Redis lokal tidak memakai autentikasi |
| `JWT_SECRET` | Ya | Secret penandatanganan JWT |
| `JWT_EXPIRATION_HOURS` | Tidak | Masa berlaku JWT; service menggunakan `24` jam jika kosong atau tidak valid |
| `GOOGLE_CLIENT_ID` | Untuk Google login | OAuth Web Client ID sekaligus audience ID token |
| `GOOGLE_CLIENT_SECRET` | Untuk Google login | OAuth Web Client Secret; hanya boleh tersedia di backend |
| `GOOGLE_REDIRECT_URL` | Untuk Google login | Callback URI yang harus sama persis dengan konfigurasi Google Cloud |

Contoh callback lokal:

```env
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback
```

Jangan memakai nilai secret dari contoh deployment untuk production.

## Menjalankan aplikasi

### Opsi 1 — API di host, dependency di Docker

Cara ini paling mudah untuk development karena aplikasi membaca `.env` secara langsung.

```bash
docker compose -f deploy/docker-compose.local.yml up -d postgres redis
make migrate-up
make run
```

### Opsi 2 — Seluruh stack di Docker Compose

```bash
make up
make migrate-up
```

`make migrate-up` dijalankan dari host dan memakai PostgreSQL yang diekspos di `localhost:${DB_PORT}`. Pastikan isi `.env` sesuai dengan port dan credential di Compose.

Hentikan stack:

```bash
make down
```

Target tambahan yang tersedia:

```bash
make dev        # foreground, memakai docker-compose
make dev-build  # rebuild image/container lalu foreground
```

### Verifikasi aplikasi

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

Response sehat:

```json
{"message":"I'm healthy"}
```

Response ready:

```json
{"message":"I'm ready"}
```

Jika PostgreSQL atau Redis tidak dapat dijangkau, `/ready` mengembalikan `503`, misalnya:

```json
{
  "postgres": "down",
  "redis": "up"
}
```

## Database migration

Migration awal membuat extension UUID, tabel `users`, constraint unik email/Google ID, dan index email.

```bash
make migrate-up
make migrate-down
make migrate-create name=create_new_table
```

Nilai koneksi migration dibentuk oleh Makefile:

```text
postgres://${DB_USER}:${DB_PASSWORD}@localhost:${DB_PORT}/${DB_NAME}?sslmode=disable
```

## Dokumentasi Swagger

Setelah API berjalan:

- Swagger UI: [http://localhost:8080/swagger](http://localhost:8080/swagger)
- Swagger JSON runtime: [http://localhost:8080/swagger/doc.json](http://localhost:8080/swagger/doc.json)
- Generated YAML: [`docs/swagger.yaml`](docs/swagger.yaml)
- Generated JSON: [`docs/swagger.json`](docs/swagger.json)

Kontrak dokumentasi dipisahkan dari kode runtime dan bersumber dari [`docs/spec/api_annotations.go`](docs/spec/api_annotations.go). Generate ulang dengan:

```bash
swag init -g api_annotations.go -d docs/spec -o docs
```

Test di `docs/docs_test.go` memastikan `docs.go`, `swagger.json`, dan `swagger.yaml` tetap konsisten serta memuat seluruh route, method, status code, dan response example.

Untuk mencoba endpoint logout di Swagger:

1. Jalankan register dan login.
2. Salin `data.access_token` dari response login.
3. Klik **Authorize**.
4. Masukkan nilai `Bearer <access_token>`.
5. Jalankan `POST /api/v1/auth/logout`.

## Endpoint API

| Method | Path | Auth | Status utama | Keterangan |
|---|---|---|---|---|
| `GET` | `/health` | Tidak | `200` | Liveness process |
| `GET` | `/ready` | Tidak | `200`, `503` | Konektivitas PostgreSQL dan Redis |
| `POST` | `/api/v1/auth/register` | Tidak | `201`, `400`, `401` | Registrasi email/password |
| `POST` | `/api/v1/auth/login` | Tidak | `200`, `400`, `401` | Login email/password |
| `GET` | `/api/v1/auth/google/login` | Tidak | `307` | Redirect browser ke Google |
| `GET` | `/api/v1/auth/google/callback` | Tidak | `200`, `401` | Exchange authorization code dan menerbitkan JWT aplikasi |
| `POST` | `/api/v1/auth/logout` | Bearer JWT | `200`, `401` | Menghapus session token dari Redis |

### Register

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123","name":"Jane Doe"}'
```

Response `201`:

```json
{
  "status_code": 201,
  "status": "success",
  "message": "registered successfully"
}
```

Input tidak valid menghasilkan `400`. Kegagalan pada service registrasi menghasilkan `401`:

```json
{
  "status_code": 401,
  "status": "failed",
  "message": "register failed"
}
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123"}'
```

Response `200`:

```json
{
  "status_code": 200,
  "status": "success",
  "message": "logged in successfully",
  "data": {
    "access_token": "<jwt>",
    "token_type": "Bearer",
    "expires_in": 86400
  }
}
```

Credential salah menghasilkan `401`:

```json
{
  "status_code": 401,
  "status": "failed",
  "message": "login failed"
}
```

### Validation error

```json
{
  "status_code": 400,
  "status": "failed",
  "message": "invalid request body",
  "errors": {
    "email": "Invalid email format",
    "password": "This field is required"
  }
}
```

Untuk malformed JSON, field `errors` dapat tidak tersedia.

### Google OAuth2 login

1. Buat OAuth Client dengan tipe **Web application** di Google Cloud Console.
2. Daftarkan nilai `GOOGLE_REDIRECT_URL` sebagai **Authorized redirect URI**. Scheme, host, port, path, dan trailing slash harus sama persis.
3. Isi `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, dan `GOOGLE_REDIRECT_URL`.
4. Restart API.
5. Buka endpoint berikut langsung melalui browser:

```text
http://localhost:8080/api/v1/auth/google/login
```

Backend mengarahkan browser ke Google dengan scope `openid`, `email`, dan `profile`. Setelah callback berhasil, API mengembalikan response login berisi JWT aplikasi.

Jika memakai tunnel atau domain HTTPS, contoh callback-nya:

```text
https://your-domain.example/api/v1/auth/google/callback
```

Nilai tersebut wajib dipasang di environment dan Google Cloud Console.

### Logout

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H 'Authorization: Bearer <access_token>'
```

Response `200`:

```json
{
  "status_code": 200,
  "status": "success",
  "message": "logged out successfully"
}
```

JWT hilang, invalid, atau expired ditolak middleware:

```json
{"error":"invalid or expired token"}
```

Kegagalan revocation menghasilkan:

```json
{
  "status_code": 401,
  "status": "failed",
  "message": "logout failed"
}
```

## Testing

Jalankan seluruh test:

```bash
make test
```

Coverage per fungsi:

```bash
make test-cover
```

Race detector dan static analysis:

```bash
go test -race ./...
go vet ./...
```

Test menggunakan SQL mock, fake Redis commands, fake repository, dan mocked HTTP transport. PostgreSQL, Redis, dan koneksi Google nyata tidak dibutuhkan untuk unit test.

Cakupan skenario utama:

- request binding dan seluruh validation error
- register/login/logout sukses dan gagal
- bcrypt password hashing
- JWT generation, signature invalid, dan token expired
- Redis session store, validation, TTL, dan revocation
- PostgreSQL create/find/not-found/database error
- Google OAuth redirect, code exchange, missing ID token, dan service failure
- Google tokeninfo network error, audience mismatch, serta auto-register
- health/readiness saat dependency up atau down
- seluruh route serta konsistensi generated Swagger

## Build

Binary lokal:

```bash
mkdir -p bin
make build
./bin/api
```

Docker image:

```bash
docker build -f deploy/Dockerfile -t kda-auth-service:latest .
```

## Deployment Kubernetes

Manifest tersedia di `deploy/k8s/`:

- `redis.yaml`: ConfigMap, Secret, Service, dan single-replica Redis
- `auth_service.yaml`: Secret, NodePort Service, dan two-replica auth service

Manifest PostgreSQL tidak tersedia di direktori tersebut. Sediakan PostgreSQL secara terpisah dan pastikan `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, serta `DB_NAME` pada Secret auth service menunjuk database yang benar.

### Persiapan

1. Build dan push image:

   ```bash
   docker build -f deploy/Dockerfile -t <registry>/auth-service:<tag> .
   docker push <registry>/auth-service:<tag>
   ```

2. Ganti `your-registry/auth-service:latest` di `deploy/k8s/auth_service.yaml` dengan image tersebut.
3. Ganti seluruh example secret sebelum deploy.
4. Tambahkan `GOOGLE_CLIENT_SECRET` dan `GOOGLE_REDIRECT_URL` ke Secret auth service jika Google OAuth digunakan. Redirect URL harus menunjuk alamat publik callback.
5. Siapkan PostgreSQL yang dapat diakses dari cluster. Manifest auth service saat ini mengharapkan hostname `postgres-service`, kecuali `DB_HOST` diubah.

### Apply manifest

```bash
kubectl apply -f deploy/k8s/redis.yaml
kubectl apply -f deploy/k8s/auth_service.yaml
```

Verifikasi:

```bash
kubectl get pods
kubectl get services
kubectl rollout status deployment/auth-service
```

Manifest belum menyediakan migration Job. Jalankan migration sebagai Job terpisah atau dari environment yang dapat mengakses PostgreSQL sebelum menerima traffic aplikasi.

## Troubleshooting

### `/ready` mengembalikan 503

- Pastikan PostgreSQL sudah menerima koneksi dan migration telah dijalankan.
- Pastikan `DB_HOST`, `DB_PORT`, dan credential benar.
- Pastikan `REDIS_HOST` memakai format `host:port`.
- Jika Redis memakai password, nilai `REDIS_PASSWORD` API harus sama dengan konfigurasi Redis.

### Google mengembalikan `redirect_uri_mismatch`

- Bandingkan `GOOGLE_REDIRECT_URL` dengan Authorized redirect URI di Google Cloud Console.
- Pastikan scheme, domain, port, path, dan trailing slash identik.
- Restart aplikasi setelah mengubah environment.

### Swagger kosong atau masih menampilkan kontrak lama

```bash
swag init -g api_annotations.go -d docs/spec -o docs
```

Setelah generate, restart API/Air dan refresh browser tanpa cache.

### Tabel `users` tidak ditemukan

```bash
make migrate-up
```

Pastikan `DB_PORT` di `.env` menunjuk port PostgreSQL yang diekspos ke host.

## Catatan keamanan dan batasan saat ini

- Jangan commit credential asli di `.env`, Docker Compose, atau Kubernetes manifest. Gunakan secret manager dan segera rotasi credential yang pernah terlanjur masuk source control.
- Google OAuth callback saat ini belum memvalidasi nilai `state` terhadap session/cookie. Tambahkan validasi state sebelum penggunaan production untuk perlindungan CSRF.
- Handler Google login saat ini mencetak konfigurasi OAuth ke log, termasuk client secret. Jangan gunakan perilaku ini di production dan jangan membagikan log tersebut.
- Middleware JWT memvalidasi signature dan expiration, tetapi belum mengecek keberadaan session di Redis. Karena itu, revocation Redis belum otomatis memblokir JWT pada endpoint protected lain sampai middleware melakukan validasi session.
- Koneksi PostgreSQL saat ini menggunakan `sslmode=disable`; aktifkan TLS untuk production.
- Secret pada manifest Kubernetes memakai `stringData` dan ditujukan sebagai contoh development.
- Deployment Kubernetes memerlukan PostgreSQL eksternal atau manifest PostgreSQL terpisah; repository saat ini tidak menyediakannya.
- Gunakan HTTPS untuk callback OAuth dan seluruh traffic production.
