//
const buttonMic = document.body.querySelector('#button-mic');
const micOpen = document.body.querySelector('#mic-open');
const micMute = document.body.querySelector('#mic-mute');

function toggleMic(recording) {
    if (recording) {
        show(micOpen);
        hide(micMute);
    } else {
        hide(micOpen);
        show(micMute);
    }
}

let recognition;
let isRecording = false;

if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    recognition = new SpeechRecognition();
    recognition.lang = 'en-US';
    recognition.interimResults = false;

    recognition.onresult = function (event) {
        const transcript = event.results[0][0].transcript;
        inputPrompt.value += (inputPrompt.value ? ' ' : '') + transcript;
    };

    recognition.onerror = function (event) {
        alert('Speech recognition error: ' + event.error);
        buttonMic.disabled = false;
        isRecording = false;
        toggleMic(false);
    };

    recognition.onend = function () {
        buttonMic.disabled = false;
        isRecording = false;
        toggleMic(false);
    };
} else {
    buttonMic.disabled = true;
    buttonMic.title = "Speech recognition not supported";
}

buttonMic.addEventListener('click', async () => {
    const granted = await checkMicPermission();

    console.log("microphone access permission ", granted)
    if (!granted) {
        console.error("Microphone permission denied. Please allow access to use this feature.");
        return;
    }

    if (!recognition || isRecording) {
        recognition && recognition.stop();
        return;
    }
    isRecording = true;
    buttonMic.disabled = true;
    toggleMic(true);

    recognition.start();
});

async function checkMicPermission() {
    let status = await navigator.permissions.query({ name: 'microphone' });
    if (status.state === 'granted') {
        return true;
    }

    // Open permissions page (perm.html) in a new tab
    // await new Promise(resolve => {
    //     chrome.tabs.create({ url: chrome.runtime.getURL('perm/request.html'), active: true });
    // });
    chrome.tabs.create({ url: chrome.runtime.getURL('perm/request.html'), active: true });

    status = await navigator.permissions.query({ name: 'microphone' });
    return status.state === 'granted';
}

chrome.runtime.onMessage.addListener(function (request, sender, sendResponse) {
    if (request.action === 'transcribe-voice') {
        if (!recognition) {
            sendResponse({ error: "speech recognition not supported." });
            return;
        }
        // Temporarily store the sendResponse function
        let responded = false;
        recognition.onresult = function (event) {
            const transcript = event.results[0][0].transcript;
            sendResponse({ data: transcript });
            responded = true;
        };
        recognition.onerror = function (event) {
            sendResponse({ error: event.error });
            responded = true;
        };
        recognition.onend = function () {
            if (!responded) sendResponse({ error: "no speech detected." });
        };
        recognition.start();

        return true;
    }
});
