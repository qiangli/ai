package agent

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
)

func PrintOutput(format string, output *api.Output) error {
	s, err := util.FormatContent(format, output)
	if err != nil {
		return err
	}

	if isOutputTTY() {
		log.Infof("\n[%s]\n", output.Display)
		log.Println(s)
	} else {
		log.Println(s)
	}

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

func processTextContent(cfg *api.AppConfig, output *api.Output) {
	content := output.Content

	//
	// doc := util.ParseMarkdown(content)
	// total := len(doc.CodeBlocks)

	// clipboard
	if cfg.Clipout {
		clip := util.NewClipboard()
		if cfg.ClipAppend {
			if err := clip.Append(content); err != nil {
				log.Debugf("failed to append content to clipboard: %v\n", err)
			}
		} else {
			if err := clip.Write(content); err != nil {
				log.Debugf("failed to copy content to clipboard: %v\n", err)
			}
		}
	}

	if cfg.Output != "" {
		SaveOutput(cfg.Output, output)
	}

	// isPiped := func() bool {
	// 	stat, err := os.Stdout.Stat()
	// 	if err != nil {
	// 		return false
	// 	}
	// 	return (stat.Mode() & os.ModeCharDevice) == 0
	// }()

	PrintOutput(cfg.Format, output)

	// if total > 0 && isPiped {
	// 	// if there are code blocks and stdout is redirected
	// 	// we send the code blocks to the stdout
	// 	const codeTpl = "%s\n"
	// 	var snippets []string
	// 	for _, v := range doc.CodeBlocks {
	// 		snippets = append(snippets, v.Code)
	// 	}
	// 	// show code snippets
	// 	PrintOutput(cfg.Format, &api.Output{
	// 		Content: fmt.Sprintf(codeTpl, strings.Join(snippets, "\n")),
	// 	})
	// }
}

func processImageContent(cfg *api.AppConfig, message *api.Output) {
	var imageFile string
	if cfg.Output != "" {
		imageFile = cfg.Output
	} else {
		imageFile = filepath.Join(os.TempDir(), "image.png")
	}

	if err := saveImage(message.Content, imageFile); err != nil {
		log.Errorf("failed to save image: %v\n", err)
		return
	}

	if err := util.PrintImage(os.Stdout, imageFile); err != nil {
		if err := util.ViewImage(imageFile); err != nil {
			log.Errorf("failed to view image: %v\n", err)
		}
	}
}

func saveImage(b64Image string, dest string) error {
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
		log.Errorf("failed to write image to %s: %v\n", dest, err)
	}
	log.Infof("Image content saved to %s\n", dest)

	return nil
}
