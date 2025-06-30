// console.log('sidepanel.js');

function setHubIcon(active) {
    const icon = document.getElementById('hub-icon');
    icon.src = active ? 'images/socket-active.png' : 'images/socket-inactive.png';
}

// On load: query hub state and update icon
document.addEventListener('DOMContentLoaded', () => {
    chrome.runtime.sendMessage({ action: "get-hub-status" }, function (response) {
        if (response && typeof response.active === "boolean") {
            setHubIcon(response.active);
        }
    });

    // init selector
    chrome.tabs.query({ active: true, lastFocusedWindow: true }, (tabs) => {
        const url = tabs[0]?.url || '';
        updateSelector(url);
    });
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === "tab-switched") {
        const url = message.url || '';
        updateSelector(url);
    }
});

function updateSelector(url) {
    if (!url) {
        return
    }
    const selectorValue = getDefaultSelector(url, "body");
    document.getElementById('input-selector').value = selectorValue;
}

document.getElementById('toggle-hub').addEventListener('click', () => {
    chrome.runtime.sendMessage({ action: "toggle-hub" }, function (response) {
        if (response && typeof response.active === "boolean") {
            setHubIcon(response.active);
        }
    });
});


const inputSelector = document.body.querySelector('#input-selector');
const buttonGetContent = document.body.querySelector('#button-get-content');
const screenshot = document.body.querySelector('#screenshot');
const buttonScreenshot = document.body.querySelector('#button-screenshot');

const inputPrompt = document.body.querySelector('#input-prompt');
const buttonSend = document.body.querySelector('#button-send');
const buttonCopy = document.body.querySelector('#button-copy');
const buttonReset = document.body.querySelector('#button-reset');

const elementResponse = document.body.querySelector('#response');
const elementLoading = document.body.querySelector('#loading');
const elementError = document.body.querySelector('#error');

buttonGetContent.addEventListener('click', async () => {
    try {
        const value = inputSelector.value || 'body';
        const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

        let content;

        // check if any selected text
        await chrome.scripting.executeScript({
            target: { tabId: tab.id },
            files: ["content-script.js"]
        });
        const response = await chrome.tabs.sendMessage(tab.id, { action: 'get-selection' });
        content = response?.text ?? "";

        if (!content) {
            const [{ result: textArray }] = await chrome.scripting.executeScript({
                target: { tabId: tab.id },
                func: (selector) => {
                    function getTexts(doc) {
                        let result = [];
                        try {
                            // Get matching elements in this doc
                            result = Array.from(doc.querySelectorAll(selector))
                                .map(node => node.innerText)
                                .filter(Boolean);
                            // Recursively check iframes (same-origin only)
                            for (const iframe of Array.from(doc.querySelectorAll('iframe'))) {
                                try {
                                    if (iframe.contentDocument) {
                                        result = result.concat(getTexts(iframe.contentDocument));
                                    }
                                } catch (e) {
                                    // Cross-origin, skip
                                }
                            }
                        } catch (err) {
                            // doc might not be accessible, skip
                        }
                        return result;
                    }
                    return getTexts(document);
                },
                args: [value]
            });

            content = textArray.join('\n');
        }

        if (inputPrompt.value != "") {
            inputPrompt.value = inputPrompt.value + "\n" + content;
        } else {
            inputPrompt.value = content;
        }
    } catch (e) {
        showError(e);
    }
});

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
        } else {
            console.error('Screenshot capture failed', response && response.error);
            if (response) {
                showError(response.error);
            }
        }
    });
});

// user prompt
buttonSend.addEventListener('click', () => {
    showLoading();

    const imageUrl = getScreenshotUrl();
    const prompt = inputPrompt.value.trim();

    const parts = [];
    if (imageUrl && imageUrl.startsWith("data:")) {
        parts.push({
            contentType: "image/png",
            content: imageUrl,
        });
    }

    const payload = {
        content: prompt,
        parts: parts,
    };

    const msg = {
        id: "",
        type: "hub",
        sender: "chrome",
        recipient: "ai",
        payload: JSON.stringify(payload),
    };

    chrome.runtime.sendMessage({ action: 'send-message', data: msg }, (response) => {
        if (response && response.success) {
            showResponse("Success")
            // wait for real response
            showLoading()
        } else {
            console.error("Websocket send error:", response && response.error);
            if (response.error) {
                showError(response.error);
            }
        }
        console.log("response", response)
    });
});

buttonCopy.addEventListener('click', () => {
    const prompt = inputPrompt.value.trim();
    writeToClipboard(prompt);
});

buttonReset.addEventListener('click', () => {
    hide(screenshot)
    hide(elementError);
    hide(elementResponse);
    hide(elementLoading);

    setScreenshotUrl("");
    // reset to default?
    // inputSelector.value = '';
    inputPrompt.value = '';
    elementResponse.textContent = '';
    elementError.textContent = '';
});

chrome.runtime.onMessage.addListener((message) => {
    if (message.action === 'handle-message') {
        console.log("handle-message", message)

        // message
        if (message.data) {
            showResponse(`${message.data.code} ${message.data.payload}`);
        }
        if (message.error) {
            showError(message.error)
        }
    }

    if (message.action === 'handle-socket') {
        console.log("handle-socket", message)

        setHubIcon(message.active);
        hide(elementError);

        if (message.error) {
            showError(message.error)
        }
    }
});

function showLoading() {
    hide(elementResponse);
    hide(elementError);
    show(elementLoading);
}

function showResponse(response) {
    hide(elementLoading);
    hide(elementError);
    show(elementResponse);

    elementResponse.value = response;
}

function showError(error) {
    show(elementError);
    hide(elementResponse);
    hide(elementLoading);
    elementError.textContent = error;
}
