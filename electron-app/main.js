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

let mainWindow;
let goProcess;

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

app.whenReady().then(createWindow);

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
