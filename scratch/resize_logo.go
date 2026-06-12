package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
)

type rgba struct {
	r, g, b, a float64
}

func main() {
	outputPath := filepath.Clean("../logo/logo_small.go")
	outFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	fmt.Fprintln(outFile, "package logo")
	fmt.Fprintln(outFile)

	// 1. Process original logo (git-userhub-logo.png)
	originalPath := filepath.Clean("../logo/git-userhub-logo.png")
	if _, err := os.Stat(originalPath); err == nil {
		if err := processImage(originalPath, "SmallPixelLines", outFile); err != nil {
			fmt.Printf("Error processing original logo: %v\n", err)
		}
	} else {
		fmt.Fprintln(outFile, "var SmallPixelLines = []string{}")
	}

	fmt.Fprintln(outFile)

	// 2. Process new logo (logo.png)
	newPath := filepath.Clean("../logo/logo.png")
	if err := processImage(newPath, "NewSmallPixelLines", outFile); err != nil {
		fmt.Printf("Error processing new logo: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully generated logo/logo_small.go with both SmallPixelLines and NewSmallPixelLines")
}

func processImage(imagePath, varName string, w io.Writer) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	srcWidth := bounds.Max.X - bounds.Min.X
	srcHeight := bounds.Max.Y - bounds.Min.Y

	targetWidth := 28
	targetHeight := int(float64(targetWidth) * (float64(srcHeight) / float64(srcWidth)))
	if targetHeight%2 != 0 {
		targetHeight++
	}

	fmt.Fprintf(w, "// %s is the downscaled pixel-art logo from %s (%dx%d cells).\n", varName, filepath.Base(imagePath), targetWidth, targetHeight/2)
	fmt.Fprintf(w, "var %s = []string{\n", varName)

	for ty := 0; ty < targetHeight; ty += 2 {
		var rowStr string
		for tx := 0; tx < targetWidth; tx++ {
			cTop := bilinearSample(img, tx, ty, targetWidth, targetHeight, srcWidth, srcHeight)
			cBottom := bilinearSample(img, tx, ty+1, targetWidth, targetHeight, srcWidth, srcHeight)
			rowStr += formatHalfBlock(cTop, cBottom)
		}
		fmt.Fprintf(w, "\t%q,\n", rowStr)
	}

	fmt.Fprintln(w, "}")
	return nil
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func bilinearSample(img image.Image, tx, ty, targetW, targetH, srcW, srcH int) color.Color {
	gx := (float64(tx) + 0.5) * (float64(srcW) / float64(targetW)) - 0.5
	gy := (float64(ty) + 0.5) * (float64(srcH) / float64(targetH)) - 0.5

	gxi := int(gx)
	gyi := int(gy)
	xf := gx - float64(gxi)
	yf := gy - float64(gyi)

	x0 := clamp(gxi, 0, srcW-1)
	x1 := clamp(gxi+1, 0, srcW-1)
	y0 := clamp(gyi, 0, srcH-1)
	y1 := clamp(gyi+1, 0, srcH-1)

	r00, g00, b00, a00 := img.At(x0, y0).RGBA()
	r10, g10, b10, a10 := img.At(x1, y0).RGBA()
	r01, g01, b01, a01 := img.At(x0, y1).RGBA()
	r11, g11, b11, a11 := img.At(x1, y1).RGBA()

	c00 := rgba{float64(r00 >> 8), float64(g00 >> 8), float64(b00 >> 8), float64(a00 >> 8)}
	c10 := rgba{float64(r10 >> 8), float64(g10 >> 8), float64(b10 >> 8), float64(a10 >> 8)}
	c01 := rgba{float64(r01 >> 8), float64(g01 >> 8), float64(b01 >> 8), float64(a01 >> 8)}
	c11 := rgba{float64(r11 >> 8), float64(g11 >> 8), float64(b11 >> 8), float64(a11 >> 8)}

	r0 := c00.r*(1-xf) + c10.r*xf
	g0 := c00.g*(1-xf) + c10.g*xf
	b0 := c00.b*(1-xf) + c10.b*xf
	a0 := c00.a*(1-xf) + c10.a*xf

	r1 := c01.r*(1-xf) + c11.r*xf
	g1 := c01.g*(1-xf) + c11.g*xf
	b1 := c01.b*(1-xf) + c11.b*xf
	a1 := c01.a*(1-xf) + c11.a*xf

	r := r0*(1-yf) + r1*yf
	g := g0*(1-yf) + g1*yf
	b := b0*(1-yf) + b1*yf
	a := a0*(1-yf) + a1*yf

	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
}

func formatHalfBlock(top, bottom color.Color) string {
	tr, tg, tb, ta := top.RGBA()
	br, bg, bb, ba := bottom.RGBA()

	tRed, tGreen, tBlue, tAlpha := uint8(tr>>8), uint8(tg>>8), uint8(tb>>8), uint8(ta>>8)
	bRed, bGreen, bBlue, bAlpha := uint8(br>>8), uint8(bg>>8), uint8(bb>>8), uint8(ba>>8)

	const alphaThreshold = 20
	topIsTransparent := tAlpha < alphaThreshold
	bottomIsTransparent := bAlpha < alphaThreshold

	if topIsTransparent && bottomIsTransparent {
		return " "
	}

	if topIsTransparent {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm▄\x1b[0m", bRed, bGreen, bBlue)
	}

	if bottomIsTransparent {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm▀\x1b[0m", tRed, tGreen, tBlue)
	}

	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm\x1b[38;2;%d;%d;%dm▄\x1b[0m", tRed, tGreen, tBlue, bRed, bGreen, bBlue)
}
