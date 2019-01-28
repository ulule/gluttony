package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/discordapp/lilliput"
	"github.com/pkg/errors"
)

var EncodeOptions = map[string]map[int]int{
	".jpeg": map[int]int{lilliput.JpegQuality: 85},
	".png":  map[int]int{lilliput.PngCompression: 7},
	".webp": map[int]int{lilliput.WebpQuality: 85},
}

func resize(inputBuf []byte, outputWidth int, outputHeight int, outputFilename string, stretch bool) error {
	decoder, err := lilliput.NewDecoder(inputBuf)
	// this error reflects very basic checks,
	// mostly just for the magic bytes of the file to match known image formats
	if err != nil {
		return errors.Wrapf(err, "error decoding image")
	}
	defer decoder.Close()

	header, err := decoder.Header()
	// this error is much more comprehensive and reflects
	// format errors
	if err != nil {
		return errors.Wrapf(err, "error reading image header")
	}

	// print some basic info about the image
	fmt.Printf("file type: %s\n", decoder.Description())
	fmt.Printf("%dpx x %dpx\n", header.Width(), header.Height())

	if decoder.Duration() != 0 {
		log.Printf("duration: %.2f s\n", float64(decoder.Duration())/float64(time.Second))
	}

	// get ready to resize image,
	// using 8192x8192 maximum resize buffer size
	ops := lilliput.NewImageOps(8192)
	defer ops.Close()

	// create a buffer to store the output image, 50MB in this case
	outputImg := make([]byte, 50*1024*1024)

	// use user supplied filename to guess output type if provided
	// otherwise don't transcode (use existing type)
	outputType := "." + strings.ToLower(decoder.Description())
	if outputFilename != "" {
		outputType = filepath.Ext(outputFilename)
	}

	if outputWidth == 0 {
		outputWidth = header.Width()
	}

	if outputHeight == 0 {
		outputHeight = header.Height()
	}

	resizeMethod := lilliput.ImageOpsFit
	if stretch {
		resizeMethod = lilliput.ImageOpsResize
	}

	opts := &lilliput.ImageOptions{
		FileType:             outputType,
		Width:                outputWidth,
		Height:               outputHeight,
		ResizeMethod:         resizeMethod,
		NormalizeOrientation: true,
		EncodeOptions:        EncodeOptions[outputType],
	}

	// resize and transcode image
	outputImg, err = ops.Transform(decoder, opts, outputImg)
	if err != nil {
		return errors.Wrapf(err, "error transforming image")
	}

	if _, err := os.Stat(outputFilename); !os.IsNotExist(err) {
		log.Printf("output filename %s exists, removing\n", outputFilename)
		err = os.Remove(outputFilename)
		if err != nil {
			return errors.Wrapf(err, "error removing %s", outputFilename)
		}
	}

	err = ioutil.WriteFile(outputFilename, outputImg, 0400)
	if err != nil {
		return errors.Wrapf(err, "error writing out resized image")
	}

	log.Printf("image written to %s\n", outputFilename)

	return nil
}

func main() {
	var inputFilename string
	var outputWidth int
	var outputHeight int
	var outputFilename string
	var stretch bool
	var iteration int

	flag.StringVar(&inputFilename, "input", "", "name of input file to resize/transcode")
	flag.StringVar(&outputFilename, "output", "", "name of output file, also determines output type")
	flag.IntVar(&outputWidth, "width", 0, "width of output file")
	flag.IntVar(&outputHeight, "height", 0, "height of output file")
	flag.IntVar(&iteration, "iteration", 1, "number of iteration")
	flag.BoolVar(&stretch, "stretch", false, "perform stretching resize instead of cropping")
	flag.Parse()

	if inputFilename == "" {
		fmt.Printf("No input filename provided, quitting.\n")
		flag.Usage()
		os.Exit(1)
	}

	// decoder wants []byte, so read the whole file into a buffer
	inputBuf, err := ioutil.ReadFile(inputFilename)
	if err != nil {
		fmt.Printf("failed to read input file, %s\n", err)
		os.Exit(1)
	}

	// image has been resized, now write file out
	if outputFilename == "" {
		outputFilename = "resized" + filepath.Ext(inputFilename)
	}

	for iteration > 0 {
		err := resize(inputBuf, outputWidth, outputHeight, outputFilename, stretch)

		if err != nil {
			log.Fatal(err)
		}

		iteration -= 1
	}
}
