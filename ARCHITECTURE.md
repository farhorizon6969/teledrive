# TeleDrive Architecture

TeleDrive is a single-binary cloud storage system that bridges a web interface and virtual file system to Telegram's distributed cloud infrastructure via MTProto.

---

## 1. System Overview

```
+---------------------------------------------------------------------------------+
|                                TeleDrive Host                                   |
|                                                                                 |
|  +--------------------------+   +-------------------------+   +-------------------+  |
|  |     Web UI (Browser)     |   |   Embedded Dashboard    |   |     CLI Tool      |  |
|  | (HTML5/Vanilla/CSS/SVG)  |<->| (net/http + templates)  |<->| (teledrive login) |  |
|  +--------------------------+   +-------------------------+   +-------------------+  |
|                                         |                                       |
|                                         v                                       |
|                        +---------------------------------+                      |
|                        |       Drive Core Engine         |                      |
|                        | - Virtual Folder Tree           |                      |
|                        | - Resumable Upload Coordinator  |                      |
|                        | - Range Request Streamer        |                      |
|                        | - Share Link Authorizer         |                      |
|                        +---------------------------------+                      |
|                               |                   |                             |
|                               v                   v                             |
|                  +----------------------+  +---------------------+              |
|                  | SQLite Metadata (WAL)|  | Telegram MTProto    |              |
|                  | - modernc.org/sqlite |  | - gotd/td client    |              |
|                  | - AES-GCM Sessions   |  | - 512KB Part Worker |              |
|                  +----------------------+  +---------------------+              |
+-------------------------------------------------------|-------------------------+
                                                        | MTProto TCP/TLS
                                                        v
                                          +---------------------------+
                                          | Telegram Cloud Platform   |
                                          | (Storage Channel Vault)   |
                                          +---------------------------+
```

---

## 2. Core Workflows

### 2.1 Resumable Upload Flow

The client browser slices large files into 5 MB–10 MB **Chunks**. TeleDrive receives each chunk, slices it into 512 KB MTProto **Parts**, streams them directly to Telegram via `gotd/td`, and tracks the upload session in SQLite.

```mermaid
sequenceDiagram
    autonumber
    actor User as Web Client
    participant API as TeleDrive HTTP (net/http)
    participant Core as Drive Engine
    participant DB as SQLite (modernc)
    participant TG as Telegram MTProto (gotd/td)

    User->>API: POST /api/upload/init (name, size, folder_id, mime)
    API->>DB: Create upload_session (total_parts, status=pending)
    API-->>User: Return session_id & chunk_size (5MB)

    loop Every 5MB Chunk
        User->>API: POST /api/upload/chunk (session_id, chunk_index, bytes)
        API->>Core: Split 5MB into 10x 512KB Parts
        loop Every 512KB Part
            Core->>TG: upload.saveBigFilePart (file_id, part_index, bytes)
        end
        API->>DB: Update uploaded_parts count
        API-->>User: 200 OK (chunk confirmed)
    end

    User->>API: POST /api/upload/complete (session_id)
    API->>TG: messages.sendMedia (InputMediaUploadedDocument)
    TG-->>API: Message confirmation (telegram_message_id, file_id)
    API->>DB: Insert files record & Delete upload_session
    API-->>User: 201 Created (Virtual File ready)
```

---

### 2.2 Download & Media Streaming Flow (HTTP 206 Range)

To preview videos or resume partial downloads, TeleDrive translates HTTP byte-range requests directly into MTProto part offsets without buffering the entire file.

```mermaid
sequenceDiagram
    autonumber
    actor Client as Browser / Media Player
    participant API as TeleDrive HTTP
    participant DB as SQLite
    participant TG as Telegram MTProto

    Client->>API: GET /api/files/{id}/stream (Range: bytes=1048576-2097151)
    API->>DB: Query file metadata (telegram_file_id, size, mime_type)
    API->>API: Calculate MTProto Part offsets (part 2 to 3, 512KB each)
    API->>TG: upload.getFile (offset=1048576, limit=1048576)
    TG-->>API: Stream raw 512KB parts
    API-->>Client: 206 Partial Content (Content-Range, stream bytes)
```

---

### 2.3 Share Link Flow

Public visitors access files via cryptographic tokens. Passwords and expiry are validated at the edge before streaming bytes.

