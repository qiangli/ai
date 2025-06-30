package agent

import (
	"context"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/swarm/api"
)

func speak(cfg *api.AppConfig, text string) error {
	client := openai.NewClient(
		option.WithAPIKey(cfg.TTS.ApiKey),
		// option.WithBaseURL(baseUrl),
	)
	ctx := context.Background()

	// openai.SpeechModelTTS1
	// SpeechModelGPT4oMiniTTS
	res, err := client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Model:          openai.SpeechModelGPT4oMiniTTS,
		Input:          text,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatPCM,
		Voice:          openai.AudioSpeechNewParamsVoiceAlloy,
	})
	if err != nil {
		return err
	}

	defer res.Body.Close()

	op := &oto.NewContextOptions{}
	op.SampleRate = 24000
	op.ChannelCount = 1
	op.Format = oto.FormatSignedInt16LE

	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return err
	}

	<-readyChan

	player := otoCtx.NewPlayer(res.Body)
	player.Play()
	for player.IsPlaying() {
		time.Sleep(time.Millisecond)
	}
	err = player.Close()
	if err != nil {
		return err
	}

	return nil
}
