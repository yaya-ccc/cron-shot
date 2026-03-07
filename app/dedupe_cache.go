package app

import (
	"cron-shot/utils"
	"image"
	"image/draw"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type dedupeCacheEntry struct {
	path    string
	modTime time.Time
	hash    []byte
	img     *image.RGBA
}

type dedupeCache struct {
	mu      sync.RWMutex
	entries map[string]*dedupeCacheEntry
}

var lastCaptureCache = &dedupeCache{
	entries: map[string]*dedupeCacheEntry{},
}

var loadLatestDedupeEntry = defaultLoadLatestDedupeEntry

func defaultLoadLatestDedupeEntry(dir string) (*dedupeCacheEntry, error) {
	latest, err := utils.LoadLatestPNG(dir)
	if err != nil || latest == nil {
		return nil, err
	}
	rgba := imageToRGBA(latest.Image)
	return &dedupeCacheEntry{
		path:    latest.Path,
		modTime: latest.ModTime,
		hash:    utils.AHash16x16(rgba),
		img:     rgba,
	}, nil
}

func buildCaptureDir(storageRoot, processName, fixed, folder string) string {
	proc := utils.SanitizeProcessName(processName)
	sub := utils.SanitizeFolderName(folder)
	if strings.TrimSpace(fixed) != "" {
		fix := utils.SanitizeFolderName(fixed)
		return filepath.Join(storageRoot, proc, fix, sub)
	}
	return filepath.Join(storageRoot, proc, sub)
}

func (c *dedupeCache) get(dir string) (*dedupeCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[dir]
	if !ok {
		return nil, false
	}
	return cloneDedupeCacheEntry(entry), true
}

func (c *dedupeCache) getOrLoad(dir string) (*dedupeCacheEntry, error) {
	if entry, ok := c.get(dir); ok {
		return entry, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[dir]; ok {
		return cloneDedupeCacheEntry(entry), nil
	}

	entry, err := loadLatestDedupeEntry(dir)
	if err != nil || entry == nil {
		return nil, err
	}
	c.entries[dir] = cloneDedupeCacheEntry(entry)
	return cloneDedupeCacheEntry(entry), nil
}

func (c *dedupeCache) set(dir string, entry *dedupeCacheEntry) {
	if entry == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[dir] = cloneDedupeCacheEntry(entry)
}

func (c *dedupeCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = map[string]*dedupeCacheEntry{}
}

func RememberSavedCapture(storageRoot, processName, fixed, folder, path string, img *image.RGBA) {
	if img == nil {
		return
	}
	dir := buildCaptureDir(storageRoot, processName, fixed, folder)
	lastCaptureCache.set(dir, &dedupeCacheEntry{
		path:    path,
		modTime: time.Now(),
		hash:    utils.AHash16x16(img),
		img:     cloneRGBA(img),
	})
}

func cloneDedupeCacheEntry(entry *dedupeCacheEntry) *dedupeCacheEntry {
	if entry == nil {
		return nil
	}
	return &dedupeCacheEntry{
		path:    entry.path,
		modTime: entry.modTime,
		hash:    append([]byte(nil), entry.hash...),
		img:     cloneRGBA(entry.img),
	}
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	if src == nil {
		return nil
	}
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

func imageToRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		return cloneRGBA(rgba)
	}
	b := src.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, src, b.Min, draw.Src)
	return rgba
}
