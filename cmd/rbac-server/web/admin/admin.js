// ---------- ADMIN API ----------
async function api(url, opts = {}) {
  opts.headers = opts.headers || {};

  const token = localStorage.getItem("adminToken");
  if (!token) {
    window.location.href = "/admin";
    return;
  }

  opts.headers["Authorization"] = "Bearer " + token;

  const r = await fetch(url, opts);

  if (r.status === 401 || r.status === 403) {
    localStorage.removeItem("adminToken");
    window.location.href = "/admin";
    return;
  }

  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

function el(id) {
  return document.getElementById(id);
}

// ---------- LOGOUT ----------
function logout() {
  localStorage.removeItem("adminToken");
  window.location.href = "/admin";
}

function fmtDate(v) {
  try {
    return new Date(v).toLocaleString();
  } catch {
    return v;
  }
}

// ---------- USERS ----------
async function loadUsers() {
  const body = el("usersBody");
  body.innerHTML = `<tr><td colspan="4" class="muted">Loading…</td></tr>`;

  const res = await api("/api/admin/users");
  body.innerHTML = "";

  for (const u of res.users) {
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td>${u.email}</td>
      <td>${u.isAdmin ? "✅" : "—"}</td>
      <td>${fmtDate(u.createdAt)}</td>
      <td>
        <button class="btn secondary" data-id="${u.id}">
          ${u.isAdmin ? "Revoke" : "Make admin"}
        </button>
      </td>
    `;

    tr.querySelector("button").onclick = async () => {
      await api(`/api/admin/users/${u.id}/toggle-admin`, { method: "POST" });
      await loadUsers();
    };

    body.appendChild(tr);
  }
}

// ---------- ORGS ----------
async function loadOrgs() {
  const body = el("orgsBody");
  body.innerHTML = `<tr><td colspan="3" class="muted">Loading…</td></tr>`;

  const res = await api("/api/admin/orgs");
  body.innerHTML = "";

  for (const o of res.orgs) {
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td>${o.name}</td>
      <td>${o.ownerEmail}</td>
      <td>${fmtDate(o.createdAt)}</td>
    `;
    body.appendChild(tr);
  }
}

// ---------- INIT ----------
window.addEventListener("DOMContentLoaded", async () => {
  // если токен отсутствует → login
  const token = localStorage.getItem("adminToken");
  if (!token) {
    window.location.href = "/admin";
    return;
  }

  // показать email админа
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    if (!payload.admin) {
      throw new Error("Not admin");
    }
    el("adminEmail").textContent = payload.email;
  } catch {
    localStorage.removeItem("adminToken");
    window.location.href = "/admin";
    return;
  }

  el("logoutBtn").onclick = logout;
  el("reloadUsers").onclick = loadUsers;
  el("reloadOrgs").onclick = loadOrgs;

  await loadUsers();
  await loadOrgs();
});
