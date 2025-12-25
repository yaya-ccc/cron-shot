package sys_utils

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"cron-shot/utils"

	"github.com/lxn/win"
)

func getWindowDC(hwnd win.HWND) win.HDC {
	ret, _, _ := procGetWindowDC.Call(uintptr(hwnd))
	return win.HDC(ret)
}

func CaptureWindowImage(hwnd win.HWND) (*image.RGBA, error) {
	return captureInternal(hwnd, true)
}

func CaptureWindowImageBitBlt(hwnd win.HWND) (*image.RGBA, error) {
	return captureInternal(hwnd, false)
}

func captureInternal(hwnd win.HWND, usePrintWindow bool) (*image.RGBA, error) {
	var rect win.RECT
	win.GetWindowRect(hwnd, &rect)
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	var srcDC win.HDC
	if usePrintWindow {
		srcDC = win.GetDC(0)
		defer win.ReleaseDC(0, srcDC)
	} else {
		srcDC = getWindowDC(hwnd)
		if srcDC == 0 {
			return nil, errors.New("GetWindowDC failed")
		}
		defer win.ReleaseDC(hwnd, srcDC)
	}
	hdcMem := win.CreateCompatibleDC(srcDC)
	defer win.DeleteDC(hdcMem)
	hbm := win.CreateCompatibleBitmap(srcDC, int32(width), int32(height))
	defer win.DeleteObject(win.HGDIOBJ(hbm))
	win.SelectObject(hdcMem, win.HGDIOBJ(hbm))
	if usePrintWindow {
		const PW_RENDERFULLCONTENT = 0x00000002
		r, _, _ := procPrintWindow.Call(uintptr(hwnd), uintptr(hdcMem), uintptr(PW_RENDERFULLCONTENT))
		if r == 0 {
			return nil, errors.New("PrintWindow failed")
		}
	} else {
		const SRCCOPY = 0x00CC0020
		if !win.BitBlt(hdcMem, 0, 0, int32(width), int32(height), srcDC, 0, 0, uint32(SRCCOPY)) {
			return nil, errors.New("BitBlt failed")
		}
	}
	var bmi win.BITMAPINFO
	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bmi.BmiHeader))
	bmi.BmiHeader.BiWidth = int32(width)
	bmi.BmiHeader.BiHeight = -int32(height)
	bmi.BmiHeader.BiPlanes = 1
	bmi.BmiHeader.BiBitCount = 32
	bmi.BmiHeader.BiCompression = win.BI_RGB
	stride := width * 4
	buf := make([]byte, stride*height)
	win.GetDIBits(hdcMem, hbm, 0, uint32(height), &buf[0], &bmi, win.DIB_RGB_COLORS)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pi := 0
	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			i := row + x*4
			b := buf[i+0]
			g := buf[i+1]
			r := buf[i+2]
			a := buf[i+3]
			if a == 0 {
				a = 255
			}
			img.Pix[pi+0] = r
			img.Pix[pi+1] = g
			img.Pix[pi+2] = b
			img.Pix[pi+3] = a
			pi += 4
		}
	}
	return img, nil
}

// SaveCronShot 保存截图到 根目录\\进程名\\(固定文件夹)\\规则文件夹 下
func SaveCronShot(img *image.RGBA, root string, processName, fixedFolder, folderName string, t time.Time) (string, error) {
	proc := utils.SanitizeProcessName(processName)
	sub := utils.SanitizeFolderName(folderName)
	if fixedFolder != "" {
		fix := utils.SanitizeFolderName(fixedFolder)
		dir := filepath.Join(root, proc, fix, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
		name := t.Format("20060102_150405.000") + ".png"
		path := filepath.Join(dir, name)
		f, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			return "", err
		}
		return path, nil
	}
	dir := filepath.Join(root, proc, sub)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := t.Format("20060102_150405.000") + ".png"
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return "", err
	}
	return path, nil
}
