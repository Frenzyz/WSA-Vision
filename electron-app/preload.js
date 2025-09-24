const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('cypher', {
  getSettings: () => ipcRenderer.invoke('settings:get'),
  setSettings: (partial) => ipcRenderer.invoke('settings:set', partial),
  toggleApp: () => ipcRenderer.invoke('app:toggle'),
  onSettingsUpdated: (cb) => ipcRenderer.on('settings-updated', (_e, s) => cb(s)),
  onFocusRequested: (cb) => ipcRenderer.on('focus-input', () => cb()),
  // STT bridge
  onSttToggle: (cb) => ipcRenderer.on('stt:toggle', () => cb()),
  transcribeAudio: (payload) => ipcRenderer.invoke('stt:transcribe', payload),
  abortTranscription: () => ipcRenderer.invoke('stt:abort'),
});
// Preload script for Electron app