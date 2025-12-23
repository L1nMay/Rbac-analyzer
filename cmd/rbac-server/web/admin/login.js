async function login() {
  const email = document.getElementById("email").value;
  const password = document.getElementById("password").value;
  const status = document.getElementById("status");

  status.textContent = "Checking…";

  try {
    const r = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });

    if (!r.ok) throw new Error(await r.text());
    const res = await r.json();

    const payload = JSON.parse(atob(res.token.split(".")[1]));
    if (!payload.admin) {
      throw new Error("Not an admin account");
    }

    localStorage.setItem("adminToken", res.token);
    window.location.href = "/admin/dashboard";
  } catch (e) {
    status.textContent = e.message;
  }
}

// already logged?
(function () {
  const t = localStorage.getItem("adminToken");
  if (!t) return;
  try {
    const p = JSON.parse(atob(t.split(".")[1]));
    if (p.admin) window.location.href = "/admin/dashboard";
  } catch {}
})();

document.getElementById("loginBtn").onclick = login;
