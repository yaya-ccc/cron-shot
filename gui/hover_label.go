package gui

import (
	"image/color"

	"cron-shot/constants"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// HoverLabel 是一个支持鼠标悬停显示 Tooltip 的 Label
type HoverLabel struct {
	ttwidget.ToolTipWidget
	label       *widget.Label
	bg          *canvas.Rectangle
	highlighted bool
}

// NewHoverLabel 创建一个新的 HoverLabel
func NewHoverLabel(text string) *HoverLabel {
	l := &HoverLabel{}
	l.label = widget.NewLabel(text)
	l.bg = canvas.NewRectangle(theme.BackgroundColor())
	l.ExtendBaseWidget(l)
	l.SetToolTip(text)
	return l
}

func (l *HoverLabel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(l.bg, l.label))
}

func (l *HoverLabel) SetText(text string) {
	l.label.SetText(text)
	l.SetToolTip(text)
}

func (l *HoverLabel) SetHighlighted(h bool) {
	l.highlighted = h
	if h {
		l.bg.FillColor = color.NRGBA{R: 255, G: 236, B: 236, A: 255}
		l.bg.StrokeColor = color.NRGBA{R: 255, G: 200, B: 200, A: 255}
		l.bg.StrokeWidth = 1
	} else {
		l.bg.FillColor = theme.BackgroundColor()
		l.bg.StrokeWidth = 0
	}
	l.bg.Refresh()
	l.Refresh()
}

func (l *HoverLabel) TappedSecondary(ev *fyne.PointEvent) {
	fyne.CurrentApp().Clipboard().SetContent(l.label.Text)
	l.showCopiedBubble(ev.Position)
}

func (l *HoverLabel) showCopiedBubble(rel fyne.Position) {
	ShowBubbleMessage(AppCanvas, l, rel, constants.TextCopiedBubble, BubbleMessageInfo)
}

func (l *HoverLabel) MinSize() fyne.Size {
	s := l.label.MinSize()
	if s.Width > 10 {
		s.Width = 10
	}
	return s
}
