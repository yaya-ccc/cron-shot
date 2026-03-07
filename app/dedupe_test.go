package app

import (
	"cron-shot/config"
	"cron-shot/utils"
	"image"
	"image/color"
	"testing"
)

func TestShouldSkipDueToDedupeUsesCacheAfterFirstLoad(t *testing.T) {
	prepareDedupeTest(t, true, 90)

	loads := 0
	cached := solidRGBA(color.RGBA{R: 40, G: 90, B: 120, A: 255})
	hash := utils.AHash16x16(cached)
	loadLatestDedupeEntry = func(dir string) (*dedupeCacheEntry, error) {
		loads++
		return &dedupeCacheEntry{
			path: "latest.png",
			hash: append([]byte(nil), hash...),
			img:  cloneRGBA(cached),
		}, nil
	}

	root := t.TempDir()
	if !ShouldSkipDueToDedupe(cloneRGBA(cached), root, "proc.exe", "", "folder") {
		t.Fatal("expected first comparison to skip due to matching cached image")
	}
	if !ShouldSkipDueToDedupe(cloneRGBA(cached), root, "proc.exe", "", "folder") {
		t.Fatal("expected second comparison to skip due to in-memory cache")
	}
	if loads != 1 {
		t.Fatalf("expected loader to run once, got %d", loads)
	}
}

func TestRememberSavedCaptureUpdatesCache(t *testing.T) {
	prepareDedupeTest(t, true, 100)

	loads := 0
	loadLatestDedupeEntry = func(dir string) (*dedupeCacheEntry, error) {
		loads++
		return nil, nil
	}

	root := t.TempDir()
	img := solidRGBA(color.RGBA{R: 200, G: 10, B: 10, A: 255})
	if ShouldSkipDueToDedupe(img, root, "proc.exe", "", "folder") {
		t.Fatal("did not expect skip without prior saved image")
	}

	RememberSavedCapture(root, "proc.exe", "", "folder", "saved.png", img)

	if !ShouldSkipDueToDedupe(cloneRGBA(img), root, "proc.exe", "", "folder") {
		t.Fatal("expected cached saved image to be used for exact dedupe")
	}
	if loads != 1 {
		t.Fatalf("expected loader to run once before cache update, got %d", loads)
	}
}

func TestShouldSkipDueToDedupeDisabledDoesNotLoad(t *testing.T) {
	prepareDedupeTest(t, false, 100)

	loads := 0
	loadLatestDedupeEntry = func(dir string) (*dedupeCacheEntry, error) {
		loads++
		return nil, nil
	}

	if ShouldSkipDueToDedupe(solidRGBA(color.RGBA{R: 1, G: 2, B: 3, A: 255}), t.TempDir(), "proc.exe", "", "folder") {
		t.Fatal("did not expect skip when dedupe is disabled")
	}
	if loads != 0 {
		t.Fatalf("expected loader to stay unused, got %d", loads)
	}
}

func TestThreshold100UsesExactComparison(t *testing.T) {
	prepareDedupeTest(t, true, 100)

	existing := solidRGBA(color.RGBA{R: 20, G: 20, B: 20, A: 255})
	loadLatestDedupeEntry = func(dir string) (*dedupeCacheEntry, error) {
		return &dedupeCacheEntry{
			path: "latest.png",
			hash: utils.AHash16x16(existing),
			img:  cloneRGBA(existing),
		}, nil
	}

	other := solidRGBA(color.RGBA{R: 20, G: 21, B: 20, A: 255})
	if ShouldSkipDueToDedupe(other, t.TempDir(), "proc.exe", "", "folder") {
		t.Fatal("expected exact dedupe to keep different pixels")
	}
}

func prepareDedupeTest(t *testing.T, enabled bool, threshold int) {
	t.Helper()
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	t.Setenv("AppData", appData)
	config.Init()
	config.SetDedupeEnabled(enabled)
	config.SetDedupeThreshold(threshold)
	lastCaptureCache.reset()
	loadLatestDedupeEntry = defaultLoadLatestDedupeEntry
}

func solidRGBA(c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}
