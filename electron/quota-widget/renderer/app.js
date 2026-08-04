const state = {
  configured: false,
  data: null,
  view: "balance",
  dragging: false,
  moved: false,
  startX: 0,
  startY: 0
};

const el = {
  widget: document.querySelector("#widget"),
  configForm: document.querySelector("#configForm"),
  sitePreset: document.querySelector("#sitePreset"),
  baseUrl: document.querySelector("#baseUrl"),
  apiKey: document.querySelector("#apiKey"),
  balanceFace: document.querySelector("#balanceFace"),
  logsFace: document.querySelector("#logsFace"),
  balance: document.querySelector("#balance"),
  logList: document.querySelector("#logList"),
  message: document.querySelector("#message")
};

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function showMessage(message) {
  el.message.hidden = !message;
  el.message.textContent = message || "";
}

function resize(view) {
  requestAnimationFrame(() => {
    if (view === "balance") {
      const contentWidth = Math.ceil(el.balance.getBoundingClientRect().width);
      window.quotaWidget.resize(Math.min(240, Math.max(72, contentWidth + 22)), 44);
    } else if (view === "logs") {
      window.quotaWidget.resize(320, 280);
    } else {
      window.quotaWidget.resize(286, el.sitePreset.value === "custom" ? 154 : 118);
    }
  });
}

function updateSiteMode({ focus = false } = {}) {
  const custom = el.sitePreset.value === "custom";
  el.baseUrl.hidden = !custom;
  el.configForm.classList.toggle("is-custom", custom);
  if (!custom) el.baseUrl.value = "";
  resize("config");
  if (custom && focus) requestAnimationFrame(() => el.baseUrl.focus());
}

function selectedBaseUrl() {
  return el.sitePreset.value === "custom" ? el.baseUrl.value : el.sitePreset.value;
}

function setView(view) {
  if (!state.configured) return;
  state.view = view === "logs" ? "logs" : "balance";
  el.configForm.hidden = true;
  el.balanceFace.hidden = state.view !== "balance";
  el.logsFace.hidden = state.view !== "logs";
  showMessage("");
  window.quotaWidget.setView(state.view);
  resize(state.view);
}

function showConfig(defaultBaseUrl) {
  state.configured = false;
  state.data = null;
  el.configForm.hidden = false;
  el.balanceFace.hidden = true;
  el.logsFace.hidden = true;
  const initialUrl = defaultBaseUrl || "https://tokease.com";
  const knownPreset = ["https://tokease.com", "https://tokease.cn"].includes(initialUrl);
  el.sitePreset.value = knownPreset ? initialUrl : "custom";
  el.baseUrl.value = knownPreset ? "" : initialUrl;
  el.apiKey.value = "";
  showMessage("");
  updateSiteMode();
  requestAnimationFrame(() => el.apiKey.focus());
}

function render(data) {
  if (!data) return;
  state.configured = true;
  state.data = data;
  el.balance.textContent = data.balance || "--";
  const logs = Array.isArray(data.logs) ? data.logs : [];
  el.logList.innerHTML = logs.length
    ? logs.map((item) => `
      <div class="log-row">
        <div class="log-top">
          <span class="model">${escapeHtml(item.model)}</span>
          <strong>${escapeHtml(item.amount)}</strong>
        </div>
        <div class="log-meta">${escapeHtml(item.time)} · ${escapeHtml(item.promptTokens)} / ${escapeHtml(item.completionTokens)} · ${escapeHtml(item.duration)}</div>
      </div>
    `).join("")
    : '<div class="empty">--</div>';
  setView(state.view);
}

el.configForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const button = el.configForm.querySelector("button");
  button.disabled = true;
  showMessage("");
  try {
    const result = await window.quotaWidget.configure(selectedBaseUrl(), el.apiKey.value);
    if (!result.success) throw new Error(result.message || "查询失败");
    el.baseUrl.value = "";
    el.apiKey.value = "";
    state.view = "balance";
    render(result.data);
  } catch (error) {
    showMessage(error.message || "查询失败");
  } finally {
    button.disabled = false;
  }
});

el.widget.addEventListener("mousedown", (event) => {
  if (event.button !== 0 || event.target.closest("input, button")) return;
  state.dragging = true;
  state.moved = false;
  state.startX = event.screenX;
  state.startY = event.screenY;
  window.quotaWidget.startDrag(event.screenX, event.screenY);
});

window.addEventListener("mousemove", (event) => {
  if (!state.dragging) return;
  if (Math.abs(event.screenX - state.startX) + Math.abs(event.screenY - state.startY) > 4) {
    state.moved = true;
  }
  window.quotaWidget.dragTo(event.screenX, event.screenY);
});

window.addEventListener("mouseup", () => {
  if (!state.dragging) return;
  const toggle = !state.moved && state.configured;
  state.dragging = false;
  window.quotaWidget.endDrag();
  if (toggle) setView(state.view === "balance" ? "logs" : "balance");
});

window.addEventListener("blur", () => {
  if (!state.dragging) return;
  state.dragging = false;
  window.quotaWidget.endDrag();
});

window.quotaWidget.onData((data) => render(data));
window.quotaWidget.onQueryError(() => {});
window.quotaWidget.onMenuAction((action) => {
  if (action?.type === "view") setView(action.value);
  if (action?.type === "reset") showConfig("https://tokease.com");
});

el.sitePreset.addEventListener("change", () => updateSiteMode({ focus: true }));

window.quotaWidget.getState().then((initial) => {
  if (initial.configured && initial.data) {
    render(initial.data);
  } else {
    showConfig(initial.defaultBaseUrl);
  }
});
