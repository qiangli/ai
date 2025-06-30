package ws

func Screenshot(wsUrl, prompt string) (string, error) {
	req := &Message{
		Recipient: "chrome",
		Action:    "screenshot",
		Payload:   "",
	}

	resp, err := SendRequest(wsUrl, prompt, req)
	if err != nil {
		return "", err
	}
	return resp.Payload, nil
}

func GetSelection(wsUrl string) (string, error) {
	req := &Message{
		Recipient: "chrome",
		Action:    "get-selection",
		Payload:   "", // default
	}

	resp, err := SendRequest(wsUrl, "", req)
	if err != nil {
		return "", err
	}
	return resp.Payload, nil
}

func VoiceInput(wsUrl, prompt string) (string, error) {
	req := &Message{
		Recipient: "chrome",
		Action:    "voice",
		Payload:   "",
	}

	resp, err := SendRequest(wsUrl, prompt, req)
	if err != nil {
		return "", err
	}
	return resp.Payload, nil
}
