async function api(url, opts = {}) {
  opts.headers = opts.headers || {};
  const token = localStorage.getItem("token");
  if (token) opts.headers["Authorization"] = "Bearer " + token;

  const r = await fetch(url, opts);

  // если сессия протухла / нет токена
  if (r.status === 401) {
    localStorage.removeItem("token");
    window.location.href = "/login";
    return;
  }

  if (!r.ok) throw new Error(await r.text());
  return r.status === 204 ? null : r.json();
}

function el(id) { return document.getElementById(id); }
function msg(id, t) { el(id).textContent = t; }
function requireAuth() {
  const token = localStorage.getItem("token");
  if (!token) window.location.href = "/login";
}
function logout() {
  localStorage.removeItem("token");
  window.location.href = "/login";
}

function fmtDate(iso) {
  try { return new Date(iso).toLocaleString(); } catch { return iso || ""; }
}
function safe(obj) { return JSON.stringify(obj, null, 2); }

function currentUserId() {
  const t = localStorage.getItem("token");
  if (!t) return "";
  try {
    return JSON.parse(atob(t.split(".")[1])).sub || "";
  } catch {
    return "";
  }
}

function profileKey() {
  const uid = currentUserId();
  return uid ? `profile_v1:${uid}` : "profile_v1:anonymous";
}

function getProfile() {
  try {
    const raw = localStorage.getItem(profileKey());
    return raw ? JSON.parse(raw) : { name: "", bio: "", avatar: "" };
  } catch {
    return { name: "", bio: "", avatar: "" };
  }
}
function setProfile(p) {
  localStorage.setItem(profileKey(), JSON.stringify(p));
}

function applyAvatarTo(elm, avatarDataUrl) {
  if (!elm) return;
  if (avatarDataUrl) {
    elm.classList.add("hasImg");
    elm.style.backgroundImage = `url("${avatarDataUrl}")`;
  } else {
    elm.classList.remove("hasImg");
    elm.style.backgroundImage = "";
  }
}

function openProfileModal() {
  const p = getProfile();
  el("profileName").value = p.name || "";
  el("profileBio").value = p.bio || "";
  applyAvatarTo(el("avatarBig"), p.avatar || "");
  applyAvatarTo(el("navAvatar"), p.avatar || "");
  msg("profileStatus", "");
  el("profileModal").classList.remove("hidden");
  el("profileModal").setAttribute("aria-hidden", "false");
}

function closeProfileModal() {
  el("profileModal").classList.add("hidden");
  el("profileModal").setAttribute("aria-hidden", "true");
}

async function saveProfile() {
  const p = getProfile();
  p.name = (el("profileName").value || "").trim();
  p.bio = (el("profileBio").value || "").trim();

  const file = el("profileAvatar").files && el("profileAvatar").files[0];
  if (file) {
    // небольшой guard, чтобы не зафигачить гигабайт в localStorage
    if (file.size > 2 * 1024 * 1024) {
      msg("profileStatus", "Avatar too big (max 2MB for MVP)");
      return;
    }
    const dataUrl = await new Promise((resolve, reject) => {
      const fr = new FileReader();
      fr.onload = () => resolve(String(fr.result || ""));
      fr.onerror = () => reject(new Error("read avatar failed"));
      fr.readAsDataURL(file);
    });
    p.avatar = dataUrl;
  }

  setProfile(p);
  applyAvatarTo(el("avatarBig"), p.avatar || "");
  applyAvatarTo(el("navAvatar"), p.avatar || "");
  msg("profileStatus", "Saved ✓");
}

// ---------- ME ----------
async function loadMe() {
  msg("meStatus", "Loading session…");
  try {
    const res = await api("/api/app/me");
    msg("meStatus", "Session OK ✓");
    el("meBox").textContent = safe(res);
  } catch (e) {
    msg("meStatus", e.message);
  }
}

// ---------- CLUSTERS ----------
async function loadClusters() {
  try {
    const res = await api("/api/app/clusters");
    const sel = el("clusterSelect");
    sel.innerHTML = "";

    const clusters = res.clusters || [];
    for (const c of clusters) {
      const o = document.createElement("option");
      o.value = c.ID || c.id;
      o.textContent = c.Name || c.name;
      sel.appendChild(o);
    }

    msg("clusterStatus", clusters.length ? "Clusters loaded ✓" : "No clusters yet");
    if (sel.value) await loadScanHistory();
  } catch (e) {
    msg("clusterStatus", e.message);
  }
}

async function createCluster() {
  try {
    await api("/api/app/clusters", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: el("clusterName").value,
        notes: el("clusterNotes").value,
      }),
    });
    msg("clusterStatus", "Cluster added ✓");
    await loadClusters();
  } catch (e) {
    msg("clusterStatus", e.message);
  }
}

// ---------- SCANS ----------
let lastScans = [];
let selectedScanId = "";

// report state (for toggle/download)
let lastReportObj = null;
let reportExpanded = false;

function setSelectedScan(id) {
  selectedScanId = id || "";
  el("scanId").value = selectedScanId;

  const body = el("historyBody");
  body.querySelectorAll("tr[data-scan-id]").forEach(tr => {
    tr.classList.toggle("trActive", tr.getAttribute("data-scan-id") === selectedScanId);
  });
}

