// console.log("screenshot.js")

const screenshotImageId = "screenshot-img"

function setScreenshotUrl(imageUri) {
    document.getElementById(screenshotImageId).src = imageUri;
}

function getScreenshotUrl() {
    return document.getElementById(screenshotImageId).src;
}
