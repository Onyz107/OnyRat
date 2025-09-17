package banner

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/common-nighthawk/go-figure"
	"golang.org/x/term"
)

var fonts = [51]string{
	"3-d", "5lineoblique", "alligator", "alligator2", "avatar", "basic",
	"big", "chunky", "coinstak", "colossal", "cosmic", "cosmike", "diamond",
	"doom", "drpepper", "epic", "fender", "fuzzy", "gothic", "graffiti",
	"isometric1", "isometric2", "isometric3", "isometric4", "jazmine",
	"kban", "larry3d", "lean", "nancyj-fancy", "nancyj-underlined",
	"nancyj", "o8", "ogre", "pawp", "poison", "puffy", "rev", "roman",
	"rounded", "rowancap", "rozzo", "sblood", "slant", "smisome1", "smslant",
	"speed", "starwars", "stop", "tanja", "tombstone", "usaflag",
}

func PrintBanner() {
	// get terminal width; fallback to 80 if unavailable.
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
	}

	RandomFont := fonts[rand.Intn(len(fonts))]
	fig := figure.NewColorFigure("OnyRAT", RandomFont, "purple", true)

	lines := fig.Slicify()

	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	margin := 0
	if width > maxWidth {
		margin = (width - maxWidth) / 2
	}

	colors := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"purple": "\033[35m",
		"cyan":   "\033[36m",
		"reset":  "\033[0m",
	}

	colorCode := colors["purple"]
	resetCode := colors["reset"]

	for _, line := range lines {
		paddedLine := strings.Repeat(" ", margin) + line
		coloredLine := colorCode + paddedLine + resetCode
		fmt.Println(coloredLine)
	}
}
