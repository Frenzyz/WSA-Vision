let app, BrowserWindow;
try {
    ({ app, BrowserWindow } = require('electron'));
} catch (e) {}
if (!app) {
    try {
        ({ app, BrowserWindow } = require('electron/main'));
    } catch (e) {}
}
if (!app) {
    console.error('Electron app module not available. Make sure you are launching with Electron.');
    process.exit(1);
}
const path = require('path');
const { spawn } = require('child_process');
const fs = require('fs');
const { globalShortcut, ipcMain } = require('electron');

let mainWindow;
let goProcess;
let settings = null;
let shortcutRegistered = null;
let sttShortcutRegistered = null;
let activeSttProcess = null;

function getSettingsPath() {
    const userData = app.getPath('userData');
    return path.join(userData, 'settings.json');
}

function loadSettings() {
    try {
        const p = getSettingsPath();
        if (fs.existsSync(p)) {
            return JSON.parse(fs.readFileSync(p, 'utf-8'));
        }
    } catch (e) {
        console.warn('Failed to load settings:', e);
    }
    // Default accelerator per platform
    const def = process.platform === 'darwin' ? 'Alt+C' : 'Alt+C';
    return { 
        toggleShortcut: def,
        sttShortcut: 'Shift+S',
        sttPythonPath: '',
        sttModel: 'small',
        sttDevice: 'cpu',
        sttComputeType: 'int8', // float32|float16|int8
        sttLanguage: '',
        sttBatchSize: 8,
        autoExecuteAfterSTT: true
    };
}

function saveSettings(newSettings) {
    settings = { ...settings, ...newSettings };
    try { fs.writeFileSync(getSettingsPath(), JSON.stringify(settings, null, 2)); } catch (e) {}
    registerToggleShortcut(settings.toggleShortcut);
    if (mainWindow) mainWindow.webContents.send('settings-updated', settings);
}

function registerToggleShortcut(accelerator) {
    try {
        if (shortcutRegistered) globalShortcut.unregister(shortcutRegistered);
        if (accelerator) {
            const ok = globalShortcut.register(accelerator, () => {
                if (process.platform === 'darwin') {
                    if (app.isHidden()) {
                        app.show();
                        if (mainWindow) { mainWindow.show(); mainWindow.focus(); }
                    } else {
                        app.hide();
                    }
                } else if (mainWindow) {
                    if (mainWindow.isVisible()) {
                        if (mainWindow.isMinimized()) {
                            mainWindow.restore();
                        }
                        mainWindow.minimize();
                    } else {
                        mainWindow.restore();
                        mainWindow.show();
                        mainWindow.focus();
                    }
                }
                if (mainWindow) mainWindow.webContents.send('focus-input');
            });
            if (ok) shortcutRegistered = accelerator;
        }
    } catch (e) { console.warn('Shortcut registration failed', e); }
}

function registerSTTShortcut(accelerator) {
    try {
        if (sttShortcutRegistered) globalShortcut.unregister(sttShortcutRegistered);
        if (accelerator) {
            const ok = globalShortcut.register(accelerator, () => {
                if (mainWindow) {
                    mainWindow.webContents.send('stt:toggle');
                }
            });
            if (ok) sttShortcutRegistered = accelerator;
        }
    } catch (e) { console.warn('STT Shortcut registration failed', e); }
}

async function isBackendRunning() {
    try {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), 1000);
        const res = await fetch('http://localhost:8080/models', { signal: controller.signal });
        clearTimeout(timeout);
        if (res && res.ok) {
            console.log('Detected running backend on http://localhost:8080. Skipping spawn.');
            return true;
        }
    } catch (e) {
        // not running
    }
    return false;
}

function createWindow() {
    const iconPath = path.join(__dirname, 'public', 'WSA_Icon.icns');
    console.log('Icon path:', iconPath);
    console.log('Icon exists:', fs.existsSync(iconPath));
    console.log('Current directory:', __dirname);
    console.log('Files in public:', fs.readdirSync(path.join(__dirname, 'public')));
    
    mainWindow = new BrowserWindow({
        width: 1000,
        height: 800,
        minWidth: 800,
        minHeight: 600,
        transparent: true,
        titleBarStyle: 'customButtonsOnHover',
        vibrancy: 'ultra-dark',
        icon: iconPath,
        webPreferences: {
            preload: path.join(__dirname, 'preload.js'),
            nodeIntegration: false,
            contextIsolation: true,
        },
        show: false,
    });

    mainWindow.loadFile('index.html');

    // Show window with fade-in effect
    mainWindow.once('ready-to-show', () => {
        mainWindow.show();
    });

    // Start the Go backend
    startGoBackend();

    mainWindow.on('closed', function () {
        mainWindow = null;
        stopGoBackend();
    });
}

