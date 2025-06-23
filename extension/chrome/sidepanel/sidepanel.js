console.log('sidepanel.js');

const inputSelector = document.body.querySelector('#input-selector');
const inputContent = document.body.querySelector('#input-content');
const buttonGetContent = document.body.querySelector('#button-get-content');
const buttonCopyContent = document.body.querySelector('#button-copy-content');
const buttonClearContent = document.body.querySelector('#button-clear-content');

const buttonScreenshot = document.body.querySelector('#button-screenshot');

const inputPrompt = document.body.querySelector('#input-prompt');

const buttonCopyPrompt = document.body.querySelector('#button-copy-prompt');
const buttonClearPrompt = document.body.querySelector('#button-clear-prompt');

const buttonGo = document.body.querySelector('#button-go');
const buttonGet = document.body.querySelector('#button-get');
const buttonReset = document.body.querySelector('#button-reset');
const buttonOn = document.body.querySelector('#button-on');
const buttonOff = document.body.querySelector('#button-off');

const elementResponse = document.body.querySelector('#response');
const elementLoading = document.body.querySelector('#loading');
const elementError = document.body.querySelector('#error');

// extract page content from selector
buttonGetContent.addEventListener('click', async () => {
  try {
    const value = inputSelector.value;
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
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

    const content = textArray.join('\n');
    inputContent.value = content
  } catch (e) {
    showError(e);
  }
});

buttonCopyContent.addEventListener('click', async () => {
  const content = inputContent.value.trim();
  writeToClipboard(content)
});

buttonClearContent.addEventListener('click', async () => {
  inputContent.value = '';
});

// screenshot
buttonScreenshot.addEventListener('click', async () => {
  chrome.runtime.sendMessage({ action: 'captureScreenshot' });
});


// user prompt
buttonCopyPrompt.addEventListener('click', async () => {
  const prompt = inputPrompt.value.trim();
  writeToClipboard(prompt)
});

buttonClearPrompt.addEventListener('click', async () => {
  inputPrompt.value = '';
});

// watch commands
buttonGo.addEventListener('click', async () => {
  showLoading();
  try {
    writeToClipboard('#todo')
  } catch (e) {
    showError(e);
  }
});

buttonGet.addEventListener('click', async () => {
  readFromClipboard().then(text => {
    showResponse(text)
  });
});

buttonReset.addEventListener('click', async () => {
  inputPrompt.value = '';
  writeToClipboard('#reset')
});

buttonOn.addEventListener('click', async () => {
  writeToClipboard('#on')
});

buttonOff.addEventListener('click', async () => {
  writeToClipboard('#off')
});

function showLoading() {
  hide(elementResponse);
  hide(elementError);
  show(elementLoading);
}

function showResponse(response) {
  hide(elementLoading);
  show(elementResponse);

  // Make sure to preserve line breaks in the response
  elementResponse.textContent = '';
  const paragraphs = response.split(/\r?\n/);
  for (let i = 0; i < paragraphs.length; i++) {
    const paragraph = paragraphs[i];
    if (paragraph) {
      elementResponse.appendChild(document.createTextNode(paragraph));
    }
    // Don't add a new line after the final paragraph
    if (i < paragraphs.length - 1) {
      elementResponse.appendChild(document.createElement('BR'));
    }
  }
}

function showError(error) {
  show(elementError);
  hide(elementResponse);
  hide(elementLoading);
  elementError.textContent = error;
}

function show(element) {
  element.removeAttribute('hidden');
}

function hide(element) {
  element.setAttribute('hidden', '');
}
