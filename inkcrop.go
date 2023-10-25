package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/fsnotify/fsnotify"
	"github.com/makeworld-the-better-one/dither/v2"
)

// CalculateDimensions returns the most suitable dimensions for resizing the image.
func CalculateDimensions(currentSize image.Point, maxWidth float32, maxHeight float32) image.Point {
    sourceWidth := float32(currentSize.X)
    sourceHeight := float32(currentSize.Y)

    widthRatio := maxWidth / sourceWidth
    heightRatio := maxHeight / sourceHeight

    var destWidth, destHeight int
    if widthRatio < heightRatio {
        destWidth = int(sourceWidth * widthRatio)
        destHeight = int(sourceHeight * widthRatio)
    } else {
        destWidth = int(sourceWidth * heightRatio)
        destHeight = int(sourceHeight * heightRatio)
    }

    return image.Point{destWidth, destHeight}
}

func ditherImage(img image.Image, ditherAlg string, ditherStrength float32, ditherSerpentine bool) image.Image {
	// These are the colours we want in our output image
		palette := []color.Color{
				color.Black,
				color.White,
				// color.Gray16{
				// 		Y: uint16(65535 * ditherStrength),
				// },
				color.Gray16{},
				color.Transparent,
		}

		// Create ditherer
		d := dither.NewDitherer(palette)

		// lowercase the ditherAlg
		ditherAlg = strings.ToLower(ditherAlg)

		d.Serpentine = ditherSerpentine

		switch ditherAlg {
					case "floydsteinberg":
						d.Matrix = dither.ErrorDiffusionStrength(dither.FloydSteinberg, ditherStrength)
					case "jarvisjudiceninke":
						d.Matrix = dither.ErrorDiffusionStrength(dither.JarvisJudiceNinke, ditherStrength)
					case "stucki":
						d.Matrix = dither.ErrorDiffusionStrength(dither.Stucki, ditherStrength)
					case "atkinson":
						d.Matrix = dither.ErrorDiffusionStrength(dither.Atkinson, ditherStrength)
					case "sierra":
						d.Matrix = dither.ErrorDiffusionStrength(dither.Sierra, ditherStrength)
					case "sierra2":
						d.Matrix = dither.ErrorDiffusionStrength(dither.Sierra2, ditherStrength)
					case "sierralite":
						d.Matrix = dither.ErrorDiffusionStrength(dither.SierraLite, ditherStrength)
					case "stevenpigeon":
						d.Matrix = dither.ErrorDiffusionStrength(dither.StevenPigeon, ditherStrength)
					case "burkes":
						d.Matrix = dither.ErrorDiffusionStrength(dither.Burkes, ditherStrength)
					case "falsefloydsteinberg":
						d.Matrix = dither.ErrorDiffusionStrength(dither.FalseFloydSteinberg, ditherStrength)
			} // see https://github.com/makew0rld/dither for more


		// Dither the image, attempting to modify the existing image
		// If it can't then a dithered copy will be returned.
		img = d.Dither(img)

		return img
}

func watcherDaemon(input string, matches []string, outdir string, dither *bool, ditherAlg *string, ditherStrength float32, ditherSerpentine *bool, rotate *bool, crop *bool, quality *int) {
	// if the input is a file / glob pattern, use the directory name
	if strings.Contains(input, "*") {
		input = input[:strings.LastIndex(input, "/")]
	}

	log.Printf("Monitoring %s for new images\n", input)

	// Use fsnotify to monitor the input directory for new images
	// if a new image is added, simply echo the filename for now
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer fsWatcher.Close()

	err = fsWatcher.Add(input)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-fsWatcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("New image:", event.Name)
				matches, err := filepath.Glob(event.Name)
				if err != nil {
					log.Fatal(err)
				}
				processImages(&input, matches, outdir, dither, ditherAlg, ditherStrength, ditherSerpentine, rotate, crop, quality)
			}

		case err, ok := <-fsWatcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}

		// TODO: some logic here to keep the fsWatcher alive as a daemon

	}
}

// a function that symlinks an image to another name, sleeps for a given time, then links the next image, etc.
// it assumes the images have already been processed
func slideShowDaemon(input string, matches []string, outdir string, linkTimer int) {
	// symlink the first one to a file called "linkedimage.jpg" in output directory
	// sleep for 5 seconds
	// symlink the next one to a file called "linkedimage.jpg" in output directory
	// etc.

	if strings.Contains(input, "*") {
		input = input[:strings.LastIndex(input, "/")]
	}

	log.Printf("Monitoring %s for new images\n", input)

	images := make(chan string) // channel of strings

	go func() {
		for {
			for _, filepath := range matches {
				images <- filepath
			}
		}
	} ()

	for filepath := range images {
		// if the symlink already exists, remove it
		if _, err := os.Stat(outdir + "/linkedimage.jpg"); err == nil {
			err = os.Remove(outdir + "/linkedimage.jpg")
			if err != nil {
				log.Fatal(err)
			}
		}
		// symlink the file to linkedimage.jpg
		err := os.Symlink(filepath, outdir + "/linkedimage.jpg")
		if err != nil {
			log.Fatal(err)
		}

		linkTimer := time.Duration(linkTimer)
		time.Sleep(linkTimer * time.Second)

		// remove the symlink
		err = os.Remove(outdir + "/linkedimage.jpg")
		if err != nil {
			log.Fatal(err)
		}
	}
}


