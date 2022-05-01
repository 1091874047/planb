//系统操作
package big

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

/**
通过端口号找进程PID
传参：
	port：端口号
返回：
	进程PID，没找到返回-1
*/
func SysPortToPid(port int) int {
	res := -1
	var outPut []byte
	//判断系统类型
	if runtime.GOOS == "windows" {
		outPut, _ = exec.Command("cmd", "/c", fmt.Sprintf("netstat -ano -p tcp | findstr %d", port)).CombinedOutput()
	} else {
		outPut, _ = exec.Command("sh", "-c", fmt.Sprintf("lsof -i:%d ", port)).CombinedOutput()
	}
	resStr := string(outPut)
	r := regexp.MustCompile(`\s\d+\s`).FindAllString(resStr, -1)
	if len(r) > 0 {
		pid, err := strconv.Atoi(strings.TrimSpace(r[0]))
		if err != nil {
			res = -1
		} else {
			res = pid
		}
	}
	return res
}

/**
端口是否被占用
传参：
	port：端口号
返回：
	占用=true，没占用=false
*/
func SysPortInUse(port int) bool {
	if SysPortToPid(port) == -1 {
		return false
	}
	return true
}

//取系统类型，返回系统架构和系统平台
func SysGetType() (arch string, os string) {
	return runtime.GOARCH, runtime.GOOS
}

//取系统CPU信息，返回核心数和架构
func SysGetCpuInfo() (num int, arch string) {
	return runtime.NumCPU(), runtime.GOARCH
}

/**
取键码
传参：
	key：欲取的键码
返回：
	键代码
*/
func SysKeyCode(key string) int {
	if key == "" {
		return 0
	}
	d := []rune(key)
	return int(d[0])
}

/**
取外网IP
*/
func SysGetWanIp() WanIp {
	res := WanIp{}
	temp, _, _, _ := HttpSend(&HttpParms{Url: "http://pv.sohu.com/cityjson?ie=utf-8"})
	json.Unmarshal([]byte(StrGetSub(temp, "= ", ";")), &res)
	return res
}

type WanIp struct {
	Ip        string `json:"cip"`   //ip地址
	AdminCode string `json:"cid"`   //邮政编码
	Region    string `json:"cname"` //城市信息
}
