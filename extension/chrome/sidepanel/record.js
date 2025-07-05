// console.log('record.js')

const buttonRecord = document.body.querySelector('#button-record');
const recordOff = document.body.querySelector('#record-off');
const recordOn = document.body.querySelector('#record-on');

function updateRecordButton(recording) {
  if (recording) {
    show(recordOn);
    hide(recordOff);
  } else {
    hide(recordOn);
    show(recordOff);
  }
}

let recording = false;

buttonRecord.addEventListener('click', () => {
  recording = !recording;
  console.log("buttone record", recording)
  updateRecordButton(recording);

  const action = recording ? 'start-recording' : 'stop-recording'

  chrome.runtime.sendMessage({ action: action, state: recording }, (response) => {
    if (chrome.runtime.lastError) {
      console.error(chrome.runtime.lastError.message);
      showError(chrome.runtime.lastError.message);
      return;
    }
    console.log("toggle record response", response)
    if (response && response.success) {
      showResponse(response.data);
    } else {
      if (response) {
        const error = response.error || 'failed';
        showError(error);
      }
    }
  });
});
