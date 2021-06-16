package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/michaelmcallister/timelapse/pkg/capturers/gocvcapture"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/math/fixed"
)

const (
	dateFormatImg  = "2006-01-02 15:04:05"
	dateFormatFile = "2006-01-02--15-04-05"
)

// for drawing text over the image.
var (
	defaultFont font.Face
	white       = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	black       = color.RGBA{0x00, 0x00, 0x00, 0xFF}
)

var (
	deviceID     = flag.Int("device", 0, "0 based index of recording device to use.")
	timeInterval = flag.Duration("interval", 1*time.Minute, "how often to capture an image.")
	filePath     = flag.String("filepath", ".", "path to store resultant images.")
)

// ImageCapturer defines the contract for capturing an image from a video device.
type ImageCapturer interface {
	Capture() (image.Image, error)
}

func init() {
	fo, _ := truetype.Parse(gomono.TTF)
	defaultFont = truetype.NewFace(fo, &truetype.Options{
		Size: 32.0,
	})
}

func labelImage(i image.Image, label string) {
	d := &font.Drawer{
		Dst:  i.(draw.Image),
		Face: defaultFont,
	}

	imX := i.Bounds().Min.X
	imY := i.Bounds().Max.Y

	// draw the text in black first to create a faux-border.
	for xx := -2; xx < 2; xx++ {
		for yy := -2; yy < 2; yy++ {
			d.Src = image.NewUniform(black)
			d.Dot = fixed.P(imX-xx, imY-yy)
			d.DrawString(label)
		}
	}

	// draw the text in white in the centre.
	d.Src = image.NewUniform(white)
	d.Dot = fixed.P(imX, imY)
	d.DrawString(label)
}

func saveImage(i image.Image, directory, filename string) error {
	file := filepath.Join(filepath.Join(directory), filename)
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, i); err != nil {
		return err
	}
	symFile := filepath.Join(filepath.Join(directory), "latest-raw.jpg")
	// Remove existing symlink if there.
	if _, err := os.Lstat(symFile); err == nil {
		os.Remove(symFile)
	}
	return os.Symlink(file, symFile)
}

func runRecord(capturer ImageCapturer) error {
	i, err := capturer.Capture()
	if err != nil {
		return err
	}
	labelImage(i, time.Now().Format(dateFormatImg))
	t := fmt.Sprintf("%s.jpg", time.Now().Format(dateFormatFile))
	return saveImage(i, *filePath, t)
}

func main() {
	ticker := time.NewTicker(*timeInterval)
	c := gocvcapture.New(*deviceID)
	for range ticker.C {
		if err := runRecord(c); err != nil {
			log.Fatal(err)
		}
	}
}
