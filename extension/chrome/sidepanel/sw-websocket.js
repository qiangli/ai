// console.log('sw-websocket.js')
// TODO 
// import './shared.js';

// default selectors for the domain
const selectorMap = {
    'leetcode.com': '[data-track-load="description_content"]',
    'codeforces.com': '.problem-statement',
    'codechef.com': '#problem-statement',
    'geeksforgeeks.org': '.problems_problem_content__Xm_eO',
    'interviewbit.com': '.p-statement',
    'testdome.com': '.copy-protection',
};

function getDefaultSelector(url, val) {
    for (const domain in selectorMap) {
        if (url.includes(domain)) {
            return selectorMap[domain];
        }
    }
    return val;
}

//
const TEN_SECONDS_MS = 10 * 1000;
const INITIAL_BACKOFF = 3000;      // start at 3s
const MAX_BACKOFF = 60 * 1000;     // max 1 minute
let webSocket = null;
let keepAliveIntervalId = null;
let reconnectTimeoutId = null;
let reconnectBackoff = INITIAL_BACKOFF;

chrome.runtime.onMessage.addListener((msg, _, sendResponse) => {
    if (msg.action === 'toggle-hub') {
        if (webSocket) {
            disconnect(true /* intentional */);
            sendResponse({ active: false });
        } else {
            connect(true /* userInitiated */);
            sendResponse({ active: true });
        }
        return true;
    }
    if (msg.action === 'get-hub-status') {
        sendResponse({ active: !!webSocket && webSocket.readyState === WebSocket.OPEN });
        return true;
    }
});

// ... existing code ...

function connect(userInitiated = false) {
    if (webSocket && (webSocket.readyState === WebSocket.CONNECTING || webSocket.readyState === WebSocket.OPEN)) return;

    cleanup();

    webSocket = new WebSocket('ws://localhost:58080/hub');
    webSocket.onopen = () => {
        reconnectBackoff = INITIAL_BACKOFF; // Reset backoff
        chrome.runtime.sendMessage({ action: 'handle-socket', active: true });
        webSocket.send(JSON.stringify({ type: 'register', sender: 'chrome' }));
        startKeepAlive();
        console.log('[WS] Connected and registered.');
    };

    webSocket.onmessage = (event) => {
        console.log("[WS] Dispatching event", event.data);
        dispatchMessage(event);
    };

    webSocket.onclose = (evt) => {
        console.warn(`[WS] Closed (reason: ${evt.reason || "n/a"})`);
        handleDisconnect(false);
    };

    webSocket.onerror = (e) => {
        console.error('[WS] WebSocket error:', e);
        handleDisconnect(false);
    };

    if (userInitiated) {
        startKeepAlive();
    }
}

// Ensures cleanup, closes socket, and optionally stops keepalive
function disconnect(stopKeepAlive = false) {
    if (webSocket) {
        try { webSocket.close(); } catch (err) { /* ignored */ }
        webSocket.onclose = webSocket.onmessage = webSocket.onerror = null;
        webSocket = null;
    }
    if (stopKeepAlive) stopKeepAliveInterval();
    clearReconnectTimeout();
    console.log('[WS] Disconnected.');
}

function startKeepAlive() {
    if (!keepAliveIntervalId) {
        keepAliveIntervalId = setInterval(sendHeartbeat, TEN_SECONDS_MS);
    }
}
function stopKeepAliveInterval() {
    if (keepAliveIntervalId) {
        clearInterval(keepAliveIntervalId);
        keepAliveIntervalId = null;
    }
}

function sendHeartbeat() {
    if (webSocket && webSocket.readyState === WebSocket.OPEN) {
        try {
            webSocket.send(JSON.stringify({ type: 'heartbeat', sender: 'chrome' }));
            console.debug('[WS] Sent heartbeat');
        } catch (err) {
            console.warn('[WS] Heartbeat error:', err);
        }
    } else {
        handleDisconnect(false);
    }
}