```mermaid
sequenceDiagram
    autonumber
    actor Guest as Guest User
    participant Web as TeleDrive (/s/{token})
    participant DB as SQLite
    participant TG as Telegram MTProto

    Guest->>Web: GET /s/{token}
    Web->>DB: Lookup share_token (check expires_at, is_password_protected)
    alt Password Protected
        Web-->>Guest: Render Password Prompt Form
        Guest->>Web: POST /s/{token}/unlock (password)
        Web->>DB: Verify bcrypt password hash
    end
    Web-->>Guest: Render Landing Page (metadata + stream preview + download button)
    
    opt Direct Download or Stream
        Guest->>Web: GET /s/{token}/download
        Web->>DB: Increment download_count
        Web->>TG: Stream file parts
        TG-->>Guest: Pass-through file bytes
    end
```

---

### 2.4 Disaster Recovery & Backup Flow

The SQLite database is backed up to the dedicated Storage Channel periodically and on shutdown.

```mermaid
sequenceDiagram
    autonumber
    participant Scheduler as Backup Scheduler (Cron/Shutdown)
    participant DB as SQLite (WAL)
    participant TG as Telegram Storage Channel

    Scheduler->>DB: PRAGMA wal_checkpoint(TRUNCATE)
    Scheduler->>Scheduler: Vacuum into temporary snapshot + gzip
    Scheduler->>TG: Send document (teledrive-backup-{timestamp}.db.gz)
    Scheduler->>TG: Pin message (pin_id)
    Scheduler->>DB: Update last_backup_timestamp
```

---

## 3. Database Schema (SQLite)

```sql
-- Virtual Folder hierarchy
CREATE TABLE folders (
    id TEXT PRIMARY KEY,
    parent_id TEXT NULL REFERENCES folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_folders_parent ON folders(parent_id);

-- Virtual Files mapped to Telegram objects
CREATE TABLE files (
    id TEXT PRIMARY KEY,
    folder_id TEXT NULL REFERENCES folders(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    telegram_message_id INTEGER NOT NULL,
    telegram_file_id TEXT NOT NULL,
    telegram_access_hash TEXT NOT NULL,
    sha256 TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_files_folder ON files(folder_id);
CREATE INDEX idx_files_name ON files(name);

-- In-flight resumable upload sessions
CREATE TABLE upload_sessions (
    id TEXT PRIMARY KEY,
    folder_id TEXT REFERENCES folders(id),
    name TEXT NOT NULL,
    size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    total_parts INTEGER NOT NULL,
    uploaded_parts INTEGER DEFAULT 0,
    telegram_file_id INTEGER NOT NULL, -- gotd random ID
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Public share links
CREATE TABLE share_links (
    id TEXT PRIMARY KEY,
    token TEXT UNIQUE NOT NULL,
    file_id TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    password_hash TEXT NULL,
    expires_at DATETIME NULL,
    download_count INTEGER DEFAULT 0,
    max_downloads INTEGER NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_share_token ON share_links(token);

-- Encrypted Telegram MTProto session & system settings
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. Source Directory Structure

Following the Ponytail principle (minimal files, zero unnecessary abstractions):

```
teledrive/
├── CONTEXT.md
├── ARCHITECTURE.md
├── SECURITY.md
├── MILESTONES.md
├── docs/adr/
│   ├── 0001-storage-channel-target.md
│   ├── 0002-pure-go-mtproto-client.md
│   ├── 0003-pass-through-streaming.md
│   ├── 0004-modernc-sqlite-pure-go.md
│   ├── 0005-stdlib-net-http.md
│   ├── 0006-chunked-resumable-http-upload.md
│   ├── 0007-aes-gcm-session-encryption.md
│   ├── 0008-primary-account-tos-safe-mode.md
│   └── 0009-zero-build-embedded-modern-ui.md
├── cmd/
│   └── teledrive/
│       └── main.go              # CLI router (server, login, upload, list, backup)
├── internal/
│   ├── app/
│   │   └── config.go            # Minimal configuration loader
│   ├── crypto/
│   │   └── aes.go               # AES-256-GCM encryption helpers
│   ├── db/
│   │   ├── db.go                # SQLite init (WAL mode)
│   │   ├── files.go             # Virtual folder & file CRUD
│   │   └── backup.go            # Snapshot export & restore
│   ├── telegram/
│   │   ├── client.go            # gotd/td connection manager
│   │   ├── auth.go              # CLI interactive login wizard
│   │   ├── uploader.go          # 512KB MTProto part uploader
│   │   ├── downloader.go        # Range-aware MTProto part streamer
│   │   └── limiter.go           # FloodWait backoff & rate queue
│   └── web/
│       ├── server.go            # net/http ServeMux routes & middleware
│       ├── handlers_drive.go    # File/folder operations & uploads
│       ├── handlers_share.go    # Public /s/{token} & /api/shares management
│       └── static/              # Embedded UI assets (CSS design tokens, Lucide SVG, Vanilla JS)
├── go.mod
└── go.sum
```
