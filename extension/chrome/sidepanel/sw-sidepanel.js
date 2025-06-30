// console.log('sw-sidepanel.js')

chrome.sidePanel
    .setPanelBehavior({ openPanelOnActionClick: true })
    .catch((error) => console.error(error));

chrome.runtime.onMessage.addListener((message, _, sendResponse) => {
    if (message.action === 'capture-screenshot') {
        chrome.tabs.query({ active: true, currentWindow: true }).then((tabs) => {
            if (tabs.length === 0) {
                sendResponse({ error: 'No active tab' });
                return true;
            }
            const tab = tabs[0];
            chrome.tabs.captureVisibleTab(tab.windowId, { format: 'png' }).then((imageUri) => {
                console.log("imageUri:", imageUri ? imageUri.substring(0, 100) + "..." : null);
                sendResponse({ success: true, data: imageUri });
            }).catch((error) => {
                console.error('Error capturing screenshot:', error);
                sendResponse({ error: error });
            });
        }).catch((error) => {
            sendResponse({ error: error.message });
        });
        return true;
    }
});

chrome.tabs.onActivated.addListener(async (activeInfo) => {
    const tab = await chrome.tabs.get(activeInfo.tabId);
    chrome.runtime.sendMessage({
        type: "tab-switched",
        url: tab.url
    });
});

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    if (changeInfo.status === "complete" && tab.active) {
        chrome.runtime.sendMessage({
            type: "tab-switched",
            url: tab.url
        });
    }
});