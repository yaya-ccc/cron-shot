package gui

import (
	"fmt"
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	appctrl "cron-shot/app"
	"cron-shot/config"
	"cron-shot/constants"
	"cron-shot/logging"
	platformwin "cron-shot/platform/win"
	"cron-shot/sys_utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
)

// Run 启动应用程序主界面与业务逻辑
func Run() {
	defer logging.RecoverPanic("gui.Run")
	myApp := app.New()
	myApp.Settings().SetTheme(&customTheme{})
	myApp.SetIcon(platformwin.GetTrayIconResource())
	myWindow := myApp.NewWindow(constants.TextAppTitle)
	AppCanvas = myWindow.Canvas()
	config.Init()
	cfgDir, _ := os.UserConfigDir()
	baseCfg := filepath.Join(cfgDir, "CronShot")
	_ = logging.Init(baseCfg)
	logging.Info("config loaded")

	platformwin.InitAutostartRegistration()

	// 初始化各个模块
	rulesUI := NewRulesUI(myApp)
	windowStatusUI := NewWindowStatusUI()
	windowStatusUI.RulesProvider = func() []WindowRule { return rulesUI.Rules }
	processController := appctrl.NewProcessWindowController()
	processController.OnWindowsUpdated = func(w []string) { windowStatusUI.UpdateWindows(w) }
	rulesUI.OnRulesChanged = func() {
		windowStatusUI.UpdateWindows(windowStatusUI.Windows)
		// 持久化规则
		var cfgRules []config.AppRule
		for _, r := range rulesUI.Rules {
			cfgRules = append(cfgRules, config.AppRule{Pattern: r.Pattern, Enabled: r.Enabled, StorageRule: r.StorageRule, FixedFolder: r.FixedFolder})
		}
		config.SetRules(cfgRules)
	}

	var currentProcess string
	processUI := NewProcessUI(myApp, func(selectedProcess string) {
		// 进程选择回调
		// 切换监控的进程，这会立即更新一次窗口列表，并启动定时轮询
		processController.SetProcess(selectedProcess)
		currentProcess = selectedProcess
		config.SetCurrentProcess(selectedProcess)
	})

	// 从配置加载规则与进程
	if cfg := config.GetRules(); len(cfg) > 0 {
		rulesUI.Rules = nil
		for _, r := range cfg {
			rulesUI.Rules = append(rulesUI.Rules, WindowRule{Pattern: r.Pattern, Enabled: r.Enabled, StorageRule: r.StorageRule, FixedFolder: r.FixedFolder})
		}
		rulesUI.RuleList.Refresh()
	}
	if p := config.GetCurrentProcess(); strings.TrimSpace(p) != "" {
		currentProcess = p
		processController.SetProcess(p)
		processUI.EntryProcessLocked.SetText(p)
	}

	// 将规则列表和状态列表组合在中间区域
	// 使用 VBox 垂直排列
	centerContent := container.NewVBox(
		rulesUI.Container,
		windowStatusUI.Container,
	)

	autoEnabled := false
	autoCtrl := appctrl.NewAutoCaptureController(
		func() string { return currentProcess },
		func() []config.AppRule { return config.GetRules() },
	)
	autoBtn := widget.NewButton(constants.TextOpenAutoShot, nil)
	autoBtn.Importance = widget.MediumImportance
	autoBtn.OnTapped = func() {
		autoEnabled = !autoEnabled
		if autoEnabled {
			autoBtn.SetText(constants.TextCloseAutoShot)
			autoBtn.Importance = widget.HighImportance
			autoBtn.Refresh()
			autoCtrl.Start()
		} else {
			autoBtn.SetText(constants.TextOpenAutoShot)
			autoBtn.Importance = widget.MediumImportance
			autoBtn.Refresh()
			autoCtrl.Stop()
		}
	}
	settingsBtn := widget.NewButton(constants.TextSettings, func() {
		onSettingsButtonTapped(myApp, &autoEnabled, nil, autoBtn, &[]bool{config.GetDedupeEnabled()}[0], &currentProcess, rulesUI, windowStatusUI)
		if autoEnabled {
			autoCtrl.Stop()
			autoCtrl.Start()
		}
	})
	openPicturesBtn := widget.NewButton(constants.TextOpenPicturesFolder, func() {
		_ = sys_utils.OpenFolder(config.GetStorageRoot())
	})
	openConfigBtn := widget.NewButton(constants.TextOpenConfigFolder, func() {
		cfgDir, _ := os.UserConfigDir()
		base := filepath.Join(cfgDir, constants.TextAppTitle)
		_ = sys_utils.OpenFolder(base)
	})
	buttonsRow := container.NewGridWithColumns(1, autoBtn)
	bottomRow := container.NewBorder(nil, nil, nil, nil, buttonsRow)

	aboutBtn := widget.NewButton(constants.TextAbout, func() {
		w := NewSingletonWindow(constants.TextAbout)
		l1 := widget.NewLabel("版本:v0.0.3")
		l2 := widget.NewLabel("作者:YaYa")
		u, _ := url.Parse("https://github.com/YaYaccc/cron-shot")
		rowProject := widget.NewRichText(&widget.TextSegment{Text: "项目地址:"}, &widget.HyperlinkSegment{Text: "https://github.com/YaYaccc/cron-shot", URL: u})
		u2, _ := url.Parse("https://icons8.com/")
		rowIcons := widget.NewRichText(&widget.TextSegment{Text: "图标来源:"}, &widget.HyperlinkSegment{Text: "Icons8", URL: u2})
		form := container.NewVBox(l1, l2, rowProject, rowIcons)
		wrapped := fynetooltip.AddWindowToolTipLayer(container.NewPadded(form), w.Canvas())
		w.SetContent(wrapped)
		w.Resize(fyne.NewSize(420, 200))
		w.SetOnClosed(func() { fynetooltip.DestroyWindowToolTipLayer(w.Canvas()) })
		w.Show()
	})
	actionsTop := container.NewGridWithColumns(2, openPicturesBtn, openConfigBtn)
	actionsBottom := container.NewGridWithColumns(2, settingsBtn, aboutBtn)
	actionsRow := container.NewVBox(actionsTop, actionsBottom)
	centerContent = container.NewVBox(
		rulesUI.Container,
		windowStatusUI.Container,
		bottomRow,
		actionsRow,
	)

	content := container.NewBorder(
		processUI.Container,
		nil,
		nil, nil,
		centerContent,
	)

	wrapped := fynetooltip.AddWindowToolTipLayer(container.NewPadded(content), myWindow.Canvas())
	myWindow.SetContent(wrapped)
	myWindow.Resize(fyne.NewSize(600, 600))
	platformwin.SetupSystemTray(myApp, myWindow, processController)
	myWindow.SetCloseIntercept(myWindow.Hide)
	if config.GetSilentStartEnabled() {
		platformwin.StartHideOnMinimize(myWindow)
	}

	// 确保在窗口关闭时停止轮询
	myWindow.SetOnClosed(func() {
		processController.Stop()
		fynetooltip.DestroyWindowToolTipLayer(myWindow.Canvas())
	})

	if config.GetAutoCaptureEnabled() && strings.TrimSpace(currentProcess) != "" && !autoEnabled {
		autoBtn.OnTapped()
		autoBtn.Refresh()
	}
	if config.GetSilentStartEnabled() {
		myWindow.Hide()
		myApp.Run()
	} else {
		myWindow.ShowAndRun()
	}
}

