async function api(url, opts = {}) {
  opts.headers = opts.headers || {};
  const token = localStorage.getItem("token");
  if (token) opts.headers["Authorization"] = "Bearer " + token;

  const r = await fetch(url, opts);
  if (r.status === 401) {
    localStorage.removeItem("token");
    window.location.href = "/login";
    return;
  }
  if (r.status === 403) {
    alert("Admin only");
    window.location.href = "/app";
    return;
  }
  if (!r.ok) throw new Error(await r.text());
  return r.status === 204 ? null : r.json();
}

function el(id) { return document.getElementById(id); }
function fmt(iso) { try { return new Date(iso).toLocaleString(); } catch { return iso || ""; } }
function esc(s){return String(s||"").replaceAll("&","&amp;").replaceAll("<","&lt;").replaceAll(">","&gt;").replaceAll('"',"&quot;").replaceAll("'","&#039;");}

function logout() {
  localStorage.removeItem("token");
  window.location.href = "/login";
}

async function loadUsers() {
  el("usersStatus").textContent = "";
  const body = el("usersBody");
  body.innerHTML = `<tr><td colspan="4" class="muted">Loading…</td></tr>`;
  try {
    const res = await api("/api/admin/users?limit=200");
    const users = res.users || [];
    body.innerHTML = "";
    if (!users.length) {
      body.innerHTML = `<tr><td colspan="4" class="muted">No users</td></tr>`;
      return;
    }
    for (const u of users) {
      const tr = document.createElement("tr");
      tr.innerHTML = `
        <td>${fmt(u.createdAt)}</td>
        <td>${esc(u.email)}</td>
        <td>
          <label class="badge" style="cursor:pointer;">
            <input type="checkbox" data-user-id="${esc(u.id)}" ${u.isAdmin ? "checked" : ""} />
            ${u.isAdmin ? "admin" : "user"}
          </label>
        </td>
        <td class="code">${esc(u.id)}</td>
      `;
      body.appendChild(tr);
    }

    body.querySelectorAll("input[type=checkbox][data-user-id]").forEach(ch => {
      ch.addEventListener("change", async () => {
        const userId = ch.getAttribute("data-user-id");
        const isAdmin = !!ch.checked;
        try {
          await api("/api/admin/users", {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ userId, isAdmin })
          });
          ch.parentElement.innerHTML = `
            <input type="checkbox" data-user-id="${esc(userId)}" ${isAdmin ? "checked" : ""} />
            ${isAdmin ? "admin" : "user"}
          `;
          el("usersStatus").textContent = "Saved ✓";
        } catch (e) {
          el("usersStatus").textContent = e.message;
          ch.checked = !isAdmin;
        }
      });
    });

    el("usersStatus").textContent = `Loaded ${users.length} user(s)`;
  } catch (e) {
    el("usersStatus").textContent = e.message;
  }
}

async function loadOrgs() {
  el("orgsStatus").textContent = "";
  const body = el("orgsBody");
  body.innerHTML = `<tr><td colspan="6" class="muted">Loading…</td></tr>`;
  try {
    const res = await api("/api/admin/orgs?limit=200");
    const orgs = res.orgs || [];
    body.innerHTML = "";
    if (!orgs.length) {
      body.innerHTML = `<tr><td colspan="6" class="muted">No orgs</td></tr>`;
      return;
    }
    for (const o of orgs) {
      const tr = document.createElement("tr");
      tr.innerHTML = `
        <td>${fmt(o.createdAt)}</td>
        <td>${esc(o.name)}</td>
        <td>${esc(o.ownerEmail || "")}</td>
        <td>
          <select data-org-id="${esc(o.id)}" class="badge" style="background:rgba(255,255,255,0.06);">
            <option value="free" ${o.planId === "free" ? "selected" : ""}>free</option>
            <option value="pro" ${o.planId === "pro" ? "selected" : ""}>pro</option>
            <option value="enterprise" ${o.planId === "enterprise" ? "selected" : ""}>enterprise</option>
          </select>
        </td>
        <td>${o.clustersCount} / ${o.maxClusters}</td>
        <td class="code">${esc(o.id)}</td>
      `;
      body.appendChild(tr);
    }

    body.querySelectorAll("select[data-org-id]").forEach(sel => {
      sel.addEventListener("change", async () => {
        const orgId = sel.getAttribute("data-org-id");
        const planId = sel.value;
        try {
          await api("/api/admin/orgs", {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ orgId, planId })
          });
          el("orgsStatus").textContent = "Saved ✓";
          await loadOrgs();
        } catch (e) {
          el("orgsStatus").textContent = e.message;
        }
      });
    });

    el("orgsStatus").textContent = `Loaded ${orgs.length} org(s)`;
  } catch (e) {
    el("orgsStatus").textContent = e.message;
  }
}

window.addEventListener("DOMContentLoaded", async () => {
  el("logoutBtn").onclick = logout;
  el("reloadUsers").onclick = loadUsers;
  el("reloadOrgs").onclick = loadOrgs;

  await loadUsers();
  await loadOrgs();
});
