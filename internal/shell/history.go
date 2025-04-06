package shell

import (
	"fmt"
	"strconv"
)

var history []string

func clearHistory() {
	history = []string{}
	fmt.Println("\033[0;32m✔ history cleared\033[0m")
}

func showHistory() {
	width := len(strconv.Itoa(len(history)))
	for num := 1; num <= len(history); num++ {
		padded := fmt.Sprintf("%*d", width, num)
		fmt.Println("  \033[0;32m" + padded + " \033[0m │  " + history[num-1])
	}
}

func updateHistory(command string) {
	history = append(history, command)
}

func getCommandHist() []string {
	return history
}
