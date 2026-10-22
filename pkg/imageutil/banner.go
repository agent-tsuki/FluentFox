package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
)

type BannerLimits struct {
	MinWidthPx  int
	MaxSourcePx int
	MaxWidthPx  int
	JPEGQuality int
}

// NormalizeBanner returns a JPEG that keeps the source's aspect ratio and is
// at most MaxWidthPx wide. Like NormalizeAvatar it sniffs the type and checks
// the size before decoding the pixels.
func NormalizeBanner(body []byte, limits BannerLimits) ([]byte, error) {
	if !allowedTypes[http.DetectContentType(body)] {
		return nil, ErrUnsupportedType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return nil, ErrInvalidImage
	}
	if cfg.Width < limits.MinWidthPx || cfg.Height < 1 ||
		cfg.Width > limits.MaxSourcePx || cfg.Height > limits.MaxSourcePx {
		return nil, ErrInvalidImage
	}
	src, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, ErrInvalidImage
	}

	outW := min(cfg.Width, limits.MaxWidthPx)
	outH := max(cfg.Height*outW/cfg.Width, 1)
	dst := downscaleRect(src, src.Bounds(), outW, outH)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: limits.JPEGQuality}); err != nil {
		return nil, fmt.Errorf("imageutil: encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

// downscaleRect is downscale for a non-square output: the same area filter,
// with separate horizontal and vertical steps.
func downscaleRect(src image.Image, area image.Rectangle, outW, outH int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	w, h := area.Dx(), area.Dy()
	for oy := range outH {
		y0, y1 := area.Min.Y+oy*h/outH, area.Min.Y+(oy+1)*h/outH
		for ox := range outW {
			x0, x1 := area.Min.X+ox*w/outW, area.Min.X+(ox+1)*w/outW
			dst.SetRGBA(ox, oy, averageOnWhite(src, x0, y0, max(x1, x0+1), max(y1, y0+1)))
		}
	}
	return dst
}
