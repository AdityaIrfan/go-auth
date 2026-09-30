# KDA Auth and Calendar Service

REST API autentikasi berbasis Go dan Echo dengan dukungan login email/password, Google OAuth2, JWT access/refresh token, kalender event per user, PostgreSQL, Redis, health/readiness check, Swagger UI, Docker, dan Kubernetes.

Navigasi: [Arsitektur dan flow calendar](#arsitektur) · [Daftar endpoint](#endpoint-api) · [Calendar API — Events](#calendar-api--events) · [Swagger](#dokumentasi-swagger)

## Fitur

- Registrasi user menggunakan email dan password
- Password hashing dengan bcrypt
- Login email/password dan penerbitan JWT HS256
- Login Google menggunakan OAuth2 authorization-code flow
- Auto-register user saat pertama kali login melalui Google
- Penyimpanan session token dan revocation logout di Redis
- Refresh access/refresh token melalui endpoint publik
- CRUD event kalender milik user, dengan filter rentang waktu
- PostgreSQL sebagai penyimpanan data user dan event
- Liveness dan readiness endpoint
- Swagger UI dengan contoh request/response sesuai implementasi
- Unit test untuk handler, service, repository, middleware, OAuth2, JWT, response, dan validation

## Teknologi

| Komponen | Teknologi |
|---|---|
| Bahasa | Go 1.27.0 sesuai `go.mod` |
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
    Client --> Routes[Echo routes]
    Routes --> AuthHandler[AuthHandler]
    AuthHandler --> AuthService[AuthService port / authService]
    AuthService --> UserRepo[UserRepository / pgUserRepo]
    AuthService --> TokenRepo[TokenCacheRepository / Redis]
    UserRepo --> PG[(PostgreSQL: users dan events)]
    Routes --> JWT[JWT middleware: signature dan expiry]
    JWT --> EventHandler[EventHandler: user_id dari claims]
    EventHandler --> EventService[EventService port / eventService]
    EventService --> EventRepo[EventRepository / pgEventRepository]
    EventRepo --> PG
    AuthHandler --> GoogleOAuth[Google authorization dan token endpoint]
    AuthService --> GoogleInfo[Google tokeninfo endpoint]
```

Calendar menyimpan event di PostgreSQL dan membatasi akses berdasarkan `user_id` dari JWT. Google OAuth digunakan untuk login; implementasi events saat ini tidak menyinkronkan event ke Google Calendar. Redis menyimpan session auth, sedangkan middleware route events saat ini hanya memeriksa signature dan expiry JWT.

### Flow request calendar

Semua request calendar melewati pemeriksaan JWT dan handler sebelum diteruskan ke service. Diagram pertama menunjukkan alur umum; diagram kedua merinci aturan tiap operasi.

```mermaid
flowchart TD
    Request["Request Calendar API"] --> JWT{"JWT valid?"}
    JWT -->|Tidak| Unauthorized["401: invalid or expired token"]
    JWT -->|Ya| Handler["Handler membaca user_id dari JWT"]
    Handler --> Input["Baca input sesuai operasi"]
    Input --> Valid{"Input valid?"}
    Valid -->|Tidak| BadRequest["400: invalid request body atau invalid event id"]
    Valid -->|Ya| Service["Service menjalankan operasi calendar"]
    Service --> Result["Handler mengirim JSON dan HTTP status dari service"]
```

| Operasi | Pemeriksaan input di handler |
|---|---|
| Create | Bind JSON dan validasi title serta kedua timestamp |
| List | Bind filter opsional dari JSON body; tidak menjalankan validasi required |
| Update | Parse UUID event, bind JSON, lalu validasi title serta kedua timestamp |
| Delete | Parse UUID event |

```mermaid
flowchart TD
    Operation{"Operasi service"}
    Operation -->|Create| CreateTime{"End sesudah start?"}
    CreateTime -->|Tidak| TimeError["400: end_time must be after start_time"]
    CreateTime -->|Ya| Insert["Repository: simpan event baru"]

    Operation -->|List| List["Repository: cari event milik user sesuai filter"]

    Operation -->|Update atau Delete| Lookup["Repository: cari event berdasarkan id dan user_id"]
    Lookup --> Found{"Lookup berhasil?"}
    Found -->|Tidak| NotFound["404: event not found"]
    Found -->|Ya| Mutation{"Operasi"}
    Mutation -->|Update| UpdateTime{"End sama atau sesudah start?"}
    UpdateTime -->|Tidak| TimeError
    UpdateTime -->|Ya| Save["Repository: simpan perubahan event"]
    Mutation -->|Delete| Delete["Repository: hapus event milik user"]

    Insert --> Result{"Operasi repository berhasil?"}
    List --> Result
    Save --> Result
    Delete --> Result
    Result -->|Tidak| Failure["422: pesan kegagalan sesuai operasi"]
    Result -->|Ya| Success["Create: 201; List, Update, Delete: 200"]
```

Setiap jalur error langsung mengakhiri request. Create mensyaratkan `end_time > start_time`, sedangkan update menerima `end_time >= start_time`. Detail response tersedia di [Calendar API](#calendar-api--events).

Struktur direktori:

```text
cmd/api/                         application bootstrap
docs/                            Swagger source dan generated contract
internal/core/domain/            entity dan request/response model
internal/core/ports/             inbound/outbound interfaces
internal/core/services/          authentication dan calendar business rules
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

- Go 1.26.0+ (Dockerfile saat ini memakai image Go 1.27)
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
| `JWT_EXPIRATION_HOURS` | Tidak | Masa berlaku JWT; service menggunakan `24` jam jika kosong, tidak valid, atau nol; nilai negatif diteruskan apa adanya |
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
{"status_code":200,"status":"success","message":"I'm healthy"}
```

Response ready:

```json
{"status_code":200,"status":"success","message":"I'm ready"}
```

Jika PostgreSQL atau Redis tidak dapat dijangkau, `/ready` mengembalikan `503`, misalnya:

```json
{
  "status_code": 503,
  "status": "failed",
  "message": "unavailable"
}
```

## Database migration

Migration pertama membuat tabel `users`, constraint unik email/Google ID, dan index email. Migration kedua membuat tabel `events`, foreign key `user_id` dengan `ON DELETE CASCADE`, dan index `(user_id, start_time, end_time)`. Jalankan seluruh migration sebelum memakai endpoint event.

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

**Gunakan perintah di atas untuk regenerasi.** Target `make swagger` saat ini masih membaca anotasi runtime lama dan tidak digunakan untuk kontrak ini. Makefile dan anotasi runtime tidak diubah.

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
| `POST` | `/api/v1/auth/register` | Tidak | `201`, `400`, `422` | Registrasi email/password |
| `POST` | `/api/v1/auth/login` | Tidak | `200`, `400`, `401`, `422` | Login email/password |
| `GET` | `/api/v1/auth/google/login` | Tidak | `307` | Redirect browser ke Google |
| `GET` | `/api/v1/auth/google/callback` | Tidak | `200`, `401`, `422`, `500` | Exchange authorization code dan menerbitkan JWT aplikasi |
| `POST` | `/api/v1/auth/logout` | Bearer JWT | `200`, `401`, `422` | Menghapus session token dari Redis |
| `POST` | `/api/v1/auth/refresh` | Token di JSON body | `200`, `400`, `422`, `500` | Menerbitkan pasangan token baru |
| `POST` | `/api/v1/events` | Bearer JWT | `201`, `400`, `401`, `422` | Membuat event |
| `GET` | `/api/v1/events` | Bearer JWT | `200`, `400`, `401`, `422` | Daftar event milik user |
| `PUT` | `/api/v1/events/{id}` | Bearer JWT | `200`, `400`, `401`, `404`, `422` | Mengganti field event |
| `DELETE` | `/api/v1/events/{id}` | Bearer JWT | `200`, `400`, `401`, `404`, `422` | Menghapus event |

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
  "message": "register success, do login"
}
```

Input tidak valid menghasilkan `400`. Email duplikat juga menghasilkan `400` dengan pesan `email already registered`. Kegagalan lookup, hashing password, atau penyimpanan user menghasilkan `422`:

```json
{
  "status_code": 422,
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
  "message": "login success",
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "<refresh-jwt>",
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
  "message": "invalid credentials"
}
```

Kegagalan penerbitan token atau penyimpanan session menghasilkan `422` dengan pesan `login failed`. Masa berlaku refresh token adalah masa berlaku access token ditambah satu jam.

### Validation error

```json
{
  "status_code": 400,
  "status": "failed",
  "message": "invalid request body"
}
```

Malformed JSON maupun kegagalan validasi menghasilkan envelope di atas. Helper `ErrorResponse` saat ini mengabaikan parameter detail error, sehingga field `errors` tidak dikirim. Hal ini juga membuat `/ready` tidak menampilkan status masing-masing dependency.

### Google OAuth2 login

1. Buat OAuth Client dengan tipe **Web application** di Google Cloud Console.
2. Daftarkan nilai `GOOGLE_REDIRECT_URL` sebagai **Authorized redirect URI**. Scheme, host, port, path, dan trailing slash harus sama persis.
3. Isi `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, dan `GOOGLE_REDIRECT_URL`.
4. Restart API.
5. Buka endpoint berikut langsung melalui browser:

```text
http://localhost:8080/api/v1/auth/google/login
```

Exchange code gagal atau respons Google tanpa `id_token` menghasilkan `401` dengan pesan `login failed`. Kegagalan pada service Google SSO menghasilkan `422` dengan pesan yang sama.

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
  "message": "logout success"
}
```

JWT hilang, invalid, atau expired ditolak middleware:

```json
{"status_code":401,"status":"failed","message":"invalid or expired token"}
```

Kegagalan revocation menghasilkan:

```json
{
  "status_code": 422,
  "status": "failed",
  "message": "logout failed"
}
```

### Refresh token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<refresh-jwt>"}'
```

Response `200` memakai data token yang sama dengan login, dengan pesan `refresh token success`. Body tidak valid menghasilkan `400`; token invalid/expired atau kegagalan database/penerbitan token/Redis menghasilkan `422` dengan pesan `refresh token failed`.

## Calendar API — Events

Kontrak bagian ini mengikuti tag **Events** di [Swagger UI](http://localhost:8080/swagger/index.html) dan [`docs/swagger.yaml`](docs/swagger.yaml).

Seluruh route event membutuhkan `Authorization: Bearer <access_token>`. Pemilik ditentukan dari JWT. `title`, `start_time`, dan `end_time` wajib untuk create/update; `description` opsional. Timestamp memakai RFC3339.

| Field body | Tipe | Create/Update | Keterangan |
|---|---|---|---|
| `title` | string | Wajib | Tidak boleh kosong |
| `description` | string | Opsional | Jika tidak dikirim, bernilai string kosong |
| `start_time` | string, date-time | Wajib | RFC3339 dengan zona waktu, misalnya `2026-10-01T09:00:00+07:00` |
| `end_time` | string, date-time | Wajib | Create harus sesudah start; update menerima waktu yang sama |

`id`, `user_id`, `created_at`, dan `updated_at` dihasilkan backend. `user_id` dalam body tidak mengubah pemilik. Untuk update/delete, `{id}` adalah UUID event dari response create/list. Belum ada endpoint detail `GET /events/{id}` atau pagination.

### Create event — `POST /api/v1/events`

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H 'Authorization: Bearer <access_token>' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Team meeting","description":"Weekly planning","start_time":"2026-10-01T09:00:00+07:00","end_time":"2026-10-01T10:00:00+07:00"}'
```

Response `201`:

```json
{
  "status_code": 201,
  "status": "success",
  "message": "create event success",
  "data": {
    "id": "c27096d8-58d1-4014-9830-96f36ab04c9f",
    "user_id": "c8721202-510d-4a9a-b1d5-30b78c01d73b",
    "title": "Team meeting",
    "description": "Weekly planning",
    "start_time": "2026-10-01T09:00:00+07:00",
    "end_time": "2026-10-01T10:00:00+07:00",
    "created_at": "2026-09-30T08:00:00Z",
    "updated_at": "2026-09-30T08:00:00Z"
  }
}
```

### List events — `GET /api/v1/events`

List semua event milik user:

```bash
curl http://localhost:8080/api/v1/events -H 'Authorization: Bearer <access_token>'
```

Filter saat ini dibaca dari **JSON body pada GET**, bukan query parameter, karena model request tidak memiliki tag `query`. Browser/Swagger UI dapat menolak GET dengan body; gunakan curl untuk filter:

```bash
curl -X GET http://localhost:8080/api/v1/events \
  -H 'Authorization: Bearer <access_token>' \
  -H 'Content-Type: application/json' \
  -d '{"start_time":"2026-10-01T00:00:00+07:00","end_time":"2026-10-02T00:00:00+07:00"}'
```

Kedua batas opsional. Filter memilih event dengan `start_time >= batas awal` dan `end_time <= batas akhir`, bukan semua event yang overlap. Hasil diurutkan `start_time ASC`; response `200` berpesan `list events success`, dengan `data` array (kosong jika tidak ditemukan). Handler list tidak menjalankan validasi required maupun urutan waktu.

Response `200` dengan hasil:

```json
{
  "status_code": 200,
  "status": "success",
  "message": "list events success",
  "data": [
    {
      "id": "c27096d8-58d1-4014-9830-96f36ab04c9f",
      "user_id": "c8721202-510d-4a9a-b1d5-30b78c01d73b",
      "title": "Team meeting",
      "description": "Weekly planning",
      "start_time": "2026-10-01T09:00:00+07:00",
      "end_time": "2026-10-01T10:00:00+07:00",
      "created_at": "2026-09-30T08:00:00Z",
      "updated_at": "2026-09-30T08:00:00Z"
    }
  ]
}
```

Jika belum ada event yang cocok:

```json
{
  "status_code": 200,
  "status": "success",
  "message": "list events success",
  "data": []
}
```

### Update event — `PUT /api/v1/events/{id}`

Update mengganti `title`, `description`, `start_time`, dan `end_time`. `title` dan kedua timestamp wajib dikirim lagi. Description yang dihilangkan menjadi string kosong; `id`, pemilik, dan `created_at` tetap, sementara `updated_at` diperbarui.

```bash
curl -X PUT http://localhost:8080/api/v1/events/c27096d8-58d1-4014-9830-96f36ab04c9f \
  -H 'Authorization: Bearer <access_token>' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Rescheduled meeting","description":"Updated agenda","start_time":"2026-10-01T11:00:00+07:00","end_time":"2026-10-01T12:00:00+07:00"}'
```

Response `200`:

```json
{
  "status_code": 200,
  "status": "success",
  "message": "update event success",
  "data": {
    "id": "c27096d8-58d1-4014-9830-96f36ab04c9f",
    "user_id": "c8721202-510d-4a9a-b1d5-30b78c01d73b",
    "title": "Rescheduled meeting",
    "description": "Updated agenda",
    "start_time": "2026-10-01T11:00:00+07:00",
    "end_time": "2026-10-01T12:00:00+07:00",
    "created_at": "2026-09-30T08:00:00Z",
    "updated_at": "2026-09-30T09:00:00Z"
  }
}
```

### Delete event — `DELETE /api/v1/events/{id}`

```bash
curl -X DELETE http://localhost:8080/api/v1/events/c27096d8-58d1-4014-9830-96f36ab04c9f \
  -H 'Authorization: Bearer <access_token>'
```

Response `200` tanpa field `data`:

```json
{
  "status_code": 200,
  "status": "success",
  "message": "delete event success"
}
```

### Status dan error calendar

| Operasi | Status sukses | Status gagal yang didokumentasikan di Swagger |
|---|---|---|
| Create | `201` | `400`, `401`, `422` |
| List | `200` | `400`, `401`, `422` |
| Update | `200` | `400`, `401`, `404`, `422` |
| Delete | `200` | `400`, `401`, `404`, `422` |

| HTTP | Message | Kondisi |
|---|---|---|
| `400` | `invalid request body` | Binding JSON/timestamp gagal; required create/update tidak terpenuhi |
| `400` | `invalid event id` | ID path update/delete bukan UUID valid |
| `400` | `end_time must be after start_time` | Create: end <= start; update: end < start |
| `401` | `invalid or expired token` | JWT hilang, invalid, atau expired pada semua route events |
| `404` | `event not found` | Update/delete: event tidak ada, milik user lain, atau lookup repository gagal |
| `422` | `create event failed` | Insert gagal |
| `422` | `list events failed` | Query daftar gagal |
| `422` | `update event failed` | Penyimpanan perubahan gagal |
| `422` | `delete event failed` | Penghapusan gagal |

Contoh response event tidak ditemukan:

```json
{
  "status_code": 404,
  "status": "failed",
  "message": "event not found"
}
```

Error memakai envelope `status_code`, `status`, dan `message`. Helper response saat ini tidak menyertakan detail `errors`. Pada update, lookup kepemilikan dilakukan sebelum pengecekan urutan waktu; lookup yang gagal menghasilkan `404` lebih dulu.

### Mencoba calendar di Swagger

1. Login, lalu salin `data.access_token`.
2. Buka Swagger UI, klik **Authorize**, dan masukkan `Bearer <access_token>`.
3. Buka tag **Events** dan jalankan create; simpan `data.id`.
4. Jalankan list tanpa body untuk seluruh event milik user. Gunakan contoh curl di atas untuk filter JSON body pada GET karena browser dapat menolak body GET.
5. Gunakan ID event yang sama untuk update dan delete.

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

- request binding dan validation failure sesuai envelope aktual
- register/login/logout/refresh sukses dan gagal
- CRUD events, batas waktu, binding filter, dan kepemilikan user
- query repository events, rentang waktu, empty result, serta error database
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

### Perilaku runtime yang dicatat oleh test

Dokumentasi dan test mengikuti implementasi sekarang; perubahan ini tidak memperbaiki kode utama.

- `ErrorResponse` mengabaikan detail error, termasuk validasi dan status dependency readiness.
- Refresh untuk user yang tidak ditemukan dapat panic karena dereference user nil; middleware Recover pada aplikasi mengubah panic menjadi HTTP `500`.
- Google tokeninfo dengan JSON valid tetapi `email` atau `sub` kosong dapat panic karena pemanggilan `err.Error()` saat err nil; aplikasi merespons `500` melalui Recover.
- Auto-register Google menyimpan ID token yang diterima ke `google_id`, bukan klaim `sub`.
- Verifier refresh belum membedakan jenis token: access token yang valid juga diterima. Refresh token tidak disimpan/direvoke di Redis, dan refresh tidak mengecek session Redis.
- Nilai negatif `JWT_EXPIRATION_HOURS` diteruskan tanpa normalisasi dan dapat menghasilkan token expired.

Test karakterisasi panic dan perilaku di atas perlu disesuaikan jika kode utama diperbaiki pada perubahan terpisah.
