const fileEl = document.getElementById("file");
const namespaceEl = document.getElementById("namespace");
const dangerOnlyEl = document.getElementById("dangerOnly");
const analyzeBtn = document.getElementById("analyze");
const resultsEl = document.getElementById("results");
const statusEl = document.getElementById("status");
const downloadBtn = document.getElementById("download");

let lastJSON = null;

function setStatus(msg) {
  statusEl.textContent = msg || "";
}

function escapeHtml(s) {
  return s.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
}

function render(resp) {
  if (!resp || !resp.subjects || resp.subjects.length === 0) {
    resultsEl.className = "results muted";
    resultsEl.textContent = "Нет данных (возможно, после фильтрации ничего не осталось)";
    return;
  }

  resultsEl.className = "results";
  const parts = [];
  for (const subj of resp.subjects) {
    parts.push(`<div class="subject">`);
    parts.push(`<h3>${escapeHtml(subj.subject)}</h3>`);

    for (const role of subj.roles) {
      const scope = role.clusterScope ? "cluster" : "namespace";
      const dangerBadge = role.dangerous ? `<span class="badge danger">DANGEROUS</span>` : "";
      const ns = role.sourceNamespace && role.sourceNamespace.length ? role.sourceNamespace : "-";
      parts.push(`<div class="role">`);
      parts.push(`<div class="head">
        <span class="badge">${escapeHtml(role.sourceKind)}:${escapeHtml(ns)}/${escapeHtml(role.sourceName)}</span>
        <span class="badge scope">scope:${escapeHtml(scope)}</span>
        <span class="badge">via:${escapeHtml(role.boundVia)}/${escapeHtml(role.bindingName || "")}</span>
        ${dangerBadge}
      </div>`);

      if (role.dangerReasons && role.dangerReasons.length) {
        parts.push(`<div class="kv">Danger reasons:</div><ul class="list">`);
        for (const rr of role.dangerReasons) parts.push(`<li>${escapeHtml(rr)}</li>`);
        parts.push(`</ul>`);
      }

      parts.push(`<div class="kv">Permissions: ${role.permissions ? role.permissions.length : 0}</div>`);
      parts.push(`</div>`);
    }

    parts.push(`</div>`);
  }

  resultsEl.innerHTML = parts.join("\n");
}

analyzeBtn.addEventListener("click", async () => {
  const f = fileEl.files && fileEl.files[0];
  if (!f) {
    setStatus("Выбери файл YAML.");
    return;
  }

  analyzeBtn.disabled = true;
  downloadBtn.disabled = true;
  lastJSON = null;
  setStatus("Анализируем...");

  try {
    const fd = new FormData();
    fd.append("rbac", f);

    const ns = (namespaceEl.value || "").trim();
    const dangerOnly = dangerOnlyEl.checked;

    const url = `/api/analyze?dangerOnly=${dangerOnly ? "true" : "false"}&namespace=${encodeURIComponent(ns)}`;
    const res = await fetch(url, { method: "POST", body: fd });

    if (!res.ok) {
      const t = await res.text();
      throw new Error(t);
    }

    const json = await res.json();
    lastJSON = json;
    render(json);
    downloadBtn.disabled = false;
    setStatus("Готово.");
  } catch (e) {
    resultsEl.className = "results";
    resultsEl.innerHTML = `<pre>${escapeHtml(String(e.message || e))}</pre>`;
    setStatus("Ошибка.");
  } finally {
    analyzeBtn.disabled = false;
  }
});

downloadBtn.addEventListener("click", () => {
  if (!lastJSON) return;
  const blob = new Blob([JSON.stringify(lastJSON, null, 2)], { type: "application/json" });
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = "rbac-report.json";
  a.click();
  URL.revokeObjectURL(a.href);
});
