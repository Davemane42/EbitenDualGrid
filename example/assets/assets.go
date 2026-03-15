package assets

import (
	"embed"
	_ "image/png"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed *.png
var fs embed.FS

var Images map[string]*ebiten.Image

func init() {
	Images = make(map[string]*ebiten.Image)
	entries, err := fs.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		f, err := fs.Open(entry.Name())
		if err != nil {
			log.Fatal(err)
		}
		img, _, err := ebitenutil.NewImageFromReader(f)
		if err != nil {
			log.Fatal(err)
		}
		Images[strings.TrimSuffix(entry.Name(), ".png")] = img
	}
}
