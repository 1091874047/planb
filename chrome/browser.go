//本库仅支持Windows平台的所有使用Chrome内核的浏览器,线程安全的
//Browser结构体用于操作浏览器
package chrome

import (
	"b/big"
	"b/windbig"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type Browser struct {
	Ip          string   `json:"ip"`                       //调试IP地址，如果是本地填localhost或127.0.0.1，如果是外网则填外网IP，注意外网时不支持OpenBrowser方法打开浏览器
	Port        int      `json:"remote-debugging-port"`    //调试端口，可空，默认为随机生成
	Path        string   `json:"path"`                     //浏览器启动文件路径，可空，默认为Chrome浏览器
	DefaultUrl  string   `json:"new-window"`               //启动浏览器时打开的网址，可空，默认为首页
	Width       int      `json:"width"`                    //浏览器窗口宽度，单位像素，可空，默认为1000
	Height      int      `json:"height"`                   //浏览器窗口高度，单位像素，可空，默认为700
	Maximized   bool     `json:"start-maximized"`          //浏览器启动时直接最大化，为true时为最大化启动，设置的Width、Height参数将失效
	Hide        bool     `json:"hide"`                     //隐身无痕模式，true为开启，开启后DataDir设置失效
	DataDir     string   `json:"user-data-dir""`           //浏览器数据文件存放路径，可空，默认为原始路径，建议配置此项，防止数据冲突
	Proxy       string   `json:"proxy-server"`             //代理IP，可空，默认为不使用代理，格式：127.0.0.1:8888或http://user:password@proxy.com:8080
	Headless    bool     `json:"headless"`                 //是否启动无头浏览器，启用为true，无界面隐藏运行
	RemoteDebug bool     `json:"remote-debugging-address"` //是否允许外网调试，允许为true并且Hide强制为true
	DisSecurity bool     `json:"allow-insecure-localhost"` //禁用localhost上的TLS/SSL错误（无插页式，不阻止请求）,true为禁用
	Args        []string `json:"args"`                     //其他附加参数--开头
}

/**
打开一个浏览器，如果指定了端口，恰巧该端口的浏览器已经在运行中，那么本命令等价于打开一个新标签
本方法会将browser属性字段作为参数启动浏览器
所以如需自定义浏览器环境，需在调用本方法前给browser对象属性字段赋值
注：本方法不支持外网IP调试浏览器时的调用
返回：
	成功error返回nil，失败error返回具体信息
*/
func (p *Browser) OpenBrowser() error {
	if p.Ip != "localhost" && p.Ip != "127.0.0.1" && p.Ip != "" {
		return errors.New("本方法不支持外网IP浏览器")
	}
	if p.Ip == "" {
		p.Ip = "localhost"
	}
	//取浏览器路径
	if p.Path == "" {
		p.Path = windbig.ProgGetInstallDir("chrome.exe")
	}
	if p.Path == "" {
		return errors.New("未安装chrome浏览器")
	}
	port := strconv.Itoa(p.Port)
	//取浏览器端口
	if p.Port == 0 {
		for {
			p.Port = big.ProgRangeRand(1024, 65535, 0)
			if !big.SysPortInUse(p.Port) {
				break
			}
		}
	} else {
		if big.SysPortInUse(p.Port) {
			big.HttpSend(&big.HttpParms{Url: "http://" + p.Ip + ":" + port + "/json/new?"})
			return errors.New("指定的端口已有浏览器在运行，所以本次新建了一个标签页")
		}
	}
	port = strconv.Itoa(p.Port)
	args := make([]string, 0)
	//args = append(args, "cmd")
	args = append(args, "/c")
	args = append(args, p.Path)
	args = append(args, "--remote-debugging-port="+port)
	if p.Hide {
		dir, _ := os.Getwd()
		p.DataDir = dir + "\\chrome\\" + port
	}
	if p.DataDir != "" {
		args = append(args, "--user-data-dir="+p.DataDir)
	}
	if p.Proxy != "" {
		args = append(args, "--proxy-server="+p.Proxy)
	}
	if p.DisSecurity {
		args = append(args, "--allow-insecure-localhost")
	}
	if p.RemoteDebug {
		p.Headless = true
		args = append(args, "--remote-debugging-address=0.0.0.0")
	}
	if p.Headless {
		args = append(args, "--headless")
	}
	if p.Maximized {
		args = append(args, "--start-maximized")
	} else {
		if p.Width == 0 {
			p.Width = 1000
		}
		if p.Height == 0 {
			p.Height = 700
		}
		args = append(args, "--window-size="+strconv.Itoa(p.Width)+","+strconv.Itoa(p.Height))
	}
	//追加附加参数
	for _, v := range p.Args {
		args = append(args, v)
	}
	if p.DefaultUrl != "" {
		args = append(args, p.DefaultUrl)
	}
	cmd := &exec.Cmd{
		Path: "cmd",
		Args: args,
	}
	if filepath.Base("cmd") == "cmd" {
		if lp, err := exec.LookPath("cmd"); err != nil {
			return err
		} else {
			cmd.Path = lp
		}
	}
	cmd.Start()
	var res string
	//等待浏览器打开完成
	for i := 0; i < 10; i++ {
		res, _, _, _ = big.HttpSend(&big.HttpParms{Url: "http://" + p.Ip + ":" + port + "/json"})
		if res != "" {
			break
		}
	}
	if res == "" {
		return errors.New("打开浏览器超时")
	}
	time.Sleep(2 * time.Second)
	return nil
}

/**
关闭浏览器，如果是隐身无痕模式则自动删除缓存文件
*/
func (p *Browser) CloseBrowser() {
	go func() {
		//先关闭全部标签
		tags := p.GetTagList("page")
		for _, v := range tags {
			p.CloseTag(v)
		}
		//如果是隐身无痕模式则删除缓存文件
		if p.Hide {
			port := strconv.Itoa(p.Port)
			//等待浏览器关闭完成
			for i := 0; i < 10; i++ {
				res, _, _, _ := big.HttpSend(&big.HttpParms{Url: "http://" + p.Ip + ":" + port + "/json"})
				if res == "" {
					break
				}
			}
			//尝试删除10次缓存
			for i := 0; i < 100; i++ {
				time.Sleep(5 * time.Second)
				err := os.RemoveAll(p.DataDir)
				if err == nil {
					break
				}
			}
		}
	}()
}

/**
取标签列表
传参：
	filter：过滤器，用于取指定的标签类型，传入空字符串表示不过滤，
			目前已知的标签类型有：page（页面）、background_page（插件）、iframe（内嵌页）
返回：
	获取到的标签列表切片
*/
func (p *Browser) GetTagList(filter string) []Tag {
	port := strconv.Itoa(p.Port)
	_, res, _, _ := big.HttpSend(&big.HttpParms{Url: "http://" + p.Ip + ":" + port + "/json"})
	tags := make([]Tag, 0)
	err := json.Unmarshal(res, &tags)
	if err != nil {
		println(err.Error())
	}
	newtags := make([]Tag, 0)
	for _, v := range tags {
		if v.Typ == filter {
			newtags = append(newtags, v)
		}
	}
	return newtags
}

/**
创建新标签
传参：
	url：新标签网址，可空
返回：
	Tag标签对象，失败则返回error错误信息
*/
func (p *Browser) NewTag(url string) (Tag, error) {
	port := strconv.Itoa(p.Port)
	_, res, _, _ := big.HttpSend(&big.HttpParms{Url: "http://" + p.Ip + ":" + port + "/json/new?" + url})
	var tag Tag
	err := json.Unmarshal(res, &tag)
	return tag, err
}

/**
关闭标签
传参：
	tag：Tag标签对象
返回：
	成功返回true
*/
func (p *Browser) CloseTag(tag Tag) bool {
	port := strconv.Itoa(p.Port)
	res, _, _, _ := big.HttpSend(&big.HttpParms{Url: "http://" + p.Ip + ":" + port + "/json/close/" + tag.Id})
	if res == "Target is closing" {
		return true
	}
	return false
}

/**
激活标签
传参：
	tag：Tag标签对象
返回：
	成功返回true
*/
func (p *Browser) ActivateTag(tag Tag) bool {
	port := strconv.Itoa(p.Port)
	res, _, _, _ := big.HttpSend(&big.HttpParms{Url: "http://" + p.Ip + ":" + port + "/json/activate/" + tag.Id})
	if res == "Target activated" {
		return true
	}
	return false
}