function clearReconnectTimeout() {
    if (reconnectTimeoutId) {
        clearTimeout(reconnectTimeoutId);
        reconnectTimeoutId = null;
    }
}

// Handles unintentional disconnects
function handleDisconnect(intentional) {
    chrome.runtime.sendMessage({ action: 'handle-socket', active: false, error: 'Websocket closed' });
    stopKeepAliveInterval();
    clearReconnectTimeout();

    if (!intentional) {
        // Exponential backoff reconnect
        reconnectTimeoutId = setTimeout(() => {
            console.log(`[WS] Attempting reconnect...`);
            connect();
        }, reconnectBackoff);
        reconnectBackoff = Math.min(MAX_BACKOFF, reconnectBackoff * 2);
    }
    cleanup();
}

function cleanup() {
    if (webSocket) {
        try { webSocket.close(); } catch (err) { }
        webSocket.onclose = webSocket.onmessage = webSocket.onerror = null;
        webSocket = null;
    }
}

function dispatchMessage(event) {
    try {
        const message = JSON.parse(event.data);
        switch (message.action) {
            case 'screenshot':
                captureScreenshot(message);
                break;
            case 'get-selection':
                getSelection(message);
                break;
            case 'voice':
                captureVoice(message);
                break;
            default:
                chrome.runtime.sendMessage({ action: 'handle-message', data: message });
        }
    } catch (e) {
        console.error('Failed to parse payload:', e);
        chrome.runtime.sendMessage({ action: 'handle-message', error: e });
    }
}

function captureScreenshot(request) {
    chrome.tabs.query({ active: true, currentWindow: true }).then((tabs) => {
        if (tabs.length === 0) {
            sendWebsocketResponse(request, "500", "no active tab");
            return;
        }
        const tab = tabs[0];
        chrome.tabs.captureVisibleTab(tab.windowId, { format: 'png' }).then((imageUri) => {
            console.log("captured imageUri:", imageUri ? imageUri.substring(0, 100) + "..." : null);
            sendWebsocketResponse(request, "200", imageUri);
        }).catch((error) => {
            console.error('error capturing screenshot:', error);
            sendWebsocketResponse(request, "500", error.message);
        });
    }).catch((error) => {
        console.error('error querying tabs:', error);
        sendWebsocketResponse(request, "500", error.message);
    });
}

async function getSelection(request) {
    try {
        let value = request.payload;
        const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

        let content;

        if (!value) {
            const url = tab?.url || '';
            value = getDefaultSelector(url, 'body')
        }

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

        console.log("selection", content);
        sendWebsocketResponse(request, "200", content);
    } catch (e) {
        console.error('error getting text selection:', e);
        sendWebsocketResponse(request, "500", e.message);
    }
}

function captureVoice(request) {
    chrome.runtime.sendMessage({ action: 'transcribe-voice' }, (response) => {
        console.log("response", response)
        if (response.data) {
            sendWebsocketResponse(request, "200", response.data);
        } else {
            sendWebsocketResponse(request, "500", response.error);
        }
    });
}

function sendWebsocketResponse(request, code, payload) {
    try {
        if (webSocket && webSocket.readyState === WebSocket.OPEN) {
            webSocket.send(JSON.stringify({
                type: "response",
                reference: request.id,
                sender: "chrome",
                recipient: request.sender,
                code: code,
                payload: payload,
            }));
            console.log("message sent");
        } else {
            throw new Error('Websocket connection not established');
        }
    } catch (e) {
        console.error("Websocket send error:", e);
    }
}

chrome.runtime.onMessage.addListener((message, _, sendResponse) => {
    if (message.action === 'send-message') {
        try {
            if (!webSocket) {
                throw new Error('Websocket connection not established');
            }
            const data = JSON.stringify(message.data)
            webSocket.send(data);
            sendResponse({ success: true });
        } catch (e) {
            console.error('Failed to send message over Websocket:', e);
            sendResponse({ error: e.message });
        }
        return true;
    }
});
