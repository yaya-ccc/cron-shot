package app

import (
	"cron-shot/config"
	"cron-shot/utils"
	"image"
)

// ShouldSkipDueToDedupe compares the new image with the latest saved image.
func ShouldSkipDueToDedupe(img *image.RGBA, storageRoot, processName, fixed, folder string) bool {
	if !config.GetDedupeEnabled() {
		return false
	}

	dir := buildCaptureDir(storageRoot, processName, fixed, folder)
	prev, _ := lastCaptureCache.getOrLoad(dir)
	if prev == nil || prev.img == nil {
		return false
	}

	th := config.GetDedupeThreshold()
	if th >= 100 {
		return utils.ImagesEqualExact(img, prev.img)
	}

	currHash := utils.AHash16x16(img)
	dist := utils.Hamming256(prev.hash, currHash)
	sim := 1.0 - float64(dist)/256.0
	return sim*100.0 >= float64(th)
}
