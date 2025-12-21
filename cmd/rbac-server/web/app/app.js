let token = localStorage.getItem("jwt") || "";

const el = (id) => document.getElementById(id);

function setText(id, t) { el(id).textContent = t; }
function setJSON(id, obj) { el(id).textContent = JSON.stringify(obj, null, 2); }
function authHeaders() { return token ? { "Authorization": `Bearer ${token}` } : {}; }

async function api(path, opts = {}) {
  const res = await fetch(path, opts);
  if (!res.ok) {
    const txt = await res.text().catch(() => "");
    throw new Error(txt || `HTTP ${res.status}`);
  }
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) return res.json();
  return res.text();
}

function setAuthStatus(msg) { setText("authStatus", msg); }
function setClusterStatus(msg) { setText("clusterStatus", msg); }
function setScanStatus(msg) { setText("scanStatus", msg); }

async function register() {
  const email = el("email").value.trim();
  const password = el("password").value;
  const orgName = el("orgName").value.trim() || "My Organization";

  const data = await api("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, orgName })
  });

  token = data.token;
  localStorage.setItem("jwt", token);
  setAuthStatus("Registered + logged in.");
  await refreshClusters();
}

async function login() {
  const email = el("email").value.trim();
  const password = el("password").value;

  const data = await api("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password })
  });

  token = data.token;
  localStorage.setItem("jwt", token);
  setAuthStatus("Logged in.");
  await refreshClusters();
}

async function refreshClusters() {
  if (!token) { setClusterStatus("Login first."); return; }
  const data = await api("/api/app/clusters", { headers: authHeaders() });
  const select = el("clusterSelect");
  select.innerHTML = "";
  (data.clusters || []).forEach(c => {
    const opt = document.createElement("option");
    opt.value = c.ID || c.id;
    opt.textContent = `${c.Name || c.name}`;
    select.appendChild(opt);
  });
  setClusterStatus(`Clusters loaded: ${(data.clusters || []).length}`);
}

async function createCluster() {
  if (!token) { setClusterStatus("Login first."); return; }

  const name = el("clusterName").value.trim();
  const notes = el("clusterNotes").value.trim();
  if (!name) { setClusterStatus("Cluster name required."); return; }

  try {
    const created = await api("/api/app/clusters", {
      method: "POST",
      headers: { ...authHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify({ name, notes })
    });
    setClusterStatus(`Created: ${created.name || created.Name || created.id}`);
    await refreshClusters();
  } catch (e) {
    setClusterStatus(String(e.message || e));
  }
}

async function uploadScan() {
  if (!token) { setScanStatus("Login first."); return; }
  const clusterId = el("clusterSelect").value;
  if (!clusterId) { setScanStatus("Create/select a cluster first."); return; }

  const f = el("rbacFile").files && el("rbacFile").files[0];
  if (!f) { setScanStatus("Choose rbac.yaml file first."); return; }

  const fd = new FormData();
  fd.append("clusterId", clusterId);
  fd.append("rbac", f);

  setScanStatus("Uploading & analyzing...");
  const data = await api("/api/app/scans", {
    method: "POST",
    headers: authHeaders(),
    body: fd
  });

  setJSON("summary", data.summary);
  setText("scanId", data.scan.id || data.scan.ID);
  setScanStatus(`Scan created: ${data.scan.id || data.scan.ID}`);
}

async function loadHistory() {
  if (!token) { setScanStatus("Login first."); return; }
  const clusterId = el("clusterSelect").value;
  if (!clusterId) { setScanStatus("Select a cluster."); return; }

  const data = await api(`/api/app/scans?clusterId=${encodeURIComponent(clusterId)}`, {
    headers: authHeaders()
  });
  setJSON("history", data);
}

async function loadReport() {
  if (!token) { setScanStatus("Login first."); return; }
  const scanId = el("scanId").value.trim();
  if (!scanId) { setScanStatus("Enter scanId."); return; }

  const data = await api(`/api/app/scan/report?scanId=${encodeURIComponent(scanId)}`, {
    headers: authHeaders()
  });
  setJSON("report", data);
  setScanStatus("Report loaded.");
}

el("register").addEventListener("click", () => register().catch(e => setAuthStatus(String(e.message || e))));
el("login").addEventListener("click", () => login().catch(e => setAuthStatus(String(e.message || e))));
el("refreshClusters").addEventListener("click", () => refreshClusters().catch(e => setClusterStatus(String(e.message || e))));
el("createCluster").addEventListener("click", () => createCluster().catch(e => setClusterStatus(String(e.message || e))));
el("uploadScan").addEventListener("click", () => uploadScan().catch(e => setScanStatus(String(e.message || e))));
el("loadScans").addEventListener("click", () => loadHistory().catch(e => setScanStatus(String(e.message || e))));
el("loadReport").addEventListener("click", () => loadReport().catch(e => setScanStatus(String(e.message || e))));

if (token) {
  setAuthStatus("Token found. Refresh clusters.");
}
