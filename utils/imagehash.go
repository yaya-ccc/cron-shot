package utils

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LatestPNG struct {
	Path    string
	ModTime time.Time
	Image   image.Image
}

// AHash16x16 generates a 16x16 average hash.
func AHash16x16(img *image.RGBA) []byte {
	w, h := 16, 16
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	if srcW == 0 || srcH == 0 {
		return make([]byte, 32)
	}

	stepX := float64(srcW) / float64(w)
	stepY := float64(srcH) / float64(h)
	gray := make([]float64, w*h)

	idx := 0
	var sum float64
	for y := 0; y < h; y++ {
		sy := int(float64(y)*stepY + stepY/2)
		if sy >= srcH {
			sy = srcH - 1
		}
		for x := 0; x < w; x++ {
			sx := int(float64(x)*stepX + stepX/2)
			if sx >= srcW {
				sx = srcW - 1
			}
			o := img.RGBAAt(img.Bounds().Min.X+sx, img.Bounds().Min.Y+sy)
			g := 0.2126*float64(o.R) + 0.7152*float64(o.G) + 0.0722*float64(o.B)
			gray[idx] = g
			sum += g
			idx++
		}
	}

	avg := sum / float64(w*h)
	bits := make([]byte, 32)
	for i := 0; i < w*h; i++ {
		if gray[i] >= avg {
			bits[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return bits
}

// Hamming256 returns the Hamming distance for two 256-bit hashes.
func Hamming256(a, b []byte) int {
	if len(a) != 32 || len(b) != 32 {
		return 256
	}
	dist := 0
	for i := 0; i < 32; i++ {
		x := a[i] ^ b[i]
		x = (x & 0x55) + ((x >> 1) & 0x55)
		x = (x & 0x33) + ((x >> 2) & 0x33)
		x = (x & 0x0F) + ((x >> 4) & 0x0F)
		dist += int(x)
	}
	return dist
}

// AHashFromImage generates a 16x16 average hash for any image.Image.
func AHashFromImage(img image.Image) []byte {
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, img, b.Min, draw.Src)
	return AHash16x16(rgba)
}

// LatestPNGImage returns the latest PNG image in a directory.
func LatestPNGImage(dir string) (image.Image, error) {
	latest, err := LoadLatestPNG(dir)
	if err != nil || latest == nil {
		return nil, err
	}
	return latest.Image, nil
}

// LoadLatestPNG returns the latest PNG image in a directory with metadata.
func LoadLatestPNG(dir string) (*LatestPNG, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var latestPath string
	var latestMod time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".png") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latestPath = filepath.Join(dir, entry.Name())
		}
	}

	if latestPath == "" {
		return nil, nil
	}

	f, err := os.Open(latestPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return &LatestPNG{Path: latestPath, ModTime: latestMod, Image: img}, nil
}

// ImagesEqualExact compares two images pixel-by-pixel.
func ImagesEqualExact(a *image.RGBA, b image.Image) bool {
	bounds := b.Bounds()
	if a.Bounds().Dx() != bounds.Dx() || a.Bounds().Dy() != bounds.Dy() {
		return false
	}

	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, b, bounds.Min, draw.Src)
	ap := a.Pix
	bp := rgba.Pix
	if len(ap) != len(bp) {
		return false
	}
	for i := 0; i < len(ap); i++ {
		if ap[i] != bp[i] {
			return false
		}
	}
	return true
}
