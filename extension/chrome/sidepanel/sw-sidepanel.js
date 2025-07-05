// console.log('sw-sidepanel.js')

chrome.sidePanel
  .setPanelBehavior({ openPanelOnActionClick: true })
  .catch((error) => console.error(error));

const nonCapturables = ['chrome:', 'about:', 'blob:', 'chrome-extension:'];

chrome.runtime.onMessage.addListener((message, _, sendResponse) => {
  if (message.action === 'capture-screenshot') {
    chrome.tabs.query({ active: true, currentWindow: true }).then((tabs) => {
      if (!tabs || tabs.length === 0) {
        sendResponse({ error: 'No active tab' });
        return;
      }

      const tab = tabs[0];
      if (tab.url && nonCapturables.some(proto => tab.url.startsWith(proto))) {
        sendResponse({ error: 'This page cannot be captured' });
        return;
      }

      chrome.tabs.captureVisibleTab(tab.windowId, { format: 'png' }).then((imageUri) => {
        console.log("imageUri:", imageUri ? imageUri.substring(0, 100) + "..." : null);
        sendResponse({ success: true, data: imageUri });
      }).catch((error) => {
        console.error('Error capturing screenshot:', error);
        sendResponse({ error: error });
      });
    }).catch((e) => {
      sendResponse({ error: e.message });
    });
    return true; // async
  }
});

chrome.tabs.onActivated.addListener(async (activeInfo) => {
  const tab = await chrome.tabs.get(activeInfo.tabId);
  chrome.runtime.sendMessage({
    action: "tab-switched",
    url: tab.url
  });
});

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === "complete" && tab.active) {
    chrome.runtime.sendMessage({
      action: "tab-switched",
      url: tab.url
    });
  }
});
