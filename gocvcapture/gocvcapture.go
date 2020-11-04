package gocvcapture

import (
	"errors"
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

const videoCodec = "MJPG"

// GocvCapturer exports a single Capture method to retrieve images
// from a capture device.
type GocvCapturer struct {
	deviceID int
}

// New returns an instance of gocvCapturer that is capable of retrieving
// images from the supplied deviceID.
func New(deviceID int) *GocvCapturer {
	return &GocvCapturer{deviceID: deviceID}
}

// Capture returns an image, or error if unable to capture from the device.
func (g *GocvCapturer) Capture() (image.Image, error) {
	webcam, err := gocv.VideoCaptureDevice(g.deviceID)
	defer webcam.Close()
	if err != nil {
		return nil, fmt.Errorf("unable to open video capture device: %v", g.deviceID)
	}

	img := gocv.NewMat()
	defer img.Close()

	if ok := webcam.Read(&img); !ok {
		return nil, fmt.Errorf("cannot read device %v", g.deviceID)
	}
	if img.Empty() {
		return nil, fmt.Errorf("no image on device %v", g.deviceID)
	}

	return img.ToImage()
}

// Stitch combines the supplied files into a video saved to the supplied filename.
func (g *GocvCapturer) Stitch(files []image.Image, filename string) error {
	vwr, err := gocv.VideoWriterFile(filename, videoCodec, 40.0, 640, 480, true)
	if err != nil {
		return err
	}

	defer vwr.Close()

	for _, f := range files {
		m, err := gocv.ImageToMatRGB(f)
		if err != nil {
			return err
		}
		if m.Empty() {
			return fmt.Errorf("unable to read image: %s", f)
		}
		if !vwr.IsOpened() {
			return errors.New("not ready to be written to")
		}
		if err := vwr.Write(m); err != nil {
			return err
		}
	}
	return nil
}
