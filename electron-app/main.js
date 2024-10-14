const { app, BrowserWindow } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const fs = require('fs');

let mainWindow;
let goProcess;

function createWindow() {
    mainWindow = new BrowserWindow({
        width: 800,
        height: 600,
        webPreferences: {
            preload: path.join(__dirname, 'preload.js'),
        },
    });

    mainWindow.loadFile('index.html');

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

function startGoBackend() {
    const isWindows = process.platform === 'win32';
    const goExecutableName = isWindows ? 'cypher_backend' : 'cypher_backend';

    // Adjust the path to the Go executable
    const goAppPath = path.join(app.getAppPath(), 'backend/' ,goExecutableName);

    // Check if the Go executable exists
    if (!fs.existsSync(goAppPath)) {
        console.error(`Go executable not found at ${goAppPath}`);
        app.quit();
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
