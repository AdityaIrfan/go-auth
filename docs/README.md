# Swagger documentation

`spec/api_annotations.go` adalah sumber kontrak Swagger dan hanya berisi tipe/contoh dokumentasi. File tersebut dipisahkan dari anotasi kode runtime agar dokumentasi dapat mengikuti response aktual tanpa mengubah handler atau service.

Generate ulang artefak dokumentasi dengan:

```bash
swag init -g api_annotations.go -d docs/spec -o docs
```

Output yang harus ikut diperbarui:

- `docs/docs.go`, dokumen yang disajikan oleh `/swagger/doc.json`
- `docs/swagger.json`
- `docs/swagger.yaml`

Konsistensi ketiga output dan seluruh route/status/example diverifikasi oleh `docs/docs_test.go`.

Jangan memakai `make swagger` untuk kontrak ini: target tersebut masih membaca anotasi runtime lama. Gunakan perintah eksplisit di atas. Dokumentasi mencakup auth, refresh, health/readiness, dan CRUD events; parameter, response envelope, serta batasan binding mengikuti kode saat ini.
