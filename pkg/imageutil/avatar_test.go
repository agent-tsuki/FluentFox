package imageutil

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

var testLimits = AvatarLimits{MinSourcePx: 16, MaxSourcePx: 400, OutputPx: 64, JPEGQuality: 85}

func encodePNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeGIF(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gif.Encode(&buf, image.NewPaletted(image.Rect(0, 0, 32, 32), color.Palette{color.Black}), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestNormalizeAvatar(t *testing.T) {
	red := color.NRGBA{R: 255, A: 255}
	tests := []struct {
		name     string
		body     func(t *testing.T) []byte
		wantErr  error
		wantSide int
	}{
		{name: "wide image is cropped square and shrunk", body: func(t *testing.T) []byte { return encodePNG(t, 300, 200, red) }, wantSide: 64},
		{name: "small image is not enlarged", body: func(t *testing.T) []byte { return encodePNG(t, 40, 50, red) }, wantSide: 40},
		{name: "GIF is rejected", body: encodeGIF, wantErr: ErrUnsupportedType},
		{name: "text pretending to be an image is rejected", body: func(*testing.T) []byte { return []byte("<html><script>alert(1)</script>") }, wantErr: ErrUnsupportedType},
		{name: "image below the minimum is rejected", body: func(t *testing.T) []byte { return encodePNG(t, 10, 10, red) }, wantErr: ErrInvalidImage},
		{name: "image above the maximum is rejected", body: func(t *testing.T) []byte { return encodePNG(t, 500, 100, red) }, wantErr: ErrInvalidImage},
		{name: "truncated PNG is rejected", body: func(t *testing.T) []byte { return encodePNG(t, 100, 100, red)[:60] }, wantErr: ErrInvalidImage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := NormalizeAvatar(tt.body(t), testLimits)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			img, err := jpeg.Decode(bytes.NewReader(out))
			if err != nil {
				t.Fatalf("output is not a JPEG: %v", err)
			}
			if b := img.Bounds(); b.Dx() != tt.wantSide || b.Dy() != tt.wantSide {
				t.Errorf("size = %dx%d, want %dx%d", b.Dx(), b.Dy(), tt.wantSide, tt.wantSide)
			}
		})
	}
}

func TestNormalizeAvatar_FlattensTransparencyOntoWhite(t *testing.T) {
	out, err := NormalizeAvatar(encodePNG(t, 32, 32, color.NRGBA{}), testLimits)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(16, 16).RGBA()
	const nearWhite = 0xf000
	if r < nearWhite || g < nearWhite || b < nearWhite {
		t.Errorf("transparent pixel became %d,%d,%d, want white", r>>8, g>>8, b>>8)
	}
}
