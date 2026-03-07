package gui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
)

type BubbleMessageStyle int

const (
	BubbleMessageInfo BubbleMessageStyle = iota
	BubbleMessageWarning
)

func ShowBubbleMessage(canvasTarget fyne.Canvas, owner fyne.CanvasObject, rel fyne.Position, message string, style BubbleMessageStyle) {
	if canvasTarget == nil || owner == nil || message == "" {
		return
	}

	bg := color.NRGBA{R: 248, G: 249, B: 251, A: 236}
	border := color.NRGBA{R: 207, G: 214, B: 223, A: 255}
	fg := color.NRGBA{R: 60, G: 66, B: 74, A: 0}
	if style == BubbleMessageWarning {
		bg = color.NRGBA{R: 255, G: 244, B: 230, A: 245}
		border = color.NRGBA{R: 232, G: 167, B: 73, A: 255}
		fg = color.NRGBA{R: 121, G: 70, B: 5, A: 0}
	}

	text := canvas.NewText(message, fg)
	text.Alignment = fyne.TextAlignCenter
	text.TextSize = theme.Size(theme.SizeNameCaptionText)
	text.TextStyle = fyne.TextStyle{Bold: style == BubbleMessageWarning}

	bgRect := canvas.NewRectangle(bg)
	bgRect.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	bgRect.StrokeColor = border
	bgRect.StrokeWidth = 1

	content := container.NewPadded(text)
	bubble := container.NewStack(bgRect, content)
	pop := widget.NewPopUp(bubble, canvasTarget)
	fynetooltip.AddPopUpToolTipLayer(pop)
	pop.Resize(bubble.MinSize())
	pop.ShowAtRelativePosition(rel, owner)

	steps := 8
	stepDur := 200 * time.Millisecond / time.Duration(steps)
	go func() {
		for i := 1; i <= steps; i++ {
			a := uint8(i * 255 / steps)
			time.Sleep(stepDur)
			fyne.Do(func() {
				text.Color = color.NRGBA{R: fg.R, G: fg.G, B: fg.B, A: a}
				text.Refresh()
			})
		}

		time.Sleep(1100 * time.Millisecond)

		for i := steps - 1; i >= 0; i-- {
			a := uint8(i * 255 / steps)
			time.Sleep(stepDur)
			fyne.Do(func() {
				text.Color = color.NRGBA{R: fg.R, G: fg.G, B: fg.B, A: a}
				text.Refresh()
			})
		}

		fyne.Do(func() {
			pop.Hide()
			fynetooltip.DestroyPopUpToolTipLayer(pop)
		})
	}()
}
