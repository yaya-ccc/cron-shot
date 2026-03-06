package config

import (
	"cron-shot/constants"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"cron-shot/sys_utils"
)

// AppRule 表示窗口规则配置
// Pattern: 窗口匹配文本或正则；Enabled: 是否激活；
// StorageRule: 存储文件夹解析规则（支持正则捕获组）；
// FixedFolder: 固定文件夹前缀（不为空时，截图存储于该文件夹下）
type AppRule struct {
	Pattern     string `json:"pattern"`
	Enabled     bool   `json:"enabled"`
	StorageRule string `json:"storage_rule"`
	FixedFolder string `json:"fixed_folder"`
}

// AppConfig 应用整体配置
// StorageRoot: 截图根目录；ScreenshotIntervalSec: 自动截图周期（秒）；
// DedupeEnabled: 去重开关；CurrentProcess: 当前监控进程；
// AutostartEnabled: 开机自启；AutoCaptureEnabled: 启动后自动开启截图；
// SilentStartEnabled: 静默启动到托盘；Rules: 规则列表
type AppConfig struct {
	StorageRoot                  string    `json:"storage_root"`
	ScreenshotIntervalSec        int       `json:"screenshot_interval_sec"`
	DedupeEnabled                bool      `json:"dedupe_enabled"`
	DedupeThreshold              int       `json:"dedupe_threshold"`
	AllowBackgroundWindowCapture bool      `json:"allow_background_window_capture"`
	CurrentProcess               string    `json:"current_process"`
	AutostartEnabled             bool      `json:"autostart_enabled"`
	AutoCaptureEnabled           bool      `json:"auto_capture_enabled"`
	SilentStartEnabled           bool      `json:"silent_start_enabled"`
	Rules                        []AppRule `json:"rules"`
}

var (
	mu  sync.RWMutex
	app AppConfig
)

// Init 初始化默认配置并尝试加载持久化文件
func Init() {
	app.StorageRoot = GetDefaultStorageRoot()
	app.ScreenshotIntervalSec = 5
	app.DedupeEnabled = false
	app.DedupeThreshold = 100
	app.AllowBackgroundWindowCapture = false
	_ = Load()
}

// configPath 返回配置文件路径：%APPDATA%/CronShot/config.json
func configPath() string {
	dir, _ := os.UserConfigDir()
	if dir == "" {
		dir = "."
	}
	p := filepath.Join(dir, "CronShot")
	_ = os.MkdirAll(p, 0755)
	return filepath.Join(p, "config.json")
}

// Load 读取配置文件（JSON），并填充到全局 app 变量
func Load() error {
	mu.Lock()
	defer mu.Unlock()
	p := configPath()
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	var c AppConfig
	dec := json.NewDecoder(f)
	if err := dec.Decode(&c); err != nil {
		return err
	}
	if c.StorageRoot != "" {
		app.StorageRoot = c.StorageRoot
	}
	if c.ScreenshotIntervalSec > 0 {
		app.ScreenshotIntervalSec = c.ScreenshotIntervalSec
	}
	app.DedupeEnabled = c.DedupeEnabled
	if c.DedupeThreshold > 0 {
		app.DedupeThreshold = c.DedupeThreshold
	}
	app.AllowBackgroundWindowCapture = c.AllowBackgroundWindowCapture
	app.CurrentProcess = c.CurrentProcess
	app.AutostartEnabled = c.AutostartEnabled
	app.AutoCaptureEnabled = c.AutoCaptureEnabled
	app.SilentStartEnabled = c.SilentStartEnabled
	app.Rules = c.Rules
	return nil
}

// Save 写入当前 app 配置到文件（JSON 缩进）
func Save() error {
	mu.RLock()
	c := app
	mu.RUnlock()
	f, err := os.Create(configPath())
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}

// GetStorageRoot 返回截图根目录
func GetStorageRoot() string { mu.RLock(); defer mu.RUnlock(); return app.StorageRoot }

