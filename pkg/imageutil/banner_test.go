package imageutil

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"testing"
)

func TestNormalizeBanner(t *testing.T) {
	limits := BannerLimits{MinWidthPx: 32, MaxSourcePx: 400, MaxWidthPx: 100, JPEGQuality: 85}
	blue := color.NRGBA{B: 255, A: 255}
	tests := []struct {
		name    string
		body    func(t *testing.T) []byte
		wantErr error
		wantW   int
		wantH   int
	}{
		{name: "wide image keeps its aspect when shrunk", body: func(t *testing.T) []byte { return encodePNG(t, 400, 100, blue) }, wantW: 100, wantH: 25},
		{name: "narrow image is not enlarged", body: func(t *testing.T) []byte { return encodePNG(t, 60, 30, blue) }, wantW: 60, wantH: 30},
		{name: "too narrow is rejected", body: func(t *testing.T) []byte { return encodePNG(t, 20, 20, blue) }, wantErr: ErrInvalidImage},
		{name: "too tall is rejected", body: func(t *testing.T) []byte { return encodePNG(t, 100, 500, blue) }, wantErr: ErrInvalidImage},
		{name: "GIF is rejected", body: encodeGIF, wantErr: ErrUnsupportedType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := NormalizeBanner(tt.body(t), limits)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
			if err != nil || format != "jpeg" {
				t.Fatalf("output is not a JPEG: %v %q", err, format)
			}
			if cfg.Width != tt.wantW || cfg.Height != tt.wantH {
				t.Errorf("size = %dx%d, want %dx%d", cfg.Width, cfg.Height, tt.wantW, tt.wantH)
			}
		})
	}
}
