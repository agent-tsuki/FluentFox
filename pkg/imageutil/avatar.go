// Package imageutil re-encodes user images. Decoding and encoding again keeps
// only pixels: EXIF (including GPS) and anything hidden in the file is dropped.
package imageutil

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder for image.Decode
	"net/http"
)

var (
	ErrUnsupportedType = errors.New("imageutil: not a JPEG or PNG image")
	ErrInvalidImage    = errors.New("imageutil: image is corrupt or its size is out of range")
)

var allowedTypes = map[string]bool{"image/jpeg": true, "image/png": true}

type AvatarLimits struct {
	MinSourcePx int
	MaxSourcePx int
	OutputPx    int
	JPEGQuality int
}

// NormalizeAvatar returns a square JPEG of at most OutputPx a side. The type
// is sniffed from the bytes, never taken from the client's Content-Type.
func NormalizeAvatar(body []byte, limits AvatarLimits) ([]byte, error) {
	if !allowedTypes[http.DetectContentType(body)] {
		return nil, ErrUnsupportedType
	}
	// Checked before a full decode so a small file claiming huge dimensions
	// (a decompression bomb) is rejected without allocating the pixels.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return nil, ErrInvalidImage
	}
	if cfg.Width < limits.MinSourcePx || cfg.Height < limits.MinSourcePx ||
		cfg.Width > limits.MaxSourcePx || cfg.Height > limits.MaxSourcePx {
		return nil, ErrInvalidImage
	}
	src, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, ErrInvalidImage
	}

	side := min(cfg.Width, cfg.Height)
	out := min(side, limits.OutputPx)
	square := centerSquare(src.Bounds(), side)
	dst := downscale(src, square, out)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: limits.JPEGQuality}); err != nil {
		return nil, fmt.Errorf("imageutil: encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func centerSquare(b image.Rectangle, side int) image.Rectangle {
	x := b.Min.X + (b.Dx()-side)/2
	y := b.Min.Y + (b.Dy()-side)/2
	return image.Rect(x, y, x+side, y+side)
}

// downscale averages every source pixel that falls in each output pixel (an
// area filter), which stays sharp without aliasing for shrink-only resizes.
// Transparent areas are flattened onto white, since JPEG has no alpha.
func downscale(src image.Image, area image.Rectangle, out int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, out, out))
	side := area.Dx()
	for oy := range out {
		y0, y1 := area.Min.Y+oy*side/out, area.Min.Y+(oy+1)*side/out
		for ox := range out {
			x0, x1 := area.Min.X+ox*side/out, area.Min.X+(ox+1)*side/out
			dst.SetRGBA(ox, oy, averageOnWhite(src, x0, y0, max(x1, x0+1), max(y1, y0+1)))
		}
	}
	return dst
}

const (
	maxChannel = 0xffff
	opaque     = 0xff
	// color.Color channels are 16-bit; RGBA wants 8.
	channelShift = 8
)

func averageOnWhite(src image.Image, x0, y0, x1, y1 int) color.RGBA {
	var r, g, b, n uint64
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cr, cg, cb, ca := src.At(x, y).RGBA()
			// Colours are alpha-premultiplied, so adding the missing alpha as white flattens them.
			white := uint64(maxChannel - ca)
			r += uint64(cr) + white
			g += uint64(cg) + white
			b += uint64(cb) + white
			n++
		}
	}
	return color.RGBA{R: to8Bit(r / n), G: to8Bit(g / n), B: to8Bit(b / n), A: opaque}
}

// An average of 16-bit channels never exceeds 0xffff, so the shift fits a byte.
func to8Bit(v uint64) uint8 {
	return uint8(min(v, maxChannel) >> channelShift) //nolint:gosec // bounded by min above
}