app.whenReady().then(() => {
    settings = loadSettings();
    createWindow();
    registerToggleShortcut(settings.toggleShortcut);
    registerSTTShortcut(settings.sttShortcut || 'Shift+S');
});

app.on('window-all-closed', function () {
    stopGoBackend();
    if (process.platform !== 'darwin') {
        app.quit();
    }
});

async function startGoBackend() {
    // If a backend is already running on 8080, do not spawn another
    if (await isBackendRunning()) {
        return;
    }
    const isWindows = process.platform === 'win32';
    const goExecutableName = isWindows ? 'cypher_backend' : 'cypher_backend';

    // Determine the correct path for the Go executable
    let goAppPath;
    if (app.isPackaged) {
        // In packaged app, look in extraResources
        goAppPath = path.join(process.resourcesPath, 'backend', goExecutableName);
    } else {
        // In development, look in the backend folder
        goAppPath = path.join(app.getAppPath(), 'backend/', goExecutableName);
    }

    console.log('Looking for Go executable at:', goAppPath);
    console.log('App is packaged:', app.isPackaged);
    console.log('Resources path:', process.resourcesPath);

    // Check if the Go executable exists
    if (!fs.existsSync(goAppPath)) {
        console.warn(`Go executable not found at ${goAppPath}`);
        console.warn('App will run without backend functionality');
        try {
            const resourcesDir = process.resourcesPath || app.getAppPath();
            console.log('Available files in resources:', fs.readdirSync(resourcesDir));
            if (fs.existsSync(path.join(resourcesDir, 'backend'))) {
                console.log('Files in backend dir:', fs.readdirSync(path.join(resourcesDir, 'backend')));
            }
        } catch (e) {
            console.error('Error listing files:', e);
        }
        return;
    }

    // Make the Go executable (for Unix-like systems)
    if (!isWindows) {
        fs.chmodSync(goAppPath, '755');
    }

    // Start the Go process
    goProcess = spawn(goAppPath, [], {
        cwd: app.getAppPath(),
    });

    goProcess.stdout.on('data', (data) => {
        console.log(`Go Backend: ${data}`);
    });

    goProcess.stderr.on('data', (data) => {
        console.error(`Go Backend Error: ${data}`);
    });

    goProcess.on('close', (code) => {
        console.log(`Go Backend exited with code ${code}`);
        if (mainWindow) {
            mainWindow.webContents.send('go-backend-exited', code);
        }
    });
}

function stopGoBackend() {
    if (goProcess) {
        goProcess.kill();
    }
}

// IPC for settings
ipcMain.handle('settings:get', async () => {
    if (!settings) settings = loadSettings();
    return settings;
});

ipcMain.handle('settings:set', async (_e, partial) => {
    saveSettings(partial || {});
    if (partial && typeof partial.sttShortcut === 'string') {
        registerSTTShortcut(partial.sttShortcut);
    }
    return settings;
});

ipcMain.handle('app:toggle', () => {
    if (process.platform === 'darwin') {
        if (app.isHidden()) {
            app.show();
            if (mainWindow) { mainWindow.show(); mainWindow.focus(); }
        } else {
            app.hide();
        }
    } else if (mainWindow) {
        if (mainWindow.isVisible()) {
            if (mainWindow.isMinimized()) mainWindow.restore();
            mainWindow.minimize();
        } else {
            mainWindow.restore();
            mainWindow.show();
            mainWindow.focus();
        }
    }
    if (mainWindow) mainWindow.webContents.send('focus-input');
});

function resolvePythonCmd() {
    // 1) settings override
    if (settings && settings.sttPythonPath && fs.existsSync(settings.sttPythonPath)) {
        return settings.sttPythonPath;
    }
    // 2) bundled venv (packaged)
    if (app.isPackaged) {
        const venvPython = path.join(process.resourcesPath, 'python', 'venv', 'bin', 'python');
        if (fs.existsSync(venvPython)) return venvPython;
    } else {
        const devVenv = path.join(app.getAppPath(), 'python', 'venv', 'bin', 'python');
        if (fs.existsSync(devVenv)) return devVenv;
    }
    // 3) common Homebrew path (macOS ARM)
    const candidates = [
        '/opt/homebrew/bin/python3',
        '/usr/local/bin/python3',
        process.platform === 'darwin' ? '/usr/bin/python3' : 'python3'
    ];
    for (const c of candidates) {
        try { if (fs.existsSync(c)) return c; } catch (_) {}
    }
    return 'python3';
}

