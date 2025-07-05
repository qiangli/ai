//
// https://developer.chrome.com/docs/extensions/reference/api/offscreen/
async function setupOffscreen(path) {

  try {
    const existingContexts = await chrome.runtime.getContexts({});

    const offscreenDocument = existingContexts.find(
      (c) => c.contextType === 'OFFSCREEN_DOCUMENT'
    );

    // If an offscreen document is not already open, create one.
    if (offscreenDocument) {
      return offscreenDocument.documentUrl;
    }
    // Create an offscreen document.
    await chrome.offscreen.createDocument({
      url: 'offscreen.html',
      reasons: ['USER_MEDIA'],
      justification: 'Recording from chrome.tabCapture API'
    });


  } catch (e) {
    console.error(e);
  }
  return "";
}

chrome.action.onClicked.addListener(async (tab) => {
  const docUrl = await setupOffscreen("offscreen.html");
  const recording = docUrl.endsWith('#recording');

  if (recording) {
    chrome.runtime.sendMessage({
      action: 'stop-recording',
      target: 'offscreen'
    });
    chrome.action.setIcon({ path: '/images/recording-off.png' });
    return;
  }

  const streamId = await chrome.tabCapture.getMediaStreamId({
    targetTabId: tab.id
  });

  chrome.runtime.sendMessage({
    action: 'start-recording',
    target: 'offscreen',
    data: streamId
  });

  chrome.action.setIcon({ path: '/images/recording-on.png' });
});
