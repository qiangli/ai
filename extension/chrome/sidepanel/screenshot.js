// console.log("screenshot.js")

const screenshotImageId = "screenshot-img"
const screenshot = document.body.querySelector('#screenshot');
const buttonScreenshot = document.body.querySelector('#button-screenshot');

function setScreenshotUrl(imageUri) {
    document.getElementById(screenshotImageId).src = imageUri;
}

function getScreenshotUrl() {
    return document.getElementById(screenshotImageId).src;
}


buttonScreenshot.addEventListener('click', () => {
    chrome.runtime.sendMessage({ action: 'capture-screenshot' }, (response) => {
        if (chrome.runtime.lastError) {
            console.error(chrome.runtime.lastError.message);
            showError(chrome.runtime.lastError.message);
            return;
        }
        if (response && response.success) {
            show(screenshot);
            setScreenshotUrl(response.data);

            // download link
            screenshot.addEventListener('click', () => {
                const downloadLink = document.createElement('a');
                downloadLink.href = response.data;

                const timestamp = new Date().toISOString().replace(/[:\.]/g, '-');
                const filename = `chrome-screenshot-${timestamp}.png`
                downloadLink.download = filename;

                downloadLink.click();
                showResponse('screenshot saved as ' + filename);
            }, { once: true });
        } else {
            console.error('Screenshot capture failed', response && response.error);
            if (response) {
                showError(response.error);
            }
        }
    });
});
