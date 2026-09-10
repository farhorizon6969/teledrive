<div align="center">

# ⚡ TeleDrive

**High-Capacity Cloud Storage Powered by Telegram MTProto Infrastructure**

*Turn your Telegram account into a secure, unlimited personal cloud drive with a Google Drive-like Web Dashboard and CLI.*

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Author](https://img.shields.io/badge/Author-Herliansyah-purple?style=flat&logo=github)](https://github.com/herliansyah)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![GitHub](https://img.shields.io/badge/GitHub-herliansyah%2Fteledrive-181717?style=flat&logo=github)](https://github.com/herliansyah/teledrive)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-blue)](https://github.com/herliansyah/teledrive)
[![Pure Go](https://img.shields.io/badge/CGO-Zero%20(Pure%20Go)-orange)](https://modernc.org/sqlite)

[English](#english) &bull; [Bahasa Indonesia](#bahasa-indonesia)

---

</div>

<a name="english"></a>
## 🌟 English Overview

**TeleDrive** is a lightweight, single-binary personal cloud storage bridge. It uses Telegram's distributed cloud infrastructure as an object store via native **MTProto** (`gotd/td`), providing up to **2 GB (Free accounts)** or **4 GB (Premium accounts)** per file transfer, without intermediate disk spooling on your server.

### ✨ Key Features

- **🎨 Modern Zero-Build Web Interface**:
  - Dark & Light mode toggle with persistent state in local storage.
  - Dual-view explorer: instant toggle between **Grid View** (visual file cards) and **Table/List View** with instant in-memory column sorting (Name, Size, Modified Date).
  - Floating Upload Manager Drawer at bottom-right with real-time chunk progress (`Chunk 3/8`), minimizable pill state, and sequential safe-mode processing.
  - Crisp embedded Lucide SVG vector icons with zero additional network requests.
  - In-app non-intrusive toast notifications and custom modal dialogs (replacing native browser alerts/prompts).
  - Shared Links Management view in dashboard to audit active links, copy URLs, review downloads, and revoke access.
  - Expanded media and code viewer: seekable video (HTTP 206), audio, image, PDF, and syntax/plain text viewer for code files (`.txt`, `.md`, `.json`, `.go`, `.py`, `.log`).
  - Built-in **Mobile QR Code Generator** on public share links for frictionless smartphone handoff.
- **📁 Virtual File System**: Full hierarchical folder management (create, rename, move with cycle prevention, delete, search) backed by SQLite with WAL mode.
- **⚡ Resumable Chunked Upload**: Browser slices files into 5 MB chunks and streams them directly into 512 KB MTProto parts with zero VPS disk wear.
- **🎬 Instant Media Streaming (HTTP 206 Range)**: Stream and seek large videos, audio, images, and PDFs in real-time without downloading the complete file first.
- **🔗 Secure Public Share Links**: Generate public links (`/s/:token`) with optional bcrypt password protection and automatic expiration timers.
- **🛡️ Primary Account Safe Mode (Strict Telegram ToS Compliance)**:
  - Realistic official desktop client telemetry fingerprinting (`PC 64bit`, `Linux/x86_64`, `AppVersion 5.0.0`).
  - Strict sequential single-worker queue (concurrency = 1) mimicking human desktop usage.
  - Adaptive 30ms pacing delay between 512 KB parts.
  - Automated defensive `FLOOD_WAIT_X` backoff without crashing or retrying aggressively.
  - Strictly private Storage Channel (`TeleDrive Vault`) with zero external members.
- **🔐 Military-Grade Security at Rest**: Telegram MTProto session strings (`auth_key`) are encrypted at rest in SQLite using **AES-256-GCM** with SHA-256 key derivation.
- **💾 Disaster Recovery & Auto-Healing**: Export compressed SQLite snapshots to your Telegram Storage Channel with pinned message recovery (`teledrive backup` and `teledrive restore`).

---

### 📥 Pre-built Binary Downloads

Download the latest pre-compiled binary for your operating system:

| Operating System | Architecture | Binary Archive |
| :--- | :--- | :--- |
| **Linux** | x86_64 (amd64) | [`teledrive-v1.0.0-linux-amd64.tar.gz`](dist/teledrive-v1.0.0-linux-amd64.tar.gz) |
| **Linux** | ARM64 (aarch64) | [`teledrive-v1.0.0-linux-arm64.tar.gz`](dist/teledrive-v1.0.0-linux-arm64.tar.gz) |
| **macOS** | Apple Silicon (arm64) | [`teledrive-v1.0.0-darwin-arm64.tar.gz`](dist/teledrive-v1.0.0-darwin-arm64.tar.gz) |
| **macOS** | Intel (amd64) | [`teledrive-v1.0.0-darwin-amd64.tar.gz`](dist/teledrive-v1.0.0-darwin-amd64.tar.gz) |
| **Windows** | x86_64 (amd64) | [`teledrive-v1.0.0-windows-amd64.tar.gz`](dist/teledrive-v1.0.0-windows-amd64.tar.gz) |

*Checksums are verified in [`sha256sums.txt`](dist/sha256sums.txt).*

---

### 🚀 Quick Start Guide

#### 1. Setup & Telegram Pairing (First Time Only)

Obtain your personal `API_ID` and `API_HASH` from [my.telegram.org](https://my.telegram.org) (Application Type: Desktop). Then run the interactive terminal wizard:

```bash
./teledrive login
```

The wizard will guide you through:
1. Entering your `API_ID` and `API_HASH`.
2. Entering your Telegram phone number.
3. Submitting the confirmation code sent to your Telegram app.
4. Submitting your 2FA Cloud Password (if enabled).
5. Auto-creating the private `TeleDrive Vault` storage channel.

#### 2. Start the Web Dashboard

```bash
./teledrive server
```

Open your browser and navigate to:
👉 **`http://localhost:8080`**

- **Default Admin Password**: `admin123` (Change via `TELEDRIVE_ADMIN_PASSWORD`).

---

### 💻 Command-Line Interface (CLI)

```bash
# Upload a file to TeleDrive
./teledrive upload ./sample_video.mp4

# Upload to a specific virtual folder
./teledrive upload ./report.pdf --folder <folder_id>

# Download a file back to your local machine
./teledrive download <file_id> --output ./restored_report.pdf

# List root folders and files
./teledrive list

# Create a full SQLite database snapshot and upload it to Telegram
./teledrive backup

# Restore SQLite database from the pinned snapshot in your Telegram channel
./teledrive restore
```

---

### ⚙️ Configuration & Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `TELEDRIVE_PORT` | `8080` | HTTP port for the web dashboard |
| `TELEDRIVE_DB_PATH` | `teledrive.db` | Path to the SQLite database file |
| `TELEDRIVE_ADMIN_PASSWORD` | `admin123` | Password for admin dashboard login |
| `TELEDRIVE_SECRET_KEY` | *(auto-generated)* | 32-byte secret key used for AES-256-GCM session encryption |
| `TELEDRIVE_TG_APP_ID` | *(prompted)* | Telegram App ID from `my.telegram.org` |
| `TELEDRIVE_TG_APP_HASH` | *(prompted)* | Telegram App Hash from `my.telegram.org` |

---

### 👤 Author & Credits

Developed and maintained by **Herliansyah**:
- **Creator Profile**: [@herliansyah](https://github.com/herliansyah)
- **Source Code**: [https://github.com/herliansyah/teledrive](https://github.com/herliansyah/teledrive)
- **License**: Released under the [MIT License](LICENSE). Open-source, free for personal and educational use.

<hr style="margin: 40px 0;">

<a name="bahasa-indonesia"></a>
## 🇮🇩 Panduan Bahasa Indonesia

**TeleDrive** adalah jembatan penyimpanan awan (*cloud storage*) berbasis Go (single binary) yang memanfaatkan infrastruktur Telegram sebagai backend penyimpanan melalui protokol resmi **MTProto** (`gotd/td`). Dengan TeleDrive, Anda dapat menikmati kapasitas upload hingga **2 GB (akun reguler)** atau **4 GB (akun Telegram Premium)** per file dengan tampilan web mirip Google Drive.

### 🎯 Fitur Unggulan
 
- **Single Binary Portabel (Zero CGO)**: Kompilasi 100% Pure Go tanpa dependensi compiler C/gcc (`modernc.org/sqlite` dan `gotd/td`).
- **Antarmuka Web Modern (Zero-Build)**:
  - Dukungan Dark & Light Mode dengan toggle instan dan penyimpanan preferensi di browser.
  - Mode tampilan ganda: **Grid View** (kartu file interaktif) dan **Table/List View** (tabel detail dengan sorting instan Nama, Ukuran, dan Tanggal).
  - Floating Upload Manager Drawer di pojok kanan bawah dengan pemantauan per-chunk progresif (`Chunk 3/8`) dan antrean sekuensial aman dari *Flood Wait*.
  - Ikon vektor modern Lucide SVG resolusi tinggi tanpa penambahan request HTTP.
  - Notifikasi toast dan dialog modal elegan (tanpa `alert`, `confirm`, atau `prompt` bawaan browser).
  - Tab Manajemen Shared Links di dashboard untuk memantau, menyalin URL, dan mencabut (*revoke*) link berbagi aktif.
  - Penampil pratinjau media dan dokumen teks/kode (`.txt`, `.md`, `.json`, `.go`, `.py`, dll).
  - Fitur **QR Code Generator** pada link publik untuk unduhan instan langsung dari smartphone.
- **Struktur Folder Virtual**: Mengatur hierarki folder, memindahkan file (*move* dengan proteksi siklus), rename, hapus permanen, dan pencarian cepat berbasis SQLite WAL.
- **Upload Chunked Resumable**: File dipotong menjadi chunk 5MB di browser dan dialirkan langsung ke part 512KB MTProto tanpa memenuhi disk server VPS.
- **Streaming Media Langsung (HTTP 206)**: Menonton video besar, memutar audio, atau membuka dokumen PDF langsung di browser tanpa perlu mengunduh seluruh file terlebih dahulu.
- **Link Berbagi Publik (Share Link)**: Buat link publik dengan proteksi password (bcrypt) dan masa berlaku (*expiration timer*).
- **Safe Mode Kepatuhan ToS (Aman untuk Akun Utama)**:
  - Menggunakan telemetri perangkat resmi (`PC 64bit`, `Linux/x86_64`, `AppVersion 5.0.0`).
  - Antrean upload strictly 1 koneksi aktif pada satu waktu (meniru aplikasi resmi Telegram Desktop).
  - Jeda adaptif 30ms antar-part untuk menjaga koneksi tetap dingin.
  - Penanganan jeda otomatis `FLOOD_WAIT` dari server Telegram.
  - Channel storage berstatus **Private** dan terisolasi (0 anggota luar).
- **Keamanan Data**: Kunci session string MTProto dienkripsi menggunakan algoritma **AES-256-GCM** sebelum disimpan di database.
- **Disaster Recovery**: Backup database metadata SQLite otomatis ke Telegram Storage Channel (`teledrive backup` & `teledrive restore`).

---

### 🚀 Cara Penggunaan Singkat

#### 1. Pasangkan Akun Telegram Anda (Login Pertama Kali)
Dapatkan `API_ID` dan `API_HASH` Anda dari [my.telegram.org](https://my.telegram.org), lalu jalankan:
```bash
./teledrive login
```
Ikuti petunjuk di layar untuk memasukkan nomor telepon, kode OTP dari Telegram, dan password 2FA Anda.

#### 2. Jalankan Server Web Dashboard
```bash
./teledrive server
```
Buka browser di **`http://localhost:8080`** (Password admin: `admin123`).

#### 3. Perintah Terminal (CLI)
```bash
# Mengunggah file
./teledrive upload ./laporan.pdf

# Menampilkan daftar file dan folder
./teledrive list

# Mengunduh kembali file ke komputer
./teledrive download <file_id> --output ./hasil.pdf

# Cadangkan database metadata ke Telegram
./teledrive backup

# Pulihkan database dari Telegram jika pindah server
./teledrive restore
```

---

### 📚 Dokumentasi Teknis Lanjutan

- [CONTEXT.md](CONTEXT.md) — Glosarium kanonik konsep domain TeleDrive.
- [ARCHITECTURE.md](ARCHITECTURE.md) — Diagram alur data upload, streaming HTTP 206, dan skema SQLite.
- [SECURITY.md](SECURITY.md) — Panduan keamanan, enkripsi session, dan mitigasi banned Telegram.
- [MILESTONES.md](MILESTONES.md) — Rincian tahapan pengembangan dan verifikasi.
- [docs/adr/](docs/adr/) — Arsip Catatan Keputusan Arsitektur (*Architectural Decision Records*).

---

### 👤 Pembuat & Lisensi

Diciptakan dan dikembangkan oleh **Herliansyah**:
- **Profil GitHub**: [@herliansyah](https://github.com/herliansyah)
- **Repositori Resmi**: [https://github.com/herliansyah/teledrive](https://github.com/herliansyah/teledrive)
- **Lisensi**: Didistribusikan di bawah [Lisensi MIT](LICENSE). Bebas digunakan, dipelajari, dan dimodifikasi untuk keperluan personal maupun edukasi.

