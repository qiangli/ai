// Write to clipboard
async function writeToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        console.log('Text copied:', text);
    } catch (err) {
        console.error('Failed to write to clipboard: ', err);
    }
}

// Read from clipboard
async function readFromClipboard() {
    try {
        const text = await navigator.clipboard.readText();
        console.log('Clipboard contents:', text);
        return text;
    } catch (err) {
        console.error('Failed to read from clipboard: ', err);
        return null;
    }
}

// Example usage:
// writeToClipboard('Hello from side panel!');
// readFromClipboard().then(text => console.log(text));
