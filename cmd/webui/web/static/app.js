let selectedFile = null;

const dropzone = document.getElementById("dropzone");
const fileInput = document.getElementById("fileInput");
const uploadBtn = document.getElementById("uploadBtn");
const statusEl = document.getElementById("status");
const jobsEl = document.getElementById("jobs");
const jobDetailsEl = document.getElementById("jobDetails");
const refreshBtn = document.getElementById("refreshBtn");

function setStatus(msg) {
    statusEl.textContent = msg || "";
}

function humanBytes(n) {
    if (n == null) return "";
    const units = ["B", "KB", "MB", "GB"];
    let i = 0;
    let x = Number(n);
    while (x >= 1024 && i < units.length - 1) { x /= 1024; i++; }
    return `${x.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function pickFile(f) {
    selectedFile = f;
    uploadBtn.disabled = !selectedFile;
    if (selectedFile) setStatus(`Selected: ${selectedFile.name} (${humanBytes(selectedFile.size)})`);
}

dropzone.addEventListener("dragover", (e) => {
    e.preventDefault();
    dropzone.classList.add("dragover");
});

dropzone.addEventListener("dragleave", () => {
    dropzone.classList.remove("dragover");
});

dropzone.addEventListener("drop", (e) => {
    e.preventDefault();
    dropzone.classList.remove("dragover");
    const f = e.dataTransfer.files && e.dataTransfer.files[0];
    if (f) pickFile(f);
});

fileInput.addEventListener("change", () => {
    const f = fileInput.files && fileInput.files[0];
    if (f) pickFile(f);
});

uploadBtn.addEventListener("click", async () => {
    if (!selectedFile) return;

    setStatus("Uploading...");
    uploadBtn.disabled = true;

    try {
        const fd = new FormData();
        fd.append("file", selectedFile);

        const res = await fetch("/api/upload", { method: "POST", body: fd });
        if (!res.ok) {
            const txt = await res.text();
            throw new Error(`Upload failed: ${txt}`);
        }

        const out = await res.json();
        setStatus("Processed ✅");
        selectedFile = null;
        fileInput.value = "";
        await loadJobs();
        if (out && out.id) await loadJob(out.id);
    } catch (err) {
        console.error(err);
        setStatus(err.message || "Error");
    } finally {
        uploadBtn.disabled = true;
    }
});

refreshBtn.addEventListener("click", async () => {
    await loadJobs();
});

async function loadJobs() {
    const res = await fetch("/api/jobs");
    const jobs = await res.json();

    jobsEl.innerHTML = "";

    if (!jobs || jobs.length === 0) {
        jobsEl.innerHTML = `<div class="meta">No jobs yet.</div>`;
        return;
    }

    jobs.forEach((j) => {
        const badgeClass = j.status === "done" ? "done" : (j.status === "error" ? "error" : "");
        const created = j.createdAt ? new Date(j.createdAt).toLocaleString() : "";
        const el = document.createElement("div");
        el.className = "job";
        el.innerHTML = `
      <div class="top">
        <div class="name" title="${j.filename || ""}">${j.filename || "(unnamed)"}</div>
        <div class="badge ${badgeClass}">${j.status || "?"}</div>
      </div>
      <div class="meta">id: ${j.id || ""}</div>
      <div class="meta">${humanBytes(j.sizeBytes)} · checksum ${j.checksum || ""} · ${created}</div>
    `;
        el.addEventListener("click", () => loadJob(j.id));
        jobsEl.appendChild(el);
    });
}

async function loadJob(id) {
    const res = await fetch(`/api/job/${encodeURIComponent(id)}`);
    if (!res.ok) {
        jobDetailsEl.textContent = "(not found)";
        return;
    }
    const j = await res.json();
    jobDetailsEl.textContent = JSON.stringify(j, null, 2);
}

loadJobs().catch(console.error);
