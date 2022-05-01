//程序操作
package windbig

import (
	"b/big"
	"golang.org/x/sys/windows/registry"
)

/**
取程序安装目录
传参：
	name：程序名称，如：chrome.exe
返回：
	成功返回安装目录，包含文件名。失败返回空文本。
*/
func ProgGetInstallDir(name string) string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, "SOFTWARE\\WOW6432Node\\Microsoft\\Windows\\CurrentVersion\\App Paths\\"+name+"\\", registry.ALL_ACCESS)
	path := ""
	if err != nil {
		goto plan2
	}
	path, _, err = key.GetStringValue("")
	if err != nil {
		goto plan2
	}
	if path != "" {
		return path
	}
plan2:
	key, err = registry.OpenKey(registry.CLASSES_ROOT, "Applications\\"+name+"\\shell\\open\\command\\", registry.ALL_ACCESS)
	if err != nil {
		return ""
	}
	path, _, err = key.GetStringValue("")
	if err != nil {
		return ""
	}
	path = big.StrGetSub(path, "\"", "\"")
	return path
}