// STT: Transcription via WhisperX runner
ipcMain.handle('stt:transcribe', async (_e, payload) => {
    try {
        const {
            audioBuffer,
            extension = 'webm',
            model = settings?.sttModel || 'large-v2',
            device = settings?.sttDevice || 'cpu',
            computeType = settings?.sttComputeType || 'int8'
        } = payload || {};
        if (!audioBuffer || !audioBuffer.data) {
            throw new Error('Missing audio buffer');
        }

        const tmpDir = app.getPath('temp');
        const srcPath = path.join(tmpDir, `cypher_stt_${Date.now()}.${extension}`);
        const buf = Buffer.from(audioBuffer.data);
        fs.writeFileSync(srcPath, buf);

        // Resolve python runner path
        let runnerPath;
        if (app.isPackaged) {
            // Prefer unpacked copy
            const unpacked = path.join(process.resourcesPath, 'app.asar.unpacked', 'whisperx_runner.py');
            const direct = path.join(process.resourcesPath, 'whisperx_runner.py');
            runnerPath = fs.existsSync(unpacked) ? unpacked : direct;
        } else {
            runnerPath = path.join(app.getAppPath(), 'whisperx_runner.py');
        }

        if (!fs.existsSync(runnerPath)) {
            throw new Error(`WhisperX runner not found at ${runnerPath}`);
        }

        const pythonCmd = resolvePythonCmd();
        const env = { ...process.env };
        // Prefer system SSL (OpenSSL) over Apple's LibreSSL to avoid urllib3 warnings
        // Homebrew OpenSSL on macOS (arm64)
        const brewPrefix = '/opt/homebrew';
        if (process.platform === 'darwin' && fs.existsSync(brewPrefix)) {
            const opensslDir = path.join(brewPrefix, 'opt', 'openssl@3');
            const libPath = path.join(opensslDir, 'lib');
            env.LD_LIBRARY_PATH = env.LD_LIBRARY_PATH ? `${libPath}:${env.LD_LIBRARY_PATH}` : libPath;
            env.DYLD_LIBRARY_PATH = env.DYLD_LIBRARY_PATH ? `${libPath}:${env.DYLD_LIBRARY_PATH}` : libPath;
        }

        const argsArr = [runnerPath, srcPath, '--model', model, '--device', device, '--compute_type', computeType, '--no_align'];
        if (Number.isFinite(settings?.sttBatchSize)) {
            argsArr.push('--batch_size', String(settings.sttBatchSize));
        }
        if (settings?.sttLanguage) {
            argsArr.push('--language', settings.sttLanguage);
        }
        const py = spawn(pythonCmd, argsArr, { env });
        activeSttProcess = py;

        let stdout = '';
        let stderr = '';
        py.stdout.on('data', (d) => { stdout += d.toString(); });
        py.stderr.on('data', (d) => { stderr += d.toString(); });

        const code = await new Promise((resolve) => py.on('close', resolve));
        if (activeSttProcess === py) activeSttProcess = null;

        try { fs.unlinkSync(srcPath); } catch (_) {}

        function parseJsonFromStdout(output) {
            const trimmed = (output || '').trim();
            const lines = trimmed.split(/\r?\n/);
            for (let i = lines.length - 1; i >= 0; i--) {
                const line = lines[i].trim();
                if (!line) continue;
                try { return JSON.parse(line); } catch (_) {}
            }
            const start = trimmed.lastIndexOf('{');
            const end = trimmed.lastIndexOf('}');
            if (start !== -1 && end !== -1 && end > start) {
                try { return JSON.parse(trimmed.slice(start, end + 1)); } catch (_) {}
            }
            return null;
        }

        if (code !== 0) {
            const maybe = parseJsonFromStdout(stdout);
            if (maybe) return maybe;
            return { error: (stderr || `WhisperX runner exited with code ${code}`) };
        }

        const parsed = parseJsonFromStdout(stdout) || { text: '', error: 'Failed to parse STT output' };
        return parsed;
    } catch (err) {
        console.error('STT transcription failed:', err);
        return { error: String(err?.message || err) };
    }
});

ipcMain.handle('stt:abort', async () => {
    try {
        if (activeSttProcess && !activeSttProcess.killed) {
            activeSttProcess.kill('SIGTERM');
        }
    } catch (e) {}
});
