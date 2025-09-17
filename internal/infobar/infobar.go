package infobar

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/term"
)

var (
	statusMu    sync.Mutex
	statusLines = []string{}
)

func InfoStatus(msg string) func(update string) {
	msg = "\033[45m" + "\033[1m" + msg + "\033[0m" // Add a magenta background

	statusMu.Lock()
	lineIndex := len(statusLines)
	statusLines = append(statusLines, msg)
	statusMu.Unlock()

	render := func() {
		_, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return
		}
		fmt.Print("\x1b[s") // save cursor
		for i, line := range statusLines {
			fmt.Printf("\x1b[%d;1H\x1b[2K%s", h-len(statusLines)+i+1, line)
		}
		fmt.Print("\x1b[u") // restore cursor
	}

	render() // initial render

	return func(update string) {
		statusMu.Lock()
		defer statusMu.Unlock()

		update = "\033[45m" + "\033[1m" + update + "\033[0m" // Add a magenta background
		statusLines[lineIndex] = update
		render()
	}
}