func processImages(input *string, matches []string, outdir string, dither *bool, ditherAlg *string, ditherStrength float32, ditherSerpentine *bool, rotate *bool, crop *bool, quality *int) {

	log.Printf("Processing images from %s to %s\n", *input, outdir)

	for _, filepath := range matches {
		ext := strings.ToLower(filepath[strings.LastIndex(filepath, "."):])
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			log.Printf("%s is not a supported image format\n", filepath)
			continue
		}

		baseName := strings.TrimSuffix(filepath, ext)
		baseName = strings.ReplaceAll(baseName, "_", "-")

		file, err := os.Open(filepath)
		if err != nil {
			log.Fatal(err)
		}

		var img image.Image
		if ext == ".jpg" || ext == ".jpeg" {
			// Decode JPEG images
			img, err = jpeg.Decode(file)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// Decode PNG images and convert to JPEG
			pngImg, err := png.Decode(file)
			if err != nil {
				log.Fatal(err)
			}
			img = pngImg
		}

		if *dither {
			// Dither the image
			img = ditherImage(img, *ditherAlg, ditherStrength, *ditherSerpentine)
		}
		if *rotate {
			// Rotate the image 90 degrees clockwise
			img = imaging.Rotate90(img)
		}
		if *crop {
			// Crop the image to 960x540
			img = imaging.CropCenter(img, 960, 540)
		}

		file.Close()

		if img.Bounds().Max.Y > img.Bounds().Max.X {
			log.Printf("Image wider than higher, rotating 90 degrees!")
			img = imaging.Rotate(img, 90.0, color.Gray{})
		}

		newSize := CalculateDimensions(img.Bounds().Max, 960.0, 540.0)

		m := imaging.Resize(img, newSize.X, newSize.Y, imaging.Lanczos)
		width := newSize.X
		height := newSize.Y

		log.Printf("Resized from %dx%d to %dx%d\n", img.Bounds().Max.X, img.Bounds().Max.Y, width, height)

		offsetX := int((960 - width) / 2.0)
		offsetY := int((540 - height) / 2.0)

		// if a ditherer was used add the algorithm to the filename
		if *dither {
			baseName = baseName + "-" + *ditherAlg
		}

		filename := fmt.Sprintf("%s_%dx%d_%dx%d_resized.jpg", baseName, width, height, offsetX, offsetY)
		// Concatenate output directory and filename using string concatenation
		outpath := outdir + "/" + filename

		// if the destination file already exists, skip it
		if _, err := os.Stat(outpath); err == nil {
			log.Printf("File %s already exists, skipping\n", outpath)
			continue
		}

		out, err := os.Create(outpath)
		if err != nil {
			log.Fatal(err)
		}

		err = jpeg.Encode(out, m, &jpeg.Options{Quality: *quality})
		if err != nil {
			log.Fatal(err)
		}

		out.Close()

		log.Printf("Created new image %s", outpath)
	}
}

func main() {
	input := flag.String("input", "*.jp*g", "input file or glob pattern")
	output := flag.String("output", "output", "output directory")
	dither := flag.Bool("dither", true, "dither the image")
	ditherAlg := flag.String("ditherAlg", "StevenPigeon", "dithering algorithm to use (see makew0rld/dither)")
	ditherAll := flag.Bool("ditherAll", false, "dither each image with all algorithms")
	ditherStrength64 := flag.Float64("ditherStrength", 0.9, "dithering strength (0-1)")
	ditherSerpentine := flag.Bool("ditherSerpentine", false, "enable Serpentine dithering")
	rotate := flag.Bool("rotate", false, "rotate the image 90 degrees clockwise")
	crop := flag.Bool("crop", false, "crop the image to 960x540")
	quality := flag.Int("quality", 80, "set the JPEG quality 0-100 (%)")
	daemon := flag.Bool("daemon", false, "run as a daemon monitoring the input directory for new images")
	link := flag.Bool("link", false, "run as a daemon linking the input directory for new images")
	linkTimer := flag.Int("link-timer", 900, "time between relinking images in seconds")
	help := flag.Bool("help", false, "print usage")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	matches, err := filepath.Glob(*input)
	if err != nil {
		log.Fatal(err)
	}

	outdir := *output
	// if the output directory doesn't exist, create it
	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		err = os.Mkdir(outdir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	// convert ditherStrength to float32
	ditherStrength := float32(*ditherStrength64)

	if *ditherAll {
		ditherAlgs := []string{"FloydSteinberg", "JarvisJudiceNinke", "Stucki", "Atkinson", "Sierra", "Sierra2", "SierraLite", "StevenPigeon", "Burkes", "FalseFloydSteinberg"}
		for _, ditherAlg := range ditherAlgs {
			ditherAlg := ditherAlg
			processImages(input, matches, outdir, dither, &ditherAlg, ditherStrength, ditherSerpentine, rotate, crop, quality)
		}
	} else {
		if *daemon {
			input := *input
			watcherDaemon(input, matches, outdir, dither, ditherAlg, ditherStrength, ditherSerpentine, rotate, crop, quality)
		} else if *link {
			input := *input
			linkTimer := *linkTimer
			slideShowDaemon(input, matches, outdir, linkTimer)
		} else {
			processImages(input, matches, outdir, dither, ditherAlg, ditherStrength, ditherSerpentine, rotate, crop, quality)
		}
	}
}
