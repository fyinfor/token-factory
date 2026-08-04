const { app, BrowserWindow, ipcMain, Menu, nativeImage, screen, Tray } = require("electron");
const path = require("node:path");
const { normalizeBaseUrl, queryTokenFactory, validateApiKey } = require("./lib/client");
const { buildViewModel } = require("./lib/quota");

const DEFAULT_BASE_URL = "https://tokease.com";
const APP_ID = "com.tokenfactory.quota-widget";
const LOGO_PATH = path.join(__dirname, "assets", "logo.png");

let mainWindow = null;
let tray = null;
let quitting = false;
let queryInProgress = false;
let refreshTimer = null;
let refreshInterval = 30000;
let currentView = "balance";
let runtimeConfig = null;
let lastData = null;
let dragState = null;

function safeMessage(error) {
  return error && typeof error.message === "string" ? error.message.slice(0, 160) : "查询失败";
}

function startupPath() {
  return process.env.PORTABLE_EXECUTABLE_FILE || process.execPath;
}

function loginSettings() {
  const pathValue = startupPath();
  const args = app.isPackaged ? [] : [path.resolve(__dirname, "main.js")];
  return { path: pathValue, args };
}

function send(channel, payload) {
  if (mainWindow && !mainWindow.isDestroyed()) mainWindow.webContents.send(channel, payload);
}

function scheduleRefresh() {
  if (refreshTimer) clearInterval(refreshTimer);
  refreshTimer = null;
  if (!runtimeConfig || !refreshInterval) return;
  refreshTimer = setInterval(() => refreshData(true), refreshInterval);
}

async function refreshData(silent = false) {
  if (!runtimeConfig || queryInProgress) return { success: false, message: "尚未配置" };
  queryInProgress = true;
  try {
    const result = await queryTokenFactory(runtimeConfig);
    lastData = buildViewModel(result);
    send("data:update", lastData);
    return { success: true, data: lastData };
  } catch (error) {
    if (!silent) send("query:error", safeMessage(error));
    return { success: false, message: safeMessage(error) };
  } finally {
    queryInProgress = false;
  }
}

function intervalMenuItem(label, value) {
  return {
    label,
    type: "radio",
    checked: refreshInterval === value,
    click: () => {
      refreshInterval = value;
      scheduleRefresh();
    }
  };
}

function buildMenu() {
  const login = loginSettings();
  return Menu.buildFromTemplate([
    {
      label: "余额",
      type: "radio",
      enabled: Boolean(runtimeConfig),
      checked: currentView === "balance",
      click: () => {
        currentView = "balance";
        send("menu:action", { type: "view", value: "balance" });
      }
    },
    {
      label: "最近 10 条消费",
      type: "radio",
      enabled: Boolean(runtimeConfig),
      checked: currentView === "logs",
      click: () => {
        currentView = "logs";
        send("menu:action", { type: "view", value: "logs" });
      }
    },
    { type: "separator" },
    { label: "立即刷新", enabled: Boolean(runtimeConfig), click: () => refreshData(false) },
    {
      label: "自动刷新",
      submenu: [
        intervalMenuItem("关闭", 0),
        intervalMenuItem("10 秒", 10000),
        intervalMenuItem("30 秒", 30000),
        intervalMenuItem("1 分钟", 60000),
        intervalMenuItem("5 分钟", 300000)
      ]
    },
    { type: "separator" },
    {
      label: "始终置顶",
      type: "checkbox",
      checked: mainWindow?.isAlwaysOnTop() ?? true,
      click: (item) => mainWindow?.setAlwaysOnTop(item.checked, "floating")
    },
    {
      label: "开机启动",
      type: "checkbox",
      checked: app.getLoginItemSettings(login).openAtLogin,
      click: (item) => app.setLoginItemSettings({ ...login, openAtLogin: item.checked })
    },
    {
      label: "重新填写配置",
      click: () => {
        runtimeConfig = null;
        lastData = null;
        currentView = "balance";
        scheduleRefresh();
        send("menu:action", { type: "reset" });
        mainWindow?.show();
      }
    },
    { label: "隐藏", click: () => mainWindow?.hide() },
    { type: "separator" },
    {
      label: "退出",
      click: () => {
        quitting = true;
        app.quit();
      }
    }
  ]);
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 286,
    height: 118,
    frame: false,
    transparent: true,
    resizable: false,
    maximizable: false,
    alwaysOnTop: true,
    skipTaskbar: true,
    hasShadow: false,
    backgroundColor: "#00000000",
    title: "TokenFactory额度悬浮窗",
    icon: LOGO_PATH,
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true
    }
  });

  mainWindow.setAlwaysOnTop(true, "floating");
  mainWindow.loadFile(path.join(__dirname, "renderer", "index.html"));
  mainWindow.on("close", (event) => {
    if (!quitting) {
      event.preventDefault();
      mainWindow.hide();
    }
  });
  mainWindow.on("closed", () => { mainWindow = null; });
  mainWindow.webContents.on("context-menu", () => buildMenu().popup({ window: mainWindow }));
  mainWindow.webContents.setWindowOpenHandler(() => ({ action: "deny" }));
  mainWindow.webContents.on("will-navigate", (event, url) => {
    if (!url.startsWith("file://")) event.preventDefault();
  });
}

