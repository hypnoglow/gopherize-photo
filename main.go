package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png" // import for side-effect registering png format
	"log"
	"os"

	"github.com/nfnt/resize"
	"gocv.io/x/gocv"
)

func main() {
	var (
		classifierFilePath       string
		photoImagePath           string
		photoDetectScale         float64
		photoDetectMinNeighbours int
		photoDetectMinSize       int
		photoDetectMaxSize       int
		gopherImagePath          string
		gopherSizeCoeff          float64
		gopherXCoeff             float64
		gopherYCoeff             float64
		outputImagePath          string
	)

	flag.StringVar(&classifierFilePath, "classifier", "", "Path to classifier file (required)")
	flag.StringVar(&photoImagePath, "photo", "", "Path to photo image file (required)")
	flag.Float64Var(&photoDetectScale, "photo-detect-scale", 1.1, "Photo face detection scale parameter (default: 1.1)")
	flag.IntVar(&photoDetectMinNeighbours, "photo-detect-min-neighbours", 8, "Photo face detection min neighbours parameter (default: 8)")
	flag.IntVar(&photoDetectMinSize, "photo-detect-min-size", 200, "Photo face detection min size parameter (default: 200)")
	flag.IntVar(&photoDetectMaxSize, "photo-detect-max-size", 800, "Photo face detection max sie parameter (default: 800")
	flag.StringVar(&gopherImagePath, "gopher", "", "Path to gopher image file (required)")
	flag.Float64Var(&gopherSizeCoeff, "gopher-size-coeff", 3.0, "Coefficient for gopher size (default: 3.0)")
	flag.Float64Var(&gopherXCoeff, "gopher-x-coeff", 0.0, "Coefficient for gopher X axis adjustment (default: 0)")
	flag.Float64Var(&gopherYCoeff, "gopher-y-coeff", 0.0, "Coefficient for gopher Y axis adjustment (default: 0)")
	flag.StringVar(&outputImagePath, "out", "output.jpg", "Path to output image (default: output.jpg)")
	flag.Parse()

	if classifierFilePath == "" {
		flag.Usage()
		log.Fatal("Flag -classifier value is required")
	}
	if photoImagePath == "" {
		flag.Usage()
		log.Fatal("Flag -photo value is required")
	}
	if gopherImagePath == "" {
		flag.Usage()
		log.Fatal("Flag -gopher value is required")
	}

	if err := run(
		classifierFilePath,
		photoImagePath,
		photoDetectScale,
		photoDetectMinNeighbours,
		photoDetectMinSize,
		photoDetectMaxSize,
		gopherImagePath,
		gopherSizeCoeff,
		gopherXCoeff,
		gopherYCoeff,
		outputImagePath,
	); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// run runs the photo processing on input files and writes the result to the output file.
func run(
	classifierFilePath string,
	photoImagePath string,
	photoDetectScale float64,
	photoDetectMinNeighbours int,
	photoDetectMinSize int,
	photoDetectMaxSize int,
	gopherImagePath string,
	gopherSizeCoeff float64,
	gopherXCoeff float64,
	gopherYCoeff float64,
	outputImagePath string,
) error {
	rects, err := detectFace(
		classifierFilePath,
		photoImagePath,
		photoDetectScale,
		photoDetectMinNeighbours,
		photoDetectMinSize,
		photoDetectMaxSize,
	)
	if err != nil {
		return fmt.Errorf("detect face: %v", err)
	}

	photoImage, err := readImage(photoImagePath)
	if err != nil {
		return fmt.Errorf("read photo image: %v", err)
	}

	gopherImage, err := readImage(gopherImagePath)
	if err != nil {
		return fmt.Errorf("read gopher image: %v", err)
	}

	dst := image.NewRGBA(photoImage.Bounds())
	draw.Draw(dst, dst.Bounds(), photoImage, image.Point{}, draw.Over)

	for _, rect := range rects {
		size := int(float64(rect.Max.X-rect.Min.X) * gopherSizeCoeff)
		xAdj := int(float64(size) * gopherXCoeff)
		yAdj := int(float64(size) * gopherYCoeff)
		pt := image.Pt(-rect.Min.X+xAdj, -rect.Min.Y+yAdj)

		newImage := resize.Resize(uint(size), 0, gopherImage, resize.Lanczos3)

		draw.Draw(dst, dst.Bounds(), newImage, pt, draw.Over)
	}

	if err := writeOutputJpeg(outputImagePath, dst); err != nil {
		return fmt.Errorf("write output file: %v", err)
	}

	return nil
}

func readImage(fpath string) (image.Image, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("open file: %v", err)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode image: %v", err)
	}

	return img, nil
}

func writeOutputJpeg(fpath string, img image.Image) error {
	f, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("create file: %v", err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: jpeg.DefaultQuality}); err != nil {
		return fmt.Errorf("encode jpeg image: %v", err)
	}

	return nil
}

func detectFace(
	classifierFilePath string,
	photoFilePath string,
	photoDetectScale float64,
	photoDetectMinNeighbours int,
	photoDetectMinSize int,
	photoDetectMaxSize int,
) ([]image.Rectangle, error) {
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load(classifierFilePath) {
		return nil, fmt.Errorf("failed to read classifier file %s", classifierFilePath)
	}

	img := gocv.IMRead(photoFilePath, gocv.IMReadColor)
	if img.Empty() {
		return nil, fmt.Errorf("file %s is probably not an image", photoFilePath)
	}

	scale := photoDetectScale
	minNeigh := photoDetectMinNeighbours
	min := image.Pt(photoDetectMinSize, photoDetectMinSize)
	max := image.Pt(photoDetectMaxSize, photoDetectMaxSize)

	rects := classifier.DetectMultiScaleWithParams(img, scale, minNeigh, 0, min, max)
	return rects, nil
}