function renderHistory(scans) {
  const body = el("historyBody");
  body.innerHTML = "";
  lastScans = scans || [];

  if (!lastScans.length) {
    body.innerHTML = `<tr><td colspan="4" class="muted">No scans yet</td></tr>`;
    msg("historyMeta", "");
    return;
  }

  msg("historyMeta", `${lastScans.length} scan(s)`);

  for (const sc of lastScans) {
    const id = sc.ID || sc.id || "";
    const created = fmtDate(sc.CreatedAt || sc.createdAt);
    const source = sc.Source || sc.source || "";

    const tr = document.createElement("tr");
    tr.setAttribute("data-scan-id", id);
    tr.innerHTML = `
      <td>${created}</td>
      <td><span class="badge">${source || "—"}</span></td>
      <td class="code">${id}</td>
      <td>
        <button class="btn secondary" data-act="select">Select</button>
        <button class="btn" data-act="report">View</button>
      </td>
    `;

    tr.addEventListener("click", (ev) => {
      if (ev.target && ev.target.tagName === "BUTTON") return;
      setSelectedScan(id);
    });

    tr.querySelectorAll("button[data-act]").forEach(btn => {
      btn.addEventListener("click", async (ev) => {
        ev.stopPropagation();
        const act = btn.getAttribute("data-act");
        setSelectedScan(id);
        if (act === "report") {
          await loadReport();
        }
      });
    });

    body.appendChild(tr);
  }

  if (!selectedScanId) {
    setSelectedScan(lastScans[0].ID || lastScans[0].id || "");
  } else {
    setSelectedScan(selectedScanId);
  }
}

async function loadScanHistory() {
  const clusterId = el("clusterSelect").value;
  if (!clusterId) return;

  try {
    msg("scanStatus", "");
    const res = await api(`/api/app/scans?clusterId=${encodeURIComponent(clusterId)}`);
    const scans = res.scans || [];
    renderHistory(scans);
  } catch (e) {
    msg("scanStatus", e.message);
  }
}

async function uploadScan() {
  const file = el("rbacFile").files[0];
  const clusterId = el("clusterSelect").value;

  if (!file) { alert("Choose RBAC YAML file"); return; }
  if (!clusterId) { alert("Select cluster first"); return; }

  const fd = new FormData();
  fd.append("rbac", file);
  fd.append("clusterId", clusterId);

  msg("scanStatus", "Uploading & analyzing…");

  try {
    const res = await api("/api/app/scans", { method: "POST", body: fd });
    msg("scanStatus", "Scan uploaded ✓");
    el("summary").textContent = safe(res.summary || res);

    await loadScanHistory();
    if (lastScans.length) {
      setSelectedScan(lastScans[0].ID || lastScans[0].id || "");
    }
  } catch (e) {
    msg("scanStatus", e.message);
  }
}

function renderReportBox() {
  const box = el("report");
  if (!lastReportObj) {
    box.textContent = "No data";
    return;
  }

  const full = safe(lastReportObj);

  if (reportExpanded) {
    box.textContent = full;
    return;
  }

  // collapsed: show first N lines
  const lines = full.split("\n");
  const maxLines = 220;
  if (lines.length <= maxLines) {
    box.textContent = full;
  } else {
    box.textContent = lines.slice(0, maxLines).join("\n") + `\n\n… (${lines.length - maxLines} more lines hidden. Press Toggle)`;
  }
}

async function loadReport() {
  const id = (el("scanId").value || "").trim();
  if (!id) { alert("Select scan from history first"); return; }

  try {
    const res = await api(`/api/app/scan/report?scanId=${encodeURIComponent(id)}`);
    lastReportObj = res.report || res;
    reportExpanded = false; // default collapsed
    renderReportBox();
  } catch (e) {
    el("report").textContent = e.message;
  }
}

async function loadLatestReport() {
  if (!lastScans.length) {
    alert("No scans yet");
    return;
  }
  setSelectedScan(lastScans[0].ID || lastScans[0].id || "");
  await loadReport();
}

function toggleReport() {
  reportExpanded = !reportExpanded;
  renderReportBox();
}

function downloadReport() {
  if (!lastReportObj) {
    alert("Load report first");
    return;
  }
  const data = safe(lastReportObj);
  const blob = new Blob([data], { type: "application/json;charset=utf-8" });
  const url = URL.createObjectURL(blob);

  const a = document.createElement("a");
  a.href = url;
  const sid = (el("scanId").value || "scan").trim();
  a.download = `rbac-report-${sid}.json`;
  document.body.appendChild(a);
  a.click();
  a.remove();

  setTimeout(() => URL.revokeObjectURL(url), 500);
}

// ---------- INIT ----------
window.addEventListener("DOMContentLoaded", async () => {
  requireAuth();

  // profile nav
  const p = getProfile();
  applyAvatarTo(el("navAvatar"), p.avatar || "");
  el("profileBtn").onclick = openProfileModal;
  el("closeProfile").onclick = closeProfileModal;
  el("profileOverlay").onclick = closeProfileModal;
  el("saveProfile").onclick = saveProfile;

  // logout
  el("logoutBtn").onclick = logout;

  // clusters
  el("createCluster").onclick = createCluster;
  el("refreshClusters").onclick = loadClusters;

  // scans
  el("uploadScan").onclick = uploadScan;
  el("loadScans").onclick = loadScanHistory;
  el("loadReport").onclick = loadReport;
  el("loadLatestReport").onclick = loadLatestReport;

  // report tools
  el("toggleReport").onclick = toggleReport;
  el("downloadReport").onclick = downloadReport;

  // misc
  el("clearSummary").onclick = () => { el("summary").textContent = "No data"; };
  el("clusterSelect").addEventListener("change", loadScanHistory);

  await loadMe();
  await loadClusters();
});
