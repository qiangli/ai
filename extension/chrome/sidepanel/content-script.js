//

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === "get-selection") {
    sendResponse({ text: window.getSelection().toString() });
  }
});