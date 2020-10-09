package main

import (
	"fmt"
	"github.com/disintegration/imaging"
	"os/exec"
)

func CreateImageThumbnail(width uint, height uint, anchor imaging.Anchor, srcFile string, destFile string) error {
	// Open a test image.
	src, err := imaging.Open(srcFile)
	if err != nil {
		return fmt.Errorf("failed to open image: %v", err)
	}

	cropWidth := int(width)
	cropHeight := int(height)

	// Resize the image to at least the crop width and height preserving the aspect ratio.
	src = imaging.Resize(src, 0, cropHeight, imaging.Lanczos)
	if src.Bounds().Size().X < cropWidth {
		src = imaging.Resize(src, cropWidth, 0, imaging.Lanczos)
	}
	if src.Bounds().Size().Y < cropHeight {
		src = imaging.Resize(src, 0, cropHeight, imaging.Lanczos)
	}

	// Crop the image to the crop size using the center anchor.
	src = imaging.CropAnchor(src, cropWidth, cropHeight, anchor)

	// Save the resulting image
	err = imaging.Save(src, destFile)
	if err != nil {
		return fmt.Errorf("failed to save image: %v", err)
	}

	return nil
}

func CreateVideoGif(start int, duration int, scale int, frameRate int, srcFile string, destFile string) error {
	s := fmt.Sprintf("%v", start)
	t := fmt.Sprintf("%v", duration)

	e := exec.Command("ffmpeg", "-y", "-t", t, "-ss", s, "-i", srcFile, "-vf", "scale="+
		ToString(scale)+":-1", "-an", "-r", ToString(frameRate), "-crf", "24", destFile)

	stdout, err := e.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ERROR:\n%v\n%s\n", string(stdout), err)
	}

	return nil
}

func ConvertVideo(srcFile string, destFile string) {

}
