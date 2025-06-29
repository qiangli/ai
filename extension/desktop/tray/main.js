const { app, BrowserWindow, Tray, Menu, ipcMain, desktopCapturer, nativeImage } = require('electron');
const path = require('path');

let tray = null;
let win = null;
let pinnedToTray = true;

function createWindow() {
  win = new BrowserWindow({
    width: 400,
    height: 600,
    show: false,
    frame: false,
    transparent: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: true,
      contextIsolation: false,
    },
  });
  win.loadFile('index.html');
  
  win.on('close', (e) => {
    if (!app.isQuiting) {
      e.preventDefault();
      win.hide();
    }
    return false;
  });
}

app.whenReady().then(() => {
  tray = new Tray(path.join(__dirname, 'trayIcon.png'));
  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Quit',
      click: () => {
        app.isQuiting = true;
        app.quit();
      },
    },
  ]);
  tray.setToolTip('AI Tray');
  tray.setContextMenu(contextMenu);

  tray.on('click', () => {
    if (win.isVisible()) {
      win.hide();
    } else {
      win.show();
      win.focus();
    }
  });

  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

ipcMain.on('toggle-pin', () => {
  pinnedToTray = !pinnedToTray;
  if (pinnedToTray) {
    tray = new Tray(path.join(__dirname, 'trayIcon.png'));
    tray.setToolTip('AI Tray');
    tray.setContextMenu(Menu.buildFromTemplate([
      {
        label: 'Quit',
        click: () => {
          app.isQuiting = true;
          app.quit();
        },
      },
    ]));
    tray.on('click', () => {
      if (win.isVisible()) {
        win.hide();
      } else {
        win.show();
        win.focus();
      }
    });
  } else if (tray) {
    tray.destroy();
    tray = null;
  }
  // Notify renderer about pin status
  win.webContents.send('pin-status', pinnedToTray);
});

ipcMain.on('close-window', () => {
  if (win) {
    win.hide();
  }
});

ipcMain.handle('take-screenshot', async () => {
  const sources = await desktopCapturer.getSources({ types: ['screen'] });
  const screenSource = sources[0];
  return screenSource.thumbnail.toDataURL();
});
