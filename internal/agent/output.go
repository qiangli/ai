package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func PrintOutput(ctx context.Context, format string, output *api.Output) error {
	s, err := util.FormatContent(format, output)
	if err != nil {
		return err
	}

	log.GetLogger(ctx).Infof("\n[%s]\n", output.Display)
	log.GetLogger(ctx).Printf("%s\n", s)

	return nil
}

func SaveOutput(filename string, message *api.Output) error {
	if message == nil {
		return nil
	}
	if filename == "" {
		return nil
	}
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(message.Content), os.ModePerm)
}

func processTextContent(ctx context.Context, cfg *api.AppConfig, output *api.Output) {
	content := output.Content

	// clipboard
	if cfg.Clipout {
		clip := util.NewClipboard()
		if cfg.ClipAppend {
			if err := clip.Append(content); err != nil {
				log.GetLogger(ctx).Debugf("failed to append content to clipboard: %v\n", err)
			}
		} else {
			if err := clip.Write(content); err != nil {
				log.GetLogger(ctx).Debugf("failed to copy content to clipboard: %v\n", err)
			}
		}
	}

	if cfg.Output != "" {
		SaveOutput(cfg.Output, output)
	}

	if cfg.Format == "tts" {
		SpeakOutput(ctx, cfg, output)
		return
	}
	PrintOutput(ctx, cfg.Format, output)
}

func SpeakOutput(ctx context.Context, cfg *api.AppConfig, output *api.Output) {
	var s = output.Content
	log.GetLogger(ctx).Printf("%s\n", s)
	// TOSO move to tools
	//
	// err := speak(cfg, s)
	// if err != nil {
	// 	log.Println(err.Error())
	// }
}

func processImageContent(ctx context.Context, cfg *api.AppConfig, message *api.Output) {
	var imageFile string
	if cfg.Output != "" {
		imageFile = cfg.Output
	} else {
		imageFile = filepath.Join(os.TempDir(), "image.png")
	}

	log.GetLogger(ctx).Infof("image file: %s\n", imageFile)

	if err := saveImage(ctx, message.Content, imageFile); err != nil {
		log.GetLogger(ctx).Errorf("failed to save image: %v\n", err)
		return
	}

	if err := util.PrintImage(os.Stdout, imageFile); err != nil {
		if err := util.ViewImage(imageFile); err != nil {
			log.GetLogger(ctx).Errorf("failed to view image: %v\n", err)
		}
	}
}

func saveImage(ctx context.Context, b64Image string, dest string) error {
	// https://sw.kovidgoyal.net/kitty/graphics-protocol/
	img, _, err := image.Decode(base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64Image)))
	if err != nil {
		return err

	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}

	err = os.WriteFile(dest, buf.Bytes(), 0755)
	if err != nil {
		log.GetLogger(ctx).Errorf("failed to write image to %s: %v\n", dest, err)
	}
	log.GetLogger(ctx).Infof("Image content saved to %s\n", dest)

	return nil
}
