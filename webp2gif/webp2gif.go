package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
	"slices"
	"strings"

	//"github.com/chai2010/webp"
	"github.com/gen2brain/webp"
	"github.com/kazu/cmd/ext"
	"github.com/schollz/progressbar/v3"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: webp_to_gif input.webp output.gif")
		return
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]

	if !strings.HasSuffix(strings.ToLower(inputPath), ".webp") {
		fmt.Println("Input file must have .webp extension")
		return
	}

	// Open the WebP file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		return
	}
	defer inputFile.Close()

	// Create the GIF file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	// Prepare GIF frames

	var g gif.GIF

	wp, err := webp.DecodeAll(inputFile)
	if err != nil {
		fmt.Println("Error decoding WebP image:", err)
		return
	}

	bar := progressbar.Default(int64(len(wp.Image)), "converting")

	// for i := range wp.Image {
	// 	pal := toPaletted(wp.Image[i])
	// 	g.Image = append(g.Image, pal)
	// 	g.Delay = append(g.Delay, wp.Delay[i])
	// }
	g.Image = make([]*image.Paletted, len(wp.Image))
	g.Delay = make([]int, len(wp.Image))

	ext.Parallel(slices.Values(wp.Image),
		func(i int, img image.Image) {
			pal := toPaletted(img)
			g.Image[i] = pal
			g.Delay[i] = wp.Delay[i]
			bar.Add(1)
		}, ext.Concurrent(10))

	//err = gif.Encode(outputFile, img, nil)
	// Encode the GIF
	err = gif.EncodeAll(outputFile, &g)
	if err != nil {
		fmt.Println("Error encoding GIF:", err)
		return
	}

	fmt.Println("Successfully converted", inputPath, "to", outputPath)
}

func toPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	palettedImg := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(palettedImg, bounds, img, image.Point{})
	return palettedImg
}
