package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/michaelmcallister/timelapse/gocvcapture"
	"github.com/michaelmcallister/timelapse/stitchers"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/math/fixed"
)

const usage = `Usage: timelapse [OPTION]`

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

// 'record' flags.
const recordCmdName = "record"

var (
	recordCmd    = flag.NewFlagSet(recordCmdName, flag.ExitOnError)
	deviceID     = recordCmd.Int("device", 0, "0 based index of recording device to use.")
	timeInterval = recordCmd.Duration("interval", 1*time.Minute, "how often to capture an image.")
	filePath     = recordCmd.String("filepath", ".", "path to store resultant images.")
)

// 'stitch' flags.
const stitchCmdName = "stitch"

var (
	stitchCmd       = flag.NewFlagSet(stitchCmdName, flag.ExitOnError)
	stitchWidth     = flag.Int("width", 640, "width to use in the stitched file.")
	stitchHeight    = flag.Int("height", 480, "height to use in the stitched file.")
	stitchDirectory = flag.String("directory", "./", "directory full of jpgs to stitch together.")
	filename        = stitchCmd.String("filename", "out.gif", ".")
	fps             = stitchCmd.Int("fps", 60, "frames per second to use in the output.")
)

// ImageCapturer defines the contract for capturing an image from a video device.
type ImageCapturer interface {
	Capture() (image.Image, error)
}

// ImageStitcher defines the contract for taking multiple images and stitching them into a video.
type ImageStitcher interface {
	Stitch([]image.Image, string) error
}

func init() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	fo, _ := truetype.Parse(gomono.TTF)
	defaultFont = truetype.NewFace(fo, &truetype.Options{
		Size: 32.0,
	})

	switch os.Args[1] {
	case recordCmdName:
		recordCmd.Parse(os.Args[2:])
	case stitchCmdName:
		stitchCmd.Parse(os.Args[2:])
	default:
		fmt.Println("expected one of `record`, `stitch`")
	}
}

func main() {
	if recordCmd.Parsed() {
		ticker := time.NewTicker(*timeInterval)
		c := gocvcapture.New(*deviceID)
		for range ticker.C {
			runRecord(c)
		}
	}
	if stitchCmd.Parsed() {
		c := stitchers.NewMJPEGSticher(int32(*stitchWidth), int32(*stitchHeight), int32(*fps))

		var files []image.Image
		fs, err := ioutil.ReadDir(*stitchDirectory)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range fs {
			if !strings.HasSuffix(f.Name(), "jpg") {
				continue
			}
			f, err := os.Open(f.Name())
			if err != nil {
				panic(err)
			}
			image, _, err := image.Decode(f)
			if err != nil {
				panic(err)
			}
			files = append(files, image)
		}
		stitchFiles(c, files, *filename)
	}
}

func runRecord(capturer ImageCapturer) {
	i, err := capturer.Capture()
	if err != nil {
		panic(err)
	}

	labelImage(i, time.Now().Format(dateFormatImg))

	t := fmt.Sprintf("%s.jpg", time.Now().Format(dateFormatFile))
	err = saveImage(i, *filePath, t)
	if err != nil {
		panic(err)
	}
}

func stitchFiles(stitcher ImageStitcher, files []image.Image, filename string) {
	err := stitcher.Stitch(files, filename)
	if err != nil {
		panic(err)
	}
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