// SetStorageRoot 设置截图根目录并持久化
func SetStorageRoot(p string) { mu.Lock(); app.StorageRoot = p; mu.Unlock(); _ = Save() }

// GetDefaultStorageRoot 返回默认截图根目录（系统图片目录/CronShot）
func GetDefaultStorageRoot() string {
	return filepath.Join(sys_utils.GetPicturesFolderWithFallback(), constants.TextAppTitle)
}

// GetScreenshotIntervalSec 返回自动截图周期（秒）
func GetScreenshotIntervalSec() int { mu.RLock(); defer mu.RUnlock(); return app.ScreenshotIntervalSec }

// SetScreenshotIntervalSec 设置自动截图周期（秒）并持久化
func SetScreenshotIntervalSec(n int) {
	mu.Lock()
	app.ScreenshotIntervalSec = n
	mu.Unlock()
	_ = Save()
}

// GetDedupeEnabled 返回是否启用去重
func GetDedupeEnabled() bool { mu.RLock(); defer mu.RUnlock(); return app.DedupeEnabled }

// SetDedupeEnabled 设置是否启用去重并持久化
func SetDedupeEnabled(v bool) { mu.Lock(); app.DedupeEnabled = v; mu.Unlock(); _ = Save() }

func GetDedupeThreshold() int  { mu.RLock(); defer mu.RUnlock(); return app.DedupeThreshold }
func SetDedupeThreshold(n int) { mu.Lock(); app.DedupeThreshold = n; mu.Unlock(); _ = Save() }

func GetAllowBackgroundWindowCapture() bool {
	mu.RLock()
	defer mu.RUnlock()
	return app.AllowBackgroundWindowCapture
}

func SetAllowBackgroundWindowCapture(v bool) {
	mu.Lock()
	app.AllowBackgroundWindowCapture = v
	mu.Unlock()
	_ = Save()
}

// GetCurrentProcess 返回当前监控进程名
func GetCurrentProcess() string { mu.RLock(); defer mu.RUnlock(); return app.CurrentProcess }

// SetCurrentProcess 设置当前监控进程名并持久化
func SetCurrentProcess(p string) { mu.Lock(); app.CurrentProcess = p; mu.Unlock(); _ = Save() }

// GetRules 返回规则切片副本
func GetRules() []AppRule {
	mu.RLock()
	defer mu.RUnlock()
	return append([]AppRule(nil), app.Rules...)
}

// SetRules 设置规则列表并持久化
func SetRules(r []AppRule) {
	mu.Lock()
	app.Rules = append([]AppRule(nil), r...)
	mu.Unlock()
	_ = Save()
}

// GetAutostartEnabled 返回是否开机自启
func GetAutostartEnabled() bool { mu.RLock(); defer mu.RUnlock(); return app.AutostartEnabled }

// SetAutostartEnabled 设置开机自启并持久化
func SetAutostartEnabled(v bool) { mu.Lock(); app.AutostartEnabled = v; mu.Unlock(); _ = Save() }

// GetAutoCaptureEnabled 返回是否自动开启截图
func GetAutoCaptureEnabled() bool { mu.RLock(); defer mu.RUnlock(); return app.AutoCaptureEnabled }

// SetAutoCaptureEnabled 设置是否自动开启截图并持久化
func SetAutoCaptureEnabled(v bool) { mu.Lock(); app.AutoCaptureEnabled = v; mu.Unlock(); _ = Save() }

// GetSilentStartEnabled 返回是否启用静默启动
func GetSilentStartEnabled() bool { mu.RLock(); defer mu.RUnlock(); return app.SilentStartEnabled }

// SetSilentStartEnabled 设置是否启用静默启动并持久化
func SetSilentStartEnabled(v bool) { mu.Lock(); app.SilentStartEnabled = v; mu.Unlock(); _ = Save() }