function createTray() {
  const trayIcon = nativeImage.createFromPath(LOGO_PATH).resize({ width: 32, height: 32, quality: "best" });
  tray = new Tray(trayIcon);
  tray.setToolTip("TokenFactory额度悬浮窗");
  tray.on("click", () => {
    if (!mainWindow) createWindow();
    if (mainWindow.isVisible()) mainWindow.hide();
    else {
      mainWindow.show();
      mainWindow.focus();
    }
  });
  tray.on("right-click", () => buildMenu().popup());
}

ipcMain.handle("config:submit", async (_event, input) => {
  if (queryInProgress) return { success: false, message: "正在查询" };
  try {
    const candidate = {
      baseUrl: normalizeBaseUrl(input?.baseUrl || DEFAULT_BASE_URL),
      apiKey: validateApiKey(input?.apiKey)
    };
    queryInProgress = true;
    const result = await queryTokenFactory(candidate);
    const data = buildViewModel(result);
    runtimeConfig = candidate;
    lastData = data;
    scheduleRefresh();
    return { success: true, data };
  } catch (error) {
    return { success: false, message: safeMessage(error) };
  } finally {
    queryInProgress = false;
  }
});

ipcMain.handle("state:get", () => ({
  configured: Boolean(runtimeConfig),
  data: lastData,
  defaultBaseUrl: DEFAULT_BASE_URL
}));

ipcMain.handle("query:refresh", () => refreshData(false));

ipcMain.on("view:set", (_event, view) => {
  currentView = view === "logs" ? "logs" : "balance";
});

ipcMain.on("window:drag-start", (_event, cursor) => {
  if (!mainWindow || !cursor) return;
  dragState = {
    cursorX: Number(cursor.x || 0),
    cursorY: Number(cursor.y || 0),
    bounds: mainWindow.getBounds()
  };
});

ipcMain.on("window:drag-to", (_event, cursor) => {
  if (!mainWindow || !dragState || !cursor) return;
  const x = Math.round(dragState.bounds.x + Number(cursor.x || 0) - dragState.cursorX);
  const y = Math.round(dragState.bounds.y + Number(cursor.y || 0) - dragState.cursorY);
  mainWindow.setBounds({
    x,
    y,
    width: dragState.bounds.width,
    height: dragState.bounds.height
  }, false);
});

ipcMain.on("window:drag-end", () => { dragState = null; });

ipcMain.on("window:resize", (_event, requested) => {
  if (!mainWindow || !requested || dragState) return;
  const width = Math.min(360, Math.max(68, Math.round(Number(requested.width) || 0)));
  const height = Math.min(420, Math.max(42, Math.round(Number(requested.height) || 0)));
  const [currentX, currentY] = mainWindow.getPosition();
  const display = screen.getDisplayMatching({ x: currentX, y: currentY, width, height });
  const area = display.workArea;
  const x = Math.min(Math.max(currentX, area.x), area.x + area.width - width);
  const y = Math.min(Math.max(currentY, area.y), area.y + area.height - height);
  mainWindow.setBounds({ x, y, width, height }, false);
});

const singleInstanceLock = app.requestSingleInstanceLock();
if (!singleInstanceLock) {
  app.quit();
} else {
  app.setAppUserModelId(APP_ID);
  app.on("second-instance", () => {
    mainWindow?.show();
    mainWindow?.focus();
  });
  app.whenReady().then(() => {
    createWindow();
    createTray();
  });
}

app.on("before-quit", () => {
  quitting = true;
  dragState = null;
  if (refreshTimer) clearInterval(refreshTimer);
});

app.on("window-all-closed", () => {});
