// Package imageutil provides common image manipulation and drawing utilities.
// It includes functions for drawing shapes, handling colors, and other
// image processing operations used across captcha generation modules.
package imageutil

import (
	"image"
	"image/color"
	"image/draw"
)

// SolidColor represents a uniform color that implements the image.Color interface.
type SolidColor struct {
	R, G, B, A uint8
}

// RGBA returns the alpha-premultiplied color values.
func (sc *SolidColor) RGBA() (r, g, b, a uint32) {
	return uint32(sc.R) * 0x101, uint32(sc.G) * 0x101, uint32(sc.B) * 0x101, uint32(sc.A) * 0x101
}

// ColorModel returns the color model for this color.
func (sc *SolidColor) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds returns the bounds of the color pattern.
func (sc *SolidColor) Bounds() image.Rectangle {
	return image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 0xffffffff, Y: 0xffffffff}}
}

// At returns the color at the given point.
func (sc *SolidColor) At(x, y int) color.Color {
	return sc
}

// NewSolidColor creates a new SolidColor with the given RGBA values.
func NewSolidColor(r, g, b, a uint8) *SolidColor {
	return &SolidColor{R: r, G: g, B: b, A: a}
}

// DrawRect draws a hollow rectangle outline with the specified color.
func DrawRect(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	for i := x; i < x+w && i < img.Bounds().Dx(); i++ {
		for j := y; j < y+h && j < img.Bounds().Dy(); j++ {
			if i >= 0 && j >= 0 {
				img.Set(i, j, col)
			}
		}
	}
}

// DrawPixelSafe draws a single pixel at (x, y) only if the coordinates are within image bounds.
func DrawPixelSafe(img *image.RGBA, x, y int, col color.RGBA) {
	if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
		img.Set(x, y, col)
	}
}

// DrawLine draws a line from (x1, y1) to (x2, y2) using the Bresenham algorithm.
func DrawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA) {
	dx := absInt(x2 - x1)
	dy := absInt(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy

	for {
		DrawPixelSafe(img, x1, y1, col)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// DrawCircle draws a filled circle with the center at (cx, cy) and the given radius.
func DrawCircle(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	x, y, d := 0, radius, 3-2*radius
	for x <= y {
		DrawPixelSafe(img, cx+x, cy+y, col)
		DrawPixelSafe(img, cx+y, cy+x, col)
		DrawPixelSafe(img, cx-y, cy+x, col)
		DrawPixelSafe(img, cx-x, cy+y, col)
		DrawPixelSafe(img, cx+x, cy-y, col)
		DrawPixelSafe(img, cx+y, cy-x, col)
		DrawPixelSafe(img, cx-y, cy-x, col)
		DrawPixelSafe(img, cx-x, cy-y, col)
		if d < 0 {
			d = d + 4*x + 6
		} else {
			d = d + 4*(x-y) + 10
			y--
		}
		x++
	}
}

// DrawFilledRect draws a filled rectangle with the specified color.
func DrawFilledRect(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &SolidColor{col.R, col.G, col.B, col.A}, image.ZP, draw.Src)
}

// FillBackground fills the entire image with a solid background color.
func FillBackground(img *image.RGBA, r, g, b uint8) {
	bgColor := &SolidColor{R: r, G: g, B: b, A: 255}
	draw.Draw(img, img.Bounds(), bgColor, image.ZP, draw.Src)
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
