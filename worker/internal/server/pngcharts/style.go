package pngcharts

import (
	"image/color"
	"strconv"

	xfont "golang.org/x/image/font"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/font/liberation"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
)

func init() {
	font.DefaultCache.Add(liberation.Collection())
}

func hexColor(hex string) color.Color {
	value, err := strconv.ParseUint(hex[1:], 16, 32)
	if err != nil {
		return color.Black
	}
	return color.RGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 255}
}

func face(size float64, bold bool) font.Font {
	weight := xfont.WeightNormal
	if bold {
		weight = xfont.WeightBold
	}
	return font.Font{Typeface: "Liberation", Variant: "Sans", Weight: weight, Size: vg.Points(size)}
}

func textStyle(size float64, textColor color.Color, bold bool, horizontal text.XAlignment) text.Style {
	return text.Style{Color: textColor, Font: face(size, bold), XAlign: horizontal, YAlign: text.YBottom, Handler: plot.DefaultTextHandler}
}