// onSettingsButtonTapped 打开设置窗口并保存改动
func onSettingsButtonTapped(_ fyne.App, autoEnabled *bool, autoStopChanPtr *chan struct{}, _ *widget.Button, dedupeEnabled *bool, currentProcess *string, rulesUI *RulesUI, windowStatusUI *WindowStatusUI) {
	w := NewSingletonWindow(constants.TextSettingsTitle)
	entryRoot := widget.NewEntry()
	entryRoot.SetText(config.GetStorageRoot())
	entryRootWrap := container.NewGridWrap(fyne.NewSize(420, entryRoot.MinSize().Height), entryRoot)
	entryInterval := widget.NewEntry()
	entryInterval.SetText(fmt.Sprintf("%d", config.GetScreenshotIntervalSec()))
	toggleDedupe := widget.NewCheck(constants.TextDedupeTitle, func(v bool) {})
	toggleDedupe.SetChecked(config.GetDedupeEnabled())
	valueLabel := widget.NewLabel(fmt.Sprintf("%d", config.GetDedupeThreshold()))
	sliderThreshold := widget.NewSlider(1, 100)
	sliderThreshold.Step = 1
	sliderThreshold.Value = float64(config.GetDedupeThreshold())
	sliderThreshold.OnChanged = func(v float64) {
		valueLabel.SetText(fmt.Sprintf("%d", int(v)))
	}
	leftInfo := container.NewHBox(widget.NewLabel(constants.TextDedupeThreshold), valueLabel)
	thresholdRow := container.NewBorder(nil, nil, leftInfo, nil, sliderThreshold)
	thresholdRow.Hide()
	if toggleDedupe.Checked {
		thresholdRow.Show()
	}
	toggleDedupe.OnChanged = func(v bool) {
		if v {
			thresholdRow.Show()
		} else {
			thresholdRow.Hide()
		}
	}
	toggleAllowBackgroundWindowCapture := widget.NewCheck(constants.TextAllowBackgroundWindowCaptureTitle, nil)
	toggleAllowBackgroundWindowCapture.SetChecked(config.GetAllowBackgroundWindowCapture())
	bgNoteText := canvas.NewText(constants.TextAllowBackgroundWindowCaptureNote, color.NRGBA{R: 121, G: 70, B: 5, A: 255})
	bgNoteText.TextSize = theme.Size(theme.SizeNameCaptionText)
	bgNoteText.TextStyle = fyne.TextStyle{Bold: true}
	bgNoteBg := canvas.NewRectangle(color.NRGBA{R: 255, G: 244, B: 230, A: 245})
	bgNoteBg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	bgNoteBg.StrokeColor = color.NRGBA{R: 232, G: 167, B: 73, A: 255}
	bgNoteBg.StrokeWidth = 1
	bgNote := container.NewStack(bgNoteBg, container.NewPadded(bgNoteText))
	bgNote.Hide()
	if toggleAllowBackgroundWindowCapture.Checked {
		bgNote.Show()
	}
	toggleAllowBackgroundWindowCapture.OnChanged = func(v bool) {
		if v {
			bgNote.Show()
			return
		}
		bgNote.Hide()
	}
	toggleAutoStart := widget.NewCheck(constants.TextAutoStartTitle, func(v bool) {})
	toggleAutoStart.SetChecked(config.GetAutostartEnabled())
	toggleAutoCapture := widget.NewCheck(constants.TextAutoCaptureTitle, func(v bool) {})
	toggleAutoCapture.SetChecked(config.GetAutoCaptureEnabled())
	toggleSilentStart := widget.NewCheck(constants.TextSilentStartTitle, func(v bool) {})
	toggleSilentStart.SetChecked(config.GetSilentStartEnabled())
	chooseBtn := widget.NewButton(constants.TextChoose, func() {
		if p, err := sys_utils.PickFolder(); err == nil && strings.TrimSpace(p) != "" {
			entryRoot.SetText(p)
		}
	})
	resetBtn := widget.NewButton(constants.TextResetDefault, func() {
		entryRoot.SetText(config.GetDefaultStorageRoot())
	})
	save := widget.NewButton(constants.TextSave, func() {
		root := entryRoot.Text
		config.SetStorageRoot(root)
		n := 5
		if v, err := strconv.Atoi(strings.TrimSpace(entryInterval.Text)); err == nil {
			if v > 0 {
				n = v
			}
		}
		config.SetScreenshotIntervalSec(n)
		config.SetDedupeEnabled(toggleDedupe.Checked)
		*dedupeEnabled = toggleDedupe.Checked
		// threshold
		th := int(sliderThreshold.Value)
		if th < 1 {
			th = 1
		} else if th > 100 {
			th = 100
		}
		config.SetDedupeThreshold(th)
		config.SetAllowBackgroundWindowCapture(toggleAllowBackgroundWindowCapture.Checked)
		config.SetAutostartEnabled(toggleAutoStart.Checked)
		config.SetAutoCaptureEnabled(toggleAutoCapture.Checked)
		config.SetSilentStartEnabled(toggleSilentStart.Checked)
		exe, _ := os.Executable()
		if toggleAutoStart.Checked {
			_ = sys_utils.EnableAutoStart(constants.TextAppTitle, exe)
		} else {
			_ = sys_utils.DisableAutoStart(constants.TextAppTitle)
		}
		// 自动截图的重启在 settingsBtn 点击处理处统一完成
		w.Close()
	})
	cancel := widget.NewButton(constants.TextCancel, func() { w.Close() })
	form := container.NewVBox(
		widget.NewLabel(constants.TextStorageRootTitle),
		container.NewHBox(entryRootWrap, chooseBtn, resetBtn),
		widget.NewLabel(constants.TextIntervalTitle),
		entryInterval,
		toggleDedupe,
		thresholdRow,
		toggleAllowBackgroundWindowCapture,
		bgNote,
		toggleAutoStart,
		toggleAutoCapture,
		toggleSilentStart,
		container.NewHBox(save, cancel),
	)
	wrapped := fynetooltip.AddWindowToolTipLayer(container.NewPadded(form), w.Canvas())
	w.SetContent(wrapped)
	w.Resize(fyne.NewSize(520, 360))
	w.SetOnClosed(func() { fynetooltip.DestroyWindowToolTipLayer(w.Canvas()) })
	w.Show()
}
