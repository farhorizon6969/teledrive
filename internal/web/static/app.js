let currentFolderId = null;
let currentFolderPath = [];
let uploadQueue = [];
let isUploading = false;

document.addEventListener("DOMContentLoaded", () => {
    loadDriveContent();
    setupDropzone();
});

async function loadDriveContent() {
    try {
        const folderParam = currentFolderId ? `?folder_id=${currentFolderId}` : "";
        const [foldersRes, filesRes] = await Promise.all([
            fetch(`/api/folders${folderParam}`),
            fetch(`/api/files${folderParam}`)
        ]);

        const folders = await foldersRes.json();
        const files = await filesRes.json();

        renderBreadcrumbs();
        renderFolders(folders || []);
        renderFiles(files || []);
    } catch (err) {
        console.error("Failed to load drive content:", err);
    }
}

function renderBreadcrumbs() {
    const el = document.getElementById("breadcrumbs");
    if (!el) return;

    let html = `<a onclick="navigateTo(null, 'Home')">Drive</a>`;
    currentFolderPath.forEach((crumb, idx) => {
        if (idx === currentFolderPath.length - 1) {
            html += ` / <span class="active">${escapeHtml(crumb.name)}</span>`;
        } else {
            html += ` / <a onclick="navigateTo('${crumb.id}', '${escapeHtml(crumb.name)}')">${escapeHtml(crumb.name)}</a>`;
        }
    });
    el.innerHTML = html;
}

function navigateTo(folderId, folderName) {
    if (folderId === null) {
        currentFolderId = null;
        currentFolderPath = [];
    } else {
        const idx = currentFolderPath.findIndex(f => f.id === folderId);
        if (idx >= 0) {
            currentFolderPath = currentFolderPath.slice(0, idx + 1);
        } else {
            currentFolderPath.push({ id: folderId, name: folderName });
        }
        currentFolderId = folderId;
    }
    loadDriveContent();
}

function renderFolders(folders) {
    const container = document.getElementById("folders-grid");
    if (!container) return;

    if (folders.length === 0) {
        container.innerHTML = `<div style="color: var(--text-secondary); font-size: 0.9rem;">No subfolders</div>`;
        return;
    }

    container.innerHTML = folders.map(f => `
        <div class="card" onclick="navigateTo('${f.id}', '${escapeHtml(f.name)}')">
            <div class="card-icon">📁</div>
            <div class="card-title">${escapeHtml(f.name)}</div>
            <div class="card-meta">Folder</div>
            <div class="card-actions" onclick="event.stopPropagation()">
                <button class="btn-icon" title="Rename" onclick="renameFolder('${f.id}', '${escapeHtml(f.name)}')">✏️</button>
                <button class="btn-icon" title="Delete" onclick="deleteFolder('${f.id}')">🗑️</button>
            </div>
        </div>
    `).join("");
}

function renderFiles(files) {
    const container = document.getElementById("files-grid");
    if (!container) return;

    if (files.length === 0) {
        container.innerHTML = `<div style="color: var(--text-secondary); font-size: 0.9rem;">No files</div>`;
        return;
    }

    container.innerHTML = files.map(f => `
        <div class="card" onclick="previewFile('${f.id}', '${escapeHtml(f.name)}', '${f.mime_type}', ${f.size})">
            <div class="card-icon">${getFileIcon(f.mime_type)}</div>
            <div class="card-title" title="${escapeHtml(f.name)}">${escapeHtml(f.name)}</div>
            <div class="card-meta">${formatSize(f.size)}</div>
            <div class="card-actions" onclick="event.stopPropagation()">
                <button class="btn-icon" title="Share" onclick="openShareModal('${f.id}', '${escapeHtml(f.name)}')">🔗</button>
                <button class="btn-icon" title="Download" onclick="downloadFile('${f.id}', '${escapeHtml(f.name)}')">⬇️</button>
                <button class="btn-icon" title="Rename" onclick="renameFile('${f.id}', '${escapeHtml(f.name)}')">✏️</button>
                <button class="btn-icon" title="Delete" onclick="deleteFile('${f.id}')">🗑️</button>
            </div>
        </div>
    `).join("");
}

