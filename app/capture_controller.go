package app

import (
	"cron-shot/config"
	"cron-shot/logging"
	"cron-shot/sys_utils"
	"image"
	"time"

	"github.com/lxn/win"
)

// AutoCaptureController drives periodic capture for the selected process.
type AutoCaptureController struct {
	stopChan       chan struct{}
	CurrentProcess func() string
	GetRules       func() []config.AppRule
}

// NewAutoCaptureController creates a capture controller.
func NewAutoCaptureController(curr func() string, rules func() []config.AppRule) *AutoCaptureController {
	return &AutoCaptureController{CurrentProcess: curr, GetRules: rules}
}

// Start launches the background capture loop.
func (c *AutoCaptureController) Start() {
	if c.stopChan != nil {
		return
	}
	c.stopChan = make(chan struct{})
	go c.loop(c.stopChan)
}

// Stop terminates the background capture loop.
func (c *AutoCaptureController) Stop() {
	if c.stopChan != nil {
		close(c.stopChan)
		c.stopChan = nil
	}
}

func (c *AutoCaptureController) loop(stop chan struct{}) {
	defer logging.RecoverPanic("AutoCaptureController.loop")

	interval := time.Duration(config.GetScreenshotIntervalSec()) * time.Second
	if interval <= 0 {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c.runOnce()
		}
	}
}

func (c *AutoCaptureController) runOnce() {
	proc := ""
	if c.CurrentProcess != nil {
		proc = c.CurrentProcess()
	}
	if proc == "" {
		return
	}

	logging.Info("start screenshot tick for process: " + proc)
	infos, err := sys_utils.GetProcessWindowsDetailed(proc)
	if err != nil || len(infos) == 0 {
		return
	}

	rules := []config.AppRule{}
	if c.GetRules != nil {
		rules = c.GetRules()
	}

	base := time.Now()
	idx := 0
	for _, info := range infos {
		rule, ok := MatchRule(info.Title, rules)
		if !ok {
			continue
		}
		if img, path := c.captureAndSave(proc, info, rule, base.Add(time.Duration(idx)*time.Millisecond)); img != nil {
			_ = path
		}
		idx++
	}
}

func (c *AutoCaptureController) captureAndSave(proc string, info sys_utils.WindowInfo, rule *config.AppRule, t time.Time) (*image.RGBA, string) {
	allowBackgroundCapture := config.GetAllowBackgroundWindowCapture()
	isForeground := win.GetForegroundWindow() == info.HWND
	isVisible := win.IsWindowVisible(info.HWND)
	isMinimized := win.IsIconic(info.HWND)

	if !allowBackgroundCapture && (isMinimized || !isVisible) {
		return nil, ""
	}

	var img *image.RGBA
	var err error
	if allowBackgroundCapture && (!isForeground || isMinimized || !isVisible) {
		logging.Info("capture mode: PrintWindow, title=" + info.Title)
		img, err = sys_utils.CaptureWindowImage(info.HWND)
		if err != nil && isVisible && !isMinimized {
			logging.Info("capture fallback: PrintWindow -> BitBlt for inactive window, title=" + info.Title)
			img, err = sys_utils.CaptureWindowImageBitBlt(info.HWND)
		}
	} else {
		logging.Info("capture mode: BitBlt, title=" + info.Title)
		img, err = sys_utils.CaptureWindowImageBitBlt(info.HWND)
		if err != nil && allowBackgroundCapture {
			logging.Info("capture fallback: BitBlt -> PrintWindow for active window, title=" + info.Title)
			img, err = sys_utils.CaptureWindowImage(info.HWND)
		}
	}
	if err != nil {
		logging.Error("capture failed: " + err.Error())
		return nil, ""
	}

	folder, fixed := ResolveFolder(info.Title, rule)
	if ShouldSkipDueToDedupe(img, config.GetStorageRoot(), proc, fixed, folder) {
		logging.Info("skip save due to dedupe")
		return img, ""
	}

	p, err := sys_utils.SaveCronShot(img, config.GetStorageRoot(), proc, fixed, folder, t)
	if err != nil {
		logging.Error("save failed: " + err.Error())
		return img, ""
	}

	logging.Info("screenshot saved: " + p)
	return img, p
}
