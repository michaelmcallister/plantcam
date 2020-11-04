package stitchers

import (
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
)

type gifstitcher struct{}

func NewGifStitcher() *gifstitcher { return &gifstitcher{} }

// Stitch will write the images to the filename in GIF format.
func (*gifstitcher) Stitch(images []image.Image, filename string) error {
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		return err
	}

	outGif := &gif.GIF{}
	for _, m := range images {
		bounds := m.Bounds()
		palettedImage := image.NewPaletted(bounds, palette.Plan9)
		draw.Draw(palettedImage, palettedImage.Rect, m, bounds.Min, draw.Over)

		// Add new frame to animated GIF
		outGif.Image = append(outGif.Image, palettedImage)
		outGif.Delay = append(outGif.Delay, 0)
	}
	gif.EncodeAll(f, outGif)

	return f.Sync()
}
