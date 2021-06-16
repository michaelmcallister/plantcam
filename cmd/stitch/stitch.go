package main

import (
	"errors"
	"flag"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/michaelmcallister/timelapse/pkg/stitchers"
)

// ErrUnsupportedFileFormat is returned when
var ErrUnsupportedFileFormat = errors.New("unsupported file format")

// StitchFormat represents available file extensions for stitching.
type StitchFormat string

const (
	// MJPEG is a file format where each frame is compressed seperately as a JPEG.
	MJPEG StitchFormat = ".mjpeg"
	// GIF only supports up to 256 colours.
	GIF StitchFormat = ".gif"
)

var (
	stitchWidth     = flag.Int("width", 640, "width to use in the stitched file.")
	stitchHeight    = flag.Int("height", 480, "height to use in the stitched file.")
	stitchDirectory = flag.String("directory", "./", "directory full of jpgs to stitch together.")
	filename        = flag.String("filename", "out.mjpeg", ".")
	fps             = flag.Int("fps", 60, "frames per second to use in the output.")
)

// ImageStitcher defines the contract for taking multiple images and stitching them into a video.
type ImageStitcher interface {
	Stitch([]image.Image, string) error
}

func parseStitcher() (ImageStitcher, error) {
	switch ff := StitchFormat(filepath.Ext(*filename)); ff {
	case MJPEG:
		w, h, fps := int32(*stitchWidth), int32(*stitchHeight), int32(*fps)
		return stitchers.NewMJPEGSticher(w, h, fps), nil
	case GIF:
		return stitchers.NewGifStitcher(), nil
	default:
		log.Printf("Unknown file format: %s", ff)
		return nil, ErrUnsupportedFileFormat
	}
}

func main() {
	c, err := parseStitcher()
	if err != nil {
		log.Fatal(err)
	}

	var files []image.Image
	fs, err := ioutil.ReadDir(*stitchDirectory)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range fs {
		if !strings.HasSuffix(f.Name(), "jpg") {
			continue
		}
		f, err := os.Open(filepath.Join(*stitchDirectory, f.Name()))
		if err != nil {
			log.Fatal(err)
		}
		image, _, err := image.Decode(f)
		if err != nil {
			log.Fatal(err)
		}
		files = append(files, image)
	}
	if err := c.Stitch(files, *filename); err != nil {
		log.Fatal(err)
	}
}