function getFileIcon(mime) {
    if (mime.startsWith("video/")) return "🎬";
    if (mime.startsWith("audio/")) return "🎵";
    if (mime.startsWith("image/")) return "🖼️";
    if (mime === "application/pdf") return "📕";
    if (mime.includes("zip") || mime.includes("tar") || mime.includes("compressed")) return "📦";
    return "📄";
}

function formatSize(bytes) {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function escapeHtml(str) {
    return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

function setupDropzone() {
    const dropzone = document.getElementById("dropzone");
    const fileInput = document.getElementById("file-input");

    if (!dropzone || !fileInput) return;

    dropzone.addEventListener("dragover", e => {
        e.preventDefault();
        dropzone.classList.add("dragover");
    });

    dropzone.addEventListener("dragleave", () => {
        dropzone.classList.remove("dragover");
    });

    dropzone.addEventListener("drop", e => {
        e.preventDefault();
        dropzone.classList.remove("dragover");
        if (e.dataTransfer.files.length > 0) {
            handleFiles(e.dataTransfer.files);
        }
    });

    fileInput.addEventListener("change", e => {
        if (e.target.files.length > 0) {
            handleFiles(e.target.files);
        }
    });
}

function handleFiles(files) {
    for (let i = 0; i < files.length; i++) {
        uploadQueue.push(files[i]);
    }
    processUploadQueue();
}

async function processUploadQueue() {
    if (isUploading || uploadQueue.length === 0) return;
    isUploading = true;

    const file = uploadQueue.shift();
    const progressContainer = document.getElementById("upload-progress");
    const progressBar = document.getElementById("progress-bar");
    const statusText = document.getElementById("upload-status");

    if (progressContainer) progressContainer.style.display = "block";

    try {
        if (statusText) statusText.innerText = `Initializing upload: ${file.name}...`;

        // 1. Init upload session
        const initRes = await fetch("/api/upload/init", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                name: file.name,
                size: file.size,
                mime_type: file.type || "application/octet-stream",
                folder_id: currentFolderId
            })
        });

        if (!initRes.ok) throw new Error("Upload initialization failed");
        const session = await initRes.json();
        const chunkSize = session.chunk_size || 5 * 1024 * 1024;
        const totalChunks = Math.ceil(file.size / chunkSize);

        // 2. Upload chunks
        for (let chunkIdx = 0; chunkIdx < totalChunks; chunkIdx++) {
            const start = chunkIdx * chunkSize;
            const end = Math.min(file.size, start + chunkSize);
            const chunkBlob = file.slice(start, end);

            const formData = new FormData();
            formData.append("session_id", session.id);
            formData.append("chunk_index", chunkIdx);
            formData.append("chunk", chunkBlob);

            if (statusText) statusText.innerText = `Uploading ${file.name}: chunk ${chunkIdx + 1}/${totalChunks}...`;

            let retries = 3;
            let success = false;
            while (retries > 0 && !success) {
                try {
                    const chunkRes = await fetch("/api/upload/chunk", {
                        method: "POST",
                        body: formData
                    });
                    if (chunkRes.ok) success = true;
                    else retries--;
                } catch (e) {
                    retries--;
                    await new Promise(r => setTimeout(r, 1000));
                }
            }
            if (!success) throw new Error(`Failed to upload chunk ${chunkIdx}`);

            const pct = Math.round(((end) / file.size) * 100);
            if (progressBar) progressBar.style.width = `${pct}%`;
        }

        // 3. Complete upload
        if (statusText) statusText.innerText = `Committing to Telegram Storage Channel...`;
        const completeRes = await fetch("/api/upload/complete", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ session_id: session.id })
        });
        if (!completeRes.ok) throw new Error("Complete upload failed");

        if (statusText) statusText.innerText = `Upload finished: ${file.name}`;
        loadDriveContent();
    } catch (err) {
        alert("Upload error: " + err.message);
    } finally {
        isUploading = false;
        setTimeout(() => {
            if (uploadQueue.length === 0 && progressContainer) {
                progressContainer.style.display = "none";
                if (progressBar) progressBar.style.width = "0%";
            } else {
                processUploadQueue();
            }
        }, 1000);
    }
}

