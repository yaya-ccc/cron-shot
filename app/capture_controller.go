package app

import (
	"cron-shot/config"
	"cron-shot/logging"
	"cron-shot/sys_utils"
	"image"
	"time"

	"github.com/lxn/win"
)

// AutoCaptureController 负责根据配置周期性截取当前进程的窗口并保存
// 通过回调获取当前进程名与规则集合，内部使用定时器驱动循环
type AutoCaptureController struct {
	stopChan       chan struct{}
	CurrentProcess func() string
	GetRules       func() []config.AppRule
}

// NewAutoCaptureController 创建控制器
// curr: 返回当前选择的进程名；rules: 返回最新规则列表
func NewAutoCaptureController(curr func() string, rules func() []config.AppRule) *AutoCaptureController {
	return &AutoCaptureController{CurrentProcess: curr, GetRules: rules}
}

// Start 启动自动截图循环
func (c *AutoCaptureController) Start() {
	if c.stopChan != nil {
		return
	}
	c.stopChan = make(chan struct{})
	// 启动后台 goroutine 执行周期任务
	go c.loop(c.stopChan)
}

// Stop 停止自动截图循环
func (c *AutoCaptureController) Stop() {
	if c.stopChan != nil {
		close(c.stopChan)
		c.stopChan = nil
	}
}

// loop 定时驱动自动截图；收到 stop 信号后退出
func (c *AutoCaptureController) loop(stop chan struct{}) {
	defer logging.RecoverPanic("AutoCaptureController.loop")
	// 读取截图间隔（秒）；非法值时回退到 1s
	interval := time.Duration(config.GetScreenshotIntervalSec()) * time.Second
	if interval <= 0 {
		interval = time.Second
	}
	// 使用 Ticker 周期触发截图
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

// runOnce 执行一次完整的截图与保存流程
func (c *AutoCaptureController) runOnce() {
	// 从回调获取当前进程名；为空则跳过
	proc := ""
	if c.CurrentProcess != nil {
		proc = c.CurrentProcess()
	}
	if proc == "" {
		return
	}
	logging.Info("start screenshot tick for process: " + proc)
	// 枚举该进程的可见窗口（标题+句柄）
	infos, err := sys_utils.GetProcessWindowsDetailed(proc)
	if err != nil || len(infos) == 0 {
		return
	}
	// 读取规则列表用于匹配
	rules := []config.AppRule{}
	if c.GetRules != nil {
		rules = c.GetRules()
	}
	base := time.Now()
	idx := 0
	for _, info := range infos {
		title := info.Title
		// 先用窗口标题执行规则匹配（优先文本等价，其次正则）
		rule, ok := MatchRule(title, rules)
		if !ok {
			continue
		}
		// 执行截图与保存；索引用于微调多窗口的时间戳
		if img, path := c.captureAndSave(proc, info, rule, base.Add(time.Duration(idx)*time.Millisecond)); img != nil {
			_ = path
		}
		idx++
	}
}

// captureAndSave 对单个窗口执行截图、去重判断与保存
func (c *AutoCaptureController) captureAndSave(proc string, info sys_utils.WindowInfo, rule *config.AppRule, t time.Time) (*image.RGBA, string) {
	// 跳过最小化或不可见窗口，避免空白截图
	if win.IsIconic(info.HWND) || !win.IsWindowVisible(info.HWND) {
		return nil, ""
	}
	var img *image.RGBA
	var err error
	if win.GetForegroundWindow() == info.HWND {
		img, err = sys_utils.CaptureWindowImageBitBlt(info.HWND)
	} else {
		img, err = sys_utils.CaptureWindowImage(info.HWND)
	}
	if err != nil {
		logging.Error("capture failed: " + err.Error())
		return nil, ""
	}
	// 解析存储文件夹并进行去重判断
	folder, fixed := ResolveFolder(info.Title, rule)
	if ShouldSkipDueToDedupe(img, config.GetStorageRoot(), proc, fixed, folder) {
		logging.Info("skip save due to dedupe")
		return img, ""
	}
	// 保存截图到目标目录
	p, err := sys_utils.SaveCronShot(img, config.GetStorageRoot(), proc, fixed, folder, t)
	if err != nil {
		logging.Error("save failed: " + err.Error())
		return img, ""
	}
	logging.Info("screenshot saved: " + p)
	return img, p
}
