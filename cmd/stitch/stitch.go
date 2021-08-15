package main

import (
	_ "image/jpeg"
	_ "image/png"

	"errors"
	"flag"
	"image"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	log "github.com/golang/glog"
	"github.com/michaelmcallister/timelapse/pkg/stitchers"
)

// ErrUnsupportedFileFormat is returned when an unsupported file extension
// is provided.
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
	fps             = flag.Int("fps", 60, "frames per second to use in the output. Not supported for GIF")
	minLightness    = flag.Float64("lightness", 0.5, "minimum lightness (as in HSL). Images darker than this value will be discarded.")
)

// ImageStitcher defines the contract for taking multiple images and stitching them into a video.
type ImageStitcher interface {
	Stitch([]image.Image, string) error
}

func parseStitcher() (ImageStitcher, error) {
	switch ff := StitchFormat(filepath.Ext(*filename)); ff {
	case MJPEG:
		w, h, fps := int32(*stitchWidth), int32(*stitchHeight), int32(*fps)
		return stitchers.NewMJPEGStitcher(w, h, fps), nil
	case GIF:
		return stitchers.NewGifStitcher(), nil
	default:
		log.Warningf("Unknown file format: %s", ff)
		return nil, ErrUnsupportedFileFormat
	}
}

// brightness returns the perceived brightness of an RGB value, it is based on
// https://en.wikipedia.org/wiki/HSL_and_HSV#From_RGB
// The return value is a percentage of 255.
func brightness(r, g, b uint32) float32 {
	r /= 255
	g /= 255
	b /= 255
	minOf := func(vars ...uint32) uint32 {
		min := vars[0]
		for _, i := range vars {
			if min > i {
				min = i
			}
		}
		return min
	}

	maxOf := func(vars ...uint32) uint32 {
		max := vars[0]
		for _, i := range vars {
			if max < i {
				max = i
			}
		}
		return max
	}
	return (float32(maxOf(r, g, b)+minOf(r, g, b)) / 2) / 255
}

// isLightEnough returns true of the img is as bright, or brighter than min
// else false. For the definition of lightness see the following:
// https://en.wikipedia.org/wiki/HSL_and_HSV#Lightness
func isLightEnough(min float32, img image.Image) bool {
	maxPoints := float64(img.Bounds().Dx() * img.Bounds().Dy())
	samplePoints := maxPoints * 0.1
	var sum float32
	var i int
	for i = 0; i < int(samplePoints) || i == int(maxPoints); i++ {
		x := rand.Intn(img.Bounds().Dx())
		y := rand.Intn(img.Bounds().Dy())
		r, g, b, _ := img.At(x, y).RGBA()
		l := brightness(r, g, b)
		sum += l
	}
	b := (sum / float32(i))
	log.V(2).Infoln("average brightness", b)
	return b >= min
}

func main() {
	flag.Parse()

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
		if !strings.HasSuffix(f.Name(), "png") {
			continue
		}
		f, err := os.Open(filepath.Join(*stitchDirectory, f.Name()))
		if err != nil {
			log.Fatal(err)
		}
		image, _, err := image.Decode(f)
		if err != nil {
			log.Warningf("unable to decode %s due to: %v", f.Name(), err)
			log.Infoln("skipping...")
			continue
		}
		if !isLightEnough(float32(*minLightness), image) {
			log.Infof("image %q not bright enough, skipping...", f.Name())
			continue
		}
		files = append(files, image)
	}
	if err := c.Stitch(files, *filename); err != nil {
		log.Fatal(err)
	}
}
