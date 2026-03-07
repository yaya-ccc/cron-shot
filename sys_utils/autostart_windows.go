package sys_utils

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// EnableAutoStart 在注册表 Run 项中写入启动项以启用开机自启动
func EnableAutoStart(appName, exePath string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(appName, exePath)
}

// DisableAutoStart 从注册表 Run 项中移除启动项以禁用开机自启动
func DisableAutoStart(appName string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	// ignore error when value not exist
	_ = k.DeleteValue(appName)
	return nil
}

func IsAutoStartRegistered(appName string) (bool, string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, "", err
	}
	defer k.Close()
	v, _, err := k.GetStringValue(appName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	return strings.TrimSpace(v) != "", v, nil
}