// Media Preview
function previewFile(id, name, mime, size) {
    const modal = document.getElementById("preview-modal");
    const container = document.getElementById("preview-content");
    const title = document.getElementById("preview-title");

    if (!modal || !container || !title) return;

    title.innerText = name;
    const streamUrl = `/api/files/${id}/stream`;

    if (mime.startsWith("video/")) {
        container.innerHTML = `
            <video controls autoplay style="width: 100%; max-height: 70vh; border-radius: 8px;">
                <source src="${streamUrl}" type="${mime}">
                Your browser does not support the video tag.
            </video>`;
    } else if (mime.startsWith("audio/")) {
        container.innerHTML = `
            <div style="padding: 40px; text-align: center;">
                <audio controls autoplay style="width: 100%;">
                    <source src="${streamUrl}" type="${mime}">
                </audio>
            </div>`;
    } else if (mime.startsWith("image/")) {
        container.innerHTML = `
            <div style="text-align: center;">
                <img src="${streamUrl}" style="max-width: 100%; max-height: 70vh; border-radius: 8px;">
            </div>`;
    } else if (mime === "application/pdf") {
        container.innerHTML = `
            <iframe src="${streamUrl}" style="width: 100%; height: 70vh; border: none; border-radius: 8px;"></iframe>`;
    } else {
        container.innerHTML = `
            <div style="padding: 40px; text-align: center; color: var(--text-secondary);">
                <p>Preview not supported for this file type.</p>
                <button class="btn-upload" style="margin: 20px auto;" onclick="downloadFile('${id}', '${escapeHtml(name)}')">⬇️ Download File (${formatSize(size)})</button>
            </div>`;
    }

    modal.style.display = "flex";
}

function closePreview() {
    const modal = document.getElementById("preview-modal");
    const container = document.getElementById("preview-content");
    if (container) container.innerHTML = "";
    if (modal) modal.style.display = "none";
}

function downloadFile(id, name) {
    window.location.href = `/api/files/${id}/download`;
}

// Modals and Prompts
async function createFolderPrompt() {
    const name = prompt("Enter folder name:");
    if (!name || !name.trim()) return;

    await fetch("/api/folders", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim(), parent_id: currentFolderId })
    });
    loadDriveContent();
}

async function renameFolder(id, oldName) {
    const name = prompt("Enter new folder name:", oldName);
    if (!name || !name.trim() || name === oldName) return;

    await fetch(`/api/folders/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim() })
    });
    loadDriveContent();
}

async function deleteFolder(id) {
    if (!confirm("Are you sure you want to delete this folder and all its contents?")) return;
    await fetch(`/api/folders/${id}`, { method: "DELETE" });
    loadDriveContent();
}

async function renameFile(id, oldName) {
    const name = prompt("Enter new file name:", oldName);
    if (!name || !name.trim() || name === oldName) return;

    await fetch(`/api/files/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim() })
    });
    loadDriveContent();
}

async function deleteFile(id) {
    if (!confirm("Are you sure you want to delete this file?")) return;
    await fetch(`/api/files/${id}`, { method: "DELETE" });
    loadDriveContent();
}

// Share Modal
let activeShareFileId = null;

function openShareModal(fileId, fileName) {
    activeShareFileId = fileId;
    document.getElementById("share-file-name").innerText = fileName;
    document.getElementById("share-result").style.display = "none";
    document.getElementById("share-modal").style.display = "flex";
}

function closeShareModal() {
    document.getElementById("share-modal").style.display = "none";
}

async function createShareLink() {
    const password = document.getElementById("share-password").value.trim();
    const expiryDays = parseInt(document.getElementById("share-expiry").value);

    const res = await fetch("/api/share", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            file_id: activeShareFileId,
            password: password || null,
            expiry_days: expiryDays || null
        })
    });

    if (!res.ok) {
        alert("Failed to create share link");
        return;
    }

    const data = await res.json();
    const shareUrl = `${window.location.origin}/s/${data.token}`;
    document.getElementById("share-link-input").value = shareUrl;
    document.getElementById("share-result").style.display = "block";
}

function copyShareLink() {
    const input = document.getElementById("share-link-input");
    input.select();
    navigator.clipboard.writeText(input.value);
    alert("Share link copied to clipboard!");
}
