
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'displayScreenshot' && message.imageUri) {
    document.getElementById("screenshot-img").src = message.imageUri;
  }
});
