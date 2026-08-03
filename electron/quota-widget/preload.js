const { contextBridge, ipcRenderer } = require("electron");

function subscribe(channel, callback) {
  const listener = (_event, payload) => callback(payload);
  ipcRenderer.on(channel, listener);
  return () => ipcRenderer.removeListener(channel, listener);
}

contextBridge.exposeInMainWorld("quotaWidget", {
  configure: (baseUrl, apiKey) => ipcRenderer.invoke("config:submit", { baseUrl, apiKey }),
  getState: () => ipcRenderer.invoke("state:get"),
  refresh: () => ipcRenderer.invoke("query:refresh"),
  startDrag: (x, y) => ipcRenderer.send("window:drag-start", { x, y }),
  dragTo: (x, y) => ipcRenderer.send("window:drag-to", { x, y }),
  endDrag: () => ipcRenderer.send("window:drag-end"),
  resize: (width, height) => ipcRenderer.send("window:resize", { width, height }),
  setView: (view) => ipcRenderer.send("view:set", view),
  onData: (callback) => subscribe("data:update", callback),
  onMenuAction: (callback) => subscribe("menu:action", callback),
  onQueryError: (callback) => subscribe("query:error", callback)
});
