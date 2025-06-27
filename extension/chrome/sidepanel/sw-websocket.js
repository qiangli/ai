
// console.log('sw-websocket.js')

const TEN_SECONDS_MS = 10 * 1000;
let webSocket = null;
let keepAliveIntervalId = null;

chrome.runtime.onMessage.addListener((msg, _, sendResponse) => {
    if (msg.action === 'toggle-hub') {
        console.log("toggle web socket", webSocket);

        if (webSocket) {
            disconnect();
            sendResponse({ active: false });
        } else {
            connect();
            sendResponse({ active: true }); // optimistic, real status callback could be added if desired
        }
        return true;
    }

    if (msg.action === 'get-hub-status') {
        sendResponse({ active: !!webSocket });
        return true;
    }
});

function connect() {
    if (webSocket) return; // Don't double-connect

    webSocket = new WebSocket('ws://localhost:58080/hub');

    webSocket.onopen = () => {
        // chrome.action.setIcon({ path: 'images/socket-active.png' });
        console.log('onopen');
        webSocket.send(JSON.stringify({
            type: 'register',
            sender: 'chrome',
        }));
        startKeepAlive();
    };

    webSocket.onmessage = (event) => {
        console.log("onmessage", event.data);
        chrome.runtime.sendMessage({ action: 'handle-message', data: event.data });
    };

    webSocket.onclose = () => {
        // chrome.action.setIcon({ path: 'images/socket-inactive.png' });
        console.log('onclose');
        cleanup();
    };

    webSocket.onerror = (e) => {
        console.error('WebSocket error:', e);
        // Attempt cleanup on error
        cleanup();
    };
}

function disconnect() {
    if (webSocket) {
        webSocket.onclose = null; // Prevent double cleanup
        webSocket.close();
        cleanup();
    }
}

function startKeepAlive() {
    if (keepAliveIntervalId) clearInterval(keepAliveIntervalId);
    keepAliveIntervalId = setInterval(() => {
        if (webSocket && webSocket.readyState === WebSocket.OPEN) {
            console.log('keep alive');
            webSocket.send(JSON.stringify({
                type: 'heartbeat',
                sender: 'chrome',
            }));
        } else {
            cleanup();
        }
    }, TEN_SECONDS_MS);
}

function cleanup() {
    if (webSocket) {
        try { webSocket.close(); } catch (e) { }
        webSocket = null;
    }
    if (keepAliveIntervalId) {
        clearInterval(keepAliveIntervalId);
        keepAliveIntervalId = null;
    }
    // chrome.action.setIcon({ path: 'images/socket-inactive.png' });
    console.log('WebSocket cleaned up/disconnected.');
}

chrome.runtime.onMessage.addListener((message, _, sendResponse) => {
    if (message.action === 'send-message') {
        try {
            if (!webSocket) {
                throw new Error('WebSocket connection not established');
            }
            const data = JSON.stringify(message.data)
            webSocket.send(data);
            sendResponse({ success: true });
        } catch (error) {
            console.error('Failed to send message over WebSocket:', error);
            sendResponse({ error: error.message });
        }
        return true;
    }
});
