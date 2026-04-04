# AM-DL

Workspace downloader Apple Music yang dibuat lebih ramah pemula untuk Windows.

Bahasa:

- English: `README.md`
- Bahasa Indonesia: `README-ID.md`

## Isi repository ini

Repo ini disusun supaya pengguna pemula lebih cepat bisa jalan.

File/folder penting:

- `amdl.exe` — binary Windows yang sudah dibuild untuk quick start
- `config.yaml` — config awal dengan nilai placeholder yang aman
- `setup.ps1` — helper setup/build/check
- `wrapper-login.ps1` — helper login Apple Music
- `wrapper-start.ps1` — helper start/stop/status backend wrapper
- `start.bat` — start wrapper lalu buka `amdl`
- `download.bat` — helper cepat untuk command download
- `client/` — source code Go untuk aplikasi
- `wrapper-docker/` — bundle runtime Docker untuk backend
- `wrapper-src/` — bundle source/reference yang tetap disimpan karena masih bisa berguna untuk flow advanced/manual
- `tools/` — folder lokal opsional untuk binary seperti `mp4decrypt.exe`

Backend yang didukung:

- `WorldObservationLog/wrapper`

## Sebelum mulai

Kamu tetap butuh:

- langganan Apple Music aktif
- Docker Desktop terpasang dan sedang jalan
- MP4Box / GPAC terpasang

Opsional tapi disarankan:

- `ffmpeg`
- `mp4decrypt.exe` (dibutuhkan untuk Music Video)

`amdl doctor` akan kasih tahu bagian mana yang masih kurang.

---

## Jalur tercepat untuk pemula

Kalau kamu mau jalur paling singkat:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\setup.ps1
.\amdl.exe login
.\wrapper-start.ps1
.\amdl.exe doctor
.\amdl.exe
```

Kalau kamu juga mau fitur MV / AAC-LC:

```powershell
.\amdl.exe token set
```

lalu taruh `mp4decrypt.exe` di salah satu lokasi ini:

- di samping `amdl.exe`
- `tools\mp4decrypt.exe`
- atau install ke `PATH`

---

## Tutorial install detail sampai bisa dipakai

## Langkah 1 — Buka PowerShell di folder repo ini

Pastikan kamu sedang ada di folder repo ini.

Kamu harus bisa melihat file seperti:

- `amdl.exe`
- `setup.ps1`
- `wrapper-login.ps1`
- `wrapper-start.ps1`

Cek dengan:

```powershell
dir
```

## Langkah 2 — Izinkan script PowerShell lokal untuk sesi ini

Jalankan:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
```

Ini hanya berlaku untuk window PowerShell yang sedang dipakai.

## Langkah 3 — Jalankan setup

Jalankan:

```powershell
.\setup.ps1
```

Yang dilakukan command ini:

- build `amdl.exe` kalau perlu
- menyiapkan image Docker wrapper
- menyiapkan flow config lokal

Kalau berhasil, biasanya akan muncul:

- `Setup complete`
- petunjuk next step yang mengarah ke `amdl.exe login`

## Langkah 4 — Login dengan akun Apple Music milikmu sendiri

Jalankan:

```powershell
.\amdl.exe login
```

Yang dilakukan command ini:

- menjalankan flow login wrapper
- memakai akunmu satu kali untuk bikin session/cache lokal
- **tidak** menyimpan password ke `config.yaml`

Kalau berhasil, session lokal akan tersimpan di:

```text
wrapper-docker/rootfs/data/
```

Kalau Apple minta 2FA, selesaikan di flow terminal tersebut.

## Langkah 5 — Nyalakan backend wrapper

Jalankan:

```powershell
.\wrapper-start.ps1
```

Kalau normal, port ini akan aktif di lokal:

- `127.0.0.1:10020`
- `127.0.0.1:20020`
- `127.0.0.1:30020`

Untuk cek status:

```powershell
.\wrapper-start.ps1 -Status
```

## Langkah 6 — Jalankan doctor check

Jalankan:

```powershell
.\amdl.exe doctor
```

Minimal yang kamu mau lihat:

- backend port reachable
- login session cached locally
- MP4Box available

Warning yang mungkin muncul:

- `media-user-token` missing → fitur MV / AAC-LC belum siap
- `mp4decrypt` missing → fitur Music Video belum siap

## Langkah 7 — Mulai pakai aplikasi

Jalankan:

```powershell
.\amdl.exe
```

Ini akan membuka menu interaktif.

Pilihan yang paling penting untuk pemula:

- Search & Download
- Download from URL
- Setup Wizard
- Login to Apple Music
- Doctor Check
- Backend Status

---

## Opsional: aktifkan fitur MV / AAC-LC

Fitur ini butuh setup tambahan.

## 1) Isi `media-user-token`

Jalankan:

```powershell
.\amdl.exe token set
```

Ini akan menyimpan token secara lokal ke `config.yaml`.

## 2) Tambahkan `mp4decrypt.exe`

Taruh binary asli di salah satu lokasi ini:

- `.\mp4decrypt.exe`
- `.\tools\mp4decrypt.exe`
- atau install global ke `PATH`

## 3) Verifikasi lagi

Jalankan:

```powershell
.\amdl.exe doctor
```

Kalau MV sudah siap penuh, seharusnya warning berikut hilang:

- `media-user-token`
- `mp4decrypt`
- `Music Video readiness`

---

## Command yang berguna

```powershell
.\setup.ps1
.\amdl.exe login
.\amdl.exe logout
.\amdl.exe token set
.\amdl.exe token clear
.\wrapper-start.ps1
.\wrapper-start.ps1 -Status
.\wrapper-start.ps1 -Logs
.\amdl.exe doctor
.\amdl.exe backend status
.\amdl.exe backend guide
.\amdl.exe config show
.\amdl.exe search song "Blinding Lights"
.\amdl.exe download https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
```

---

## Troubleshooting

## Masalah: `amdl.exe` atau script tidak dikenali

Di PowerShell harus pakai prefix yang benar:

```powershell
.\amdl.exe doctor
```

bukan:

```powershell
.amdl.exe doctor
```

## Masalah: backend port unreachable

Cek:

```powershell
.\wrapper-start.ps1 -Status
```

Kalau belum jalan, start lagi:

```powershell
.\wrapper-start.ps1
```

Pastikan juga Docker Desktop sedang hidup.

## Masalah: login session missing

Jalankan:

```powershell
.\amdl.exe login
```

## Masalah: Music Video belum ready

Biasanya kamu masih kurang salah satu dari ini:

- `media-user-token` valid
- `mp4decrypt.exe`

Jalankan:

```powershell
.\amdl.exe doctor
```

lalu lihat warning baris mana yang masih muncul.

---

## Catatan

- `config.yaml` di repo ini adalah starter file dengan placeholder aman
- session/cache lokal disimpan di `wrapper-docker/rootfs/data`
- data session itu adalah data runtime lokal dan tidak boleh ikut ter-commit dengan kredensial pribadi
- `wrapper-docker/` adalah jalur runtime utama untuk pemula
- `wrapper-src/` tetap disimpan karena kamu memang minta folder itu ikut dipublikasikan untuk kebutuhan advanced/reference

## Release automation

Repo ini juga punya:

- `.github/workflows/release.yml`

Workflow ini bisa build bundle release Windows portable berisi:

- `amdl.exe`
- helper scripts
- `config.example.yaml`
- quickstart docs
- `wrapper-docker/`
