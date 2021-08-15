package stitchers

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/icza/mjpeg"
)

// MJPEGStitcher contains the necessary parameters to create an MJPEG file.
type MJPEGStitcher struct{ width, height, fps int32 }

// NewMJPEGStitcher returns a pointer to mjpegstitcher to create an MJPEG from
// a slice of image.Image
func NewMJPEGStitcher(width, height, fps int32) *MJPEGStitcher {
	return &MJPEGStitcher{width, height, fps}
}

// Stitch combines the slice of image.Image to create an mjpeg saved at filename.
func (m *MJPEGStitcher) Stitch(images []image.Image, filename string) error {
	aw, err := mjpeg.New(filename, m.width, m.height, m.fps)
	if err != nil {
		return err
	}

	for _, m := range images {
		buf := &bytes.Buffer{}
		if err := jpeg.Encode(buf, m, nil); err != nil {
			return err
		}
		if err := aw.AddFrame(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}
