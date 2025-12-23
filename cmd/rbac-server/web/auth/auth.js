async function api(url, opts = {}) {
  opts.headers = opts.headers || {};
  const token = localStorage.getItem("token");
  if (token) opts.headers["Authorization"] = "Bearer " + token;

  const r = await fetch(url, opts);
  if (!r.ok) {
    const t = await r.text();
    throw new Error(t || ("HTTP " + r.status));
  }
  return r.status === 204 ? null : r.json();
}

function el(id) { return document.getElementById(id); }
function setMsg(id, t, isErr = false) {
  const node = el(id);
  node.className = isErr ? "muted err" : "muted";
  node.textContent = t || "";
}

function redirectIfAuthed() {
  const token = localStorage.getItem("token");
  if (token) window.location.href = "/app";
}

// ---------- LOGIN ----------
async function doLogin() {
  setMsg("status", "Signing in…");
  try {
    const res = await api("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        email: el("email").value.trim(),
        password: el("password").value,
      }),
    });
    localStorage.setItem("token", res.token);
    window.location.href = "/app";
  } catch (e) {
    setMsg("status", e.message, true);
  }
}

// ---------- REGISTER ----------
async function doRegister() {
  setMsg("status", "Creating account…");
  try {
    const res = await api("/api/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        email: el("email").value.trim(),
        password: el("password").value,
        orgName: (el("orgName") ? el("orgName").value.trim() : ""),
      }),
    });
    localStorage.setItem("token", res.token);
    window.location.href = "/app";
  } catch (e) {
    setMsg("status", e.message, true);
  }
}

window.addEventListener("DOMContentLoaded", () => {
  // If already authed -> go app
  redirectIfAuthed();

  // login page?
  const loginBtn = document.getElementById("loginBtn");
  if (loginBtn) {
    loginBtn.addEventListener("click", doLogin);
    return;
  }

  // register page?
  const regBtn = document.getElementById("registerBtn");
  if (regBtn) {
    regBtn.addEventListener("click", doRegister);
  }
});
