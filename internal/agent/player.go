package agent

import (
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
)

func PlayAudio(data string) error {

	op := &oto.NewContextOptions{}
	op.SampleRate = 24000
	op.ChannelCount = 1
	op.Format = oto.FormatSignedInt16LE

	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return err
	}

	<-readyChan

	reader := strings.NewReader(data)
	player := otoCtx.NewPlayer(reader)
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
