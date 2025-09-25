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
let ffmpegPath = null;
try {
    // Resolve ffmpeg path. In dev, use ffmpeg-static require; in packaged, use resources copy
    if (process.env.ELECTRON_RUN_AS_NODE || !app) {
        ffmpegPath = require('ffmpeg-static');
    }
} catch (_) {}

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

// Python runner removed; keeping stub for compatibility
function resolvePythonCmd() { return 'python3'; }

// STT: Transcription via WhisperX runner
ipcMain.handle('stt:transcribe', async (_e, payload) => {
    try {
        const {
            audioBuffer,
            extension = 'webm',
            model = settings?.sttModel || 'small',
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

        // Apple Speech removed as requested; use Whisper below
        // Fallback: local OpenAI Whisper via Python (CPU)
        try {
            const vendorSite = app.isPackaged
                ? path.join(process.resourcesPath, 'python', 'site')
                : path.join(app.getAppPath(), 'python', 'site');
            const pythonCmd = '/usr/bin/python3';
            const env = { ...process.env };
            // Ensure vendored site-packages are discoverable
            env.PYTHONPATH = env.PYTHONPATH ? `${vendorSite}:${env.PYTHONPATH}` : vendorSite;
            env.PYTHONHOME = '';
            // Make sure ffmpeg is available for whisper decoding
            let localFfmpeg = ffmpegPath;
            if (!localFfmpeg && app.isPackaged) {
                // Use resources ffmpeg binary; ffmpeg-static structure differs per platform
                const resFfmpegDir = path.join(process.resourcesPath, 'ffmpeg');
                try {
                    const entries = fs.readdirSync(resFfmpegDir);
                    const bin = entries.find((f) => f.includes('ffmpeg'));
                    if (bin) localFfmpeg = path.join(resFfmpegDir, bin);
                } catch (_) {}
            }
            if (localFfmpeg && fs.existsSync(localFfmpeg)) {
                env.PATH = `${path.dirname(localFfmpeg)}:${env.PATH || ''}`;
            }

            // Pre-convert to WAV (16k mono) to avoid ffmpeg dependency inside whisper
            let inputForWhisper = srcPath;
            if (localFfmpeg && fs.existsSync(localFfmpeg)) {
                try {
                    const wavPath = path.join(app.getPath('temp'), `cypher_stt_${Date.now()}.wav`);
                    await new Promise((resolve, reject) => {
                        const ff = spawn(localFfmpeg, ['-y', '-i', srcPath, '-ac', '1', '-ar', '16000', wavPath]);
                        ff.on('error', reject);
                        ff.on('close', (code) => code === 0 ? resolve() : reject(new Error(`ffmpeg exit ${code}`)));
                    });
                    inputForWhisper = wavPath;
                } catch (_) { /* continue with original file */ }
            }
            const whisperRunner = `import sys, json\n\ntry:\n    import whisper\nexcept Exception as e:\n    print(json.dumps({\"error\": f'whisper import failed: {e}'}))\n    sys.exit(0)\n\nmodel_name='small'\nif len(sys.argv) > 2 and sys.argv[2]:\n    model_name=sys.argv[2]\nlang=None\nif len(sys.argv) > 3 and sys.argv[3]:\n    lang=sys.argv[3]\ntry:\n    m = whisper.load_model(model_name, device='cpu')\n    audio = sys.argv[1]\n    kwargs = {}\n    if lang: kwargs['language'] = lang\n    r = m.transcribe(audio, **kwargs)\n    print(json.dumps({\"text\": r.get('text','').strip()}))\nexcept Exception as e:\n    print(json.dumps({\"error\": str(e)}))\n`;
            const runnerPath = path.join(app.getPath('temp'), `whisper_runner_${Date.now()}.py`);
            fs.writeFileSync(runnerPath, whisperRunner);
            const chosenModel = settings?.sttModel || 'small';
            const langArg = settings?.sttLanguage || '';
            const args = [runnerPath, inputForWhisper, chosenModel, langArg];
            const sp = spawn(pythonCmd, args, { env });
            activeSttProcess = sp;
            let out = '';
            let err = '';
            sp.stdout.on('data', (d) => { out += d.toString(); });
            sp.stderr.on('data', (d) => { err += d.toString(); });
            const code = await new Promise((r) => sp.on('close', r));
            if (activeSttProcess === sp) activeSttProcess = null;
            try { fs.unlinkSync(srcPath); } catch (_) {}
            try { fs.unlinkSync(runnerPath); } catch (_) {}
            if (out.trim()) {
                try { return JSON.parse(out.trim()); } catch (_) { return { text: out.trim() }; }
            }
            return { error: err || 'whisper failed' };
        } catch (e) {
            return { error: String(e?.message || e) };
        }
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
