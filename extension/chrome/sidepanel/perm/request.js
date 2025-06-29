// credit: https://medium.com/@lynchee.owo/how-to-enable-microphone-access-in-chrome-extensions-by-code-924295170080
async function getUserPermission() {
  console.log("Getting user permission for mic");
  await navigator.mediaDevices.getUserMedia({ audio: true });
  const micPermission = await navigator.permissions.query({
    name: "microphone",
  });
  if (micPermission.state == "granted") {
    window.close();
  }
}

getUserPermission();