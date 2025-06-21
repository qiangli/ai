package util

import (
	_ "embed"
	"fmt"

	"github.com/gen2brain/beeep"
)

//go:embed icons/info.png
var iconInfo []byte

//go:embed icons/warn.png
var iconWarn []byte

func Notify(s string) {
	// beeep.AppName = "AI"
	_ = beeep.Notify("", s, iconInfo)
}

func Alert(s string) {
	// beeep.AppName = "AI"
	_ = beeep.Alert("", s, iconWarn)
}

func Beep(n int) {
	_ = beeep.Beep(beeep.DefaultFreq, n)
}

func Bell(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\a")
	}
}
