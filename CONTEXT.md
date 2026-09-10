# TeleDrive

A self-hosted personal cloud storage bridge that uses Telegram MTProto infrastructure as a high-capacity object store with a virtual file system.

## Language

**Virtual Folder**:
A logical directory node in the database hierarchy that groups files virtually without any native equivalent in Telegram.
_Avoid_: Directory, chat folder, album

**Storage Channel**:
A dedicated private Telegram channel used exclusively by TeleDrive as an object store for file parts and database snapshots.
_Avoid_: Chat, saved messages, drive folder, vault channel

**Part**:
A 512 KB data segment transferred to or from Telegram servers via MTProto `upload.saveBigFilePart` or `upload.getFile`.
_Avoid_: Block, fragment, chunk (when referring to Telegram protocol layer)

**Chunk**:
A 5 MB to 10 MB HTTP payload segment transmitted between client and server during a resumable upload.
_Avoid_: Part, slice, packet

**Upload Session**:
A temporary database record tracking the progressive assembly and Telegram part offsets of an in-flight file upload.
_Avoid_: Upload job, pending file, transfer state

**Session String**:
The serialized, authenticated MTProto cryptographic state and authorization key required to access the Telegram API without re-authenticating.
_Avoid_: Auth token, login cookie, API credential

**Flood Wait**:
A rate-limiting error returned by Telegram API specifying the mandatory cooldown duration in seconds before the client may retry.
_Avoid_: Rate limit, throttling, ban

**Pass-Through Stream**:
A pipelined byte stream directly bridging the client HTTP connection and MTProto without spooling the complete file to server disk.
_Avoid_: Temporary file, buffer cache, spool

**Database Snapshot**:
A compressed point-in-time copy of the SQLite database uploaded to the Storage Channel to guarantee zero metadata loss if server storage is destroyed.
_Avoid_: Backup dump, state export, replica

**Snapshot History**:
A chronological log of all Database Snapshots residing in the Storage Channel.
_Avoid_: Backup logs, snapshot list, dump history

**Snapshot Retention Policy**:
The automated pruning rule that preserves a fixed count of recent Database Snapshots while deleting older snapshot messages from the Storage Channel.
_Avoid_: Auto-clean, backup expiry, cleanup rule

**Point-in-Time Restore**:
The recovery process of replacing the live SQLite database with a specific historical Database Snapshot selected from the Snapshot History.
_Avoid_: Database rollback, state rewind, undo

**Range Request**:
An HTTP 206 request translated on-the-fly into specific MTProto Part offsets to enable instant video seeking and media preview.
_Avoid_: Partial download, slice streaming

**Share Link**:
A time-bounded, optionally password-protected public URL granting guest access to stream or download a specific virtual file.
_Avoid_: Public link, invite link
