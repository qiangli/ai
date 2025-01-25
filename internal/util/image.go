package util

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/BourgeoisBear/rasterm"
)

// https://sw.kovidgoyal.net/kitty/graphics-protocol/
// https://github.com/BourgeoisBear/rasterm/blob/main/rasterm_test.go
func printImage(out io.Writer, imageFile, mode string) error {
	getFile := func() (*os.File, int64, error) {
		f, err := os.Open(imageFile)
		if err != nil {
			return nil, 0, err
		}

		fi, err := f.Stat()
		if err != nil {
			return nil, 0, err
		}

		return f, fi.Size(), nil
	}

	f, imgLen, err := getFile()
	if err != nil {
		return err
	}
	defer f.Close()

	_, fmtName, err := image.DecodeConfig(f)
	if err != nil {
		return err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	switch mode {
	case "iterm":
		// WEZ/ITERM SUPPORT ALL FORMATS, SO NO NEED TO RE-ENCODE TO PNG
		err = rasterm.ItermCopyFileInline(out, f, imgLen)
	case "sixel":
		if iPaletted, bOK := img.(*image.Paletted); bOK {
			err = rasterm.SixelWriteImage(out, iPaletted)
		} else {
			fmt.Println("[NOT PALETTED, SKIPPING.]")
		}
	case "kitty":
		if fmtName == "png" {
			err = rasterm.KittyCopyPNGInline(out, f, rasterm.KittyImgOpts{})
		} else {
			err = rasterm.KittyWriteImage(out, img, rasterm.KittyImgOpts{})
		}
	}

	return err
}

func PrintImage(out io.Writer, imageFile string) error {
	var mode = ""
	if rasterm.IsKittyCapable() {
		mode = "kitty"
	} else if rasterm.IsItermCapable() {
		mode = "iterm"
	} else if ok, _ := rasterm.IsSixelCapable(); ok {
		mode = "sixel"
	} else {
		return fmt.Errorf("no supported terminal found")
	}

	return printImage(out, imageFile, mode)
}

// The ViewImage function attempts to open an image file using an external viewer.
// It first checks for a custom viewer specified in the AI_IMAGE_VIEWER environment variable.
// If not set, it defaults to using platform-specific tools to view the image.
func ViewImage(imageFile string) error {
	imageViewer, exists := os.LookupEnv("AI_IMAGE_VIEWER")
	if !exists {
		switch runtime.GOOS {
		case "windows":
			imageViewer = "rundll32.exe"
			return exec.Command(imageViewer, "shell32.dll,ImageView_Fullscreen", imageFile).Run()
		case "darwin":
			imageViewer = "open" // For macOS
			return exec.Command(imageViewer, imageFile).Run()
		default:
			imageViewer = "xdg-open" // For most Linux desktop environments
			return exec.Command(imageViewer, imageFile).Run()
		}
	}

	cmd := exec.Command(imageViewer, imageFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
