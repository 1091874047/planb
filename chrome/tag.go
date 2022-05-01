//本库仅支持Windows平台的所有使用Chrome内核的浏览器,线程安全的
//Tag结构体用于操作浏览器页面标签
package chrome

import (
	"b/big"
	"encoding/json"
	"fmt"
	simplejson "github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Tag struct {
	Description          string                                    `json:"description"`          //标签描述
	DevtoolsFrontendUrl  string                                    `json:"devtoolsFrontendUrl"`  //前端开发工具链接
	FaviconUrl           string                                    `json:"faviconUrl"`           //标签页logo
	Id                   string                                    `json:"id"`                   //标签页ID
	ParentId             string                                    `json:"parentId"`             //标签页父ID
	Title                string                                    `json:"title"`                //标签页标题
	Typ                  string                                    `json:"type"`                 //标签页类型：page（页面）、background_page（插件）、iframe（内嵌页）
	Url                  string                                    `json:"url"`                  //标签页链接
	WebSocketDebuggerUrl string                                    `json:"webSocketDebuggerUrl"` //WebSocket链接
	dialogOpenEvent      func(d DialogOpen)                        //对话框打开监听事件，当网页弹出对话框(Alert,Confirm,Prompt,Beforeunload)时自动触发
	dialogCloseEvent     func(d DialogClose)                       //对话框关闭监听事件，当网页关闭对话框时自动触发
	Frames               []frame                                   //框架集合，首个成员为主框架信息，其他均为子框架，本对象内有框架信息及网页音视频图资源信息
	connect              bool                                      //连接成功为true，断开或失败为false
	ws                   *websocket.Conn                           //ws连接对象
	taskLock             sync.Mutex                                //互斥锁
	taskId               int                                       //任务ID，自增
	taskRsp              sync.Map                                  //任务响应结果，key是任务id
	contextIds           sync.Map                                  //标签上下文ID集合，key是frameId，value是contextId
	readyState           frameLoadStep                             //标签页是否加载完成，结构体中全部字段为true，且frameStoppedLoading字段等于2才算加载完成
	logs                 []ConsoleLog                              //控制台输出日志集合
	logsLock             sync.Mutex                                //控制台日志操作互斥锁
	haveDialog           bool                                      //是否存在对话框
	px                   int                                       //鼠标在浏览器的x坐标
	py                   int                                       //鼠标在浏览器的y坐标
	hookReqEvent         func(tag *Tag, request HookHttpRequest)   //拦截请求的回调方法
	hookRespEvent        func(tag *Tag, response HookHttpResponse) //拦截响应的回调方法
}

/**
连接标签，连接成功后才可对标签进行操作
默认可以不调用本方法进行连接，在调用Tag结构体其余方法时会自动连接。
返回：
	成功返回true，失败返回false和error错误信息
*/
func (p *Tag) Connect() (bool, error) {
	p.taskLock.Lock()
	//判断标签是否连接
	if p.connect {
		//已经连接过了
		p.taskLock.Unlock()
		return true, nil
	}
	u, err := url.Parse(p.WebSocketDebuggerUrl)
	if err != nil {
		p.taskLock.Unlock()
		return false, err
	}
	p.ws, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		p.taskLock.Unlock()
		return false, err
	}
	p.connect = true
	go p.onListenerMsg()
	p.taskLock.Unlock()
	//开启各项事件
	p.Call("Page.enable", nil)
	p.Call("Runtime.enable", nil)
	return true, nil
}

/**
关闭连接，关闭连接后，将无法对标签进行操作
传参：
	tagClose：是否关闭标签，如果不关闭标签，下次则还可以继续调用Connect方法连接本标签进行操作。
*/
func (p *Tag) Close(tagClose bool) {
	p.taskLock.Lock()
	p.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	p.ws.Close()
	p.connect = false
	p.taskLock.Unlock()
	if tagClose {
		big.HttpSend(&big.HttpParms{Url: "http://" + big.StrGetSub(p.WebSocketDebuggerUrl, "ws://", "/") + "/json/close/" + p.Id})
	}
}

/**
跳转到新的URL地址
传参：
	url：新的URL地址
	referer：来路设置，可空
	timeOut：等待页面加载完成时间，单位秒，为0表示不等待
	flag：本参数在timeOut大于0时生效，如果flag为空字符串，则会等待到页面转圈结束为止，如果flag有值，则会等待到网页源码存在该值为止
返回：
	成功返回true，否则返回false，注意：当需等待网页加载完成时，返回的true和false代表网页是否加载完成。
*/
func (p *Tag) TagJump(url string, referer string, timeOut int, flag string) bool {
	parm := make(map[string]interface{})
	parm["url"] = url
	if referer != "" {
		parm["referer"] = referer
	}
	p.readyState.loadEventFired = false
	p.readyState.frameStartedLoading = false
	p.readyState.frameStoppedLoading = false
	res, err := p.Call("Page.navigate", parm)
	if res == "" || err != nil {
		return false
	}
	if timeOut > 0 {
		return p.TagLoadWaitEnd(timeOut, flag)
	}
	return true
}

/**
重新载入（刷新）
传参：
	ignoreCache：是否忽略缓存
	scriptToEvaluateOnLoad：如果设置，脚本将被重新加载后注入被检查页面的所有帧，可空。
	timeOut：等待页面加载完成时间，单位秒，为0表示不等待
	flag：本参数在timeOut大于0时生效，如果flag为空字符串，则会等待到页面转圈结束为止，如果flag有值，则会等待到网页源码存在该值为止
返回：
	成功返回true，否则返回false，注意：当需等待网页加载完成时，返回的true和false代表网页是否加载完成。
*/
func (p *Tag) ReLoad(ignoreCache bool, scriptToEvaluateOnLoad string, timeOut int, flag string) bool {
	parm := make(map[string]interface{})
	parm["ignoreCache"] = ignoreCache
	if scriptToEvaluateOnLoad != "" {
		parm["scriptToEvaluateOnLoad"] = scriptToEvaluateOnLoad
	}
	res, err := p.Call("Page.reload", parm)
	if res == "" || err != nil {
		return false
	}
	if timeOut > 0 {
		return p.TagLoadWaitEnd(timeOut, flag)
	}
	return true
}

/**
更新标签中的框架信息，当页面结构发生改变，应先调用本方法更新框架信息后才可执行后续操作
返回：
	返回是否成功，更新成功后调用本对象的.Frames属性获取框架资源信息
*/
func (p *Tag) TagFrameUpdate() bool {
	res, err := p.Call("Page.getResourceTree", nil)
	if res == "" || err != nil {
		return false
	}
	jsonobj, err := simplejson.NewJson([]byte(res))
	if err != nil {
		return false
	}
	//清空框架信息
	p.Frames = make([]frame, 0)
	//先获取主框架
	newobj := jsonobj.Get("result").Get("frameTree").Get("frame")
	newobj.Set("resources", jsonobj.Get("result").Get("frameTree").Get("resources"))
	cid, _ := p.contextIds.Load(jsonobj.Get("result").Get("frameTree").Get("frame").Get("id").Interface()) //获取框架上下文ID
	newobj.Set("contextId", cid)
	jbyte, _ := newobj.MarshalJSON()
	var fra frame
	json.Unmarshal(jbyte, &fra)
	p.Frames = append(p.Frames, fra)
	//获取子框架
	jsonobj = jsonobj.Get("result").Get("frameTree").Get("childFrames")
	childFrames, _ := jsonobj.Array()
	for i, _ := range childFrames {
		newobj = jsonobj.GetIndex(i).Get("frame")
		newobj.Set("resources", jsonobj.GetIndex(i).Get("resources"))
		cid, _ = p.contextIds.Load(jsonobj.GetIndex(i).Get("frame").Get("id").Interface())
		newobj.Set("contextId", cid)
		jbyte, _ = newobj.MarshalJSON()
		var fra frame
		json.Unmarshal(jbyte, &fra)
		p.Frames = append(p.Frames, fra)
	}
	return true
}

/**
标签页是否已加载完成
返回：
	标签对象加载完成返回true，否则返回false
*/
func (p *Tag) TagLoadIsEnd() bool {
	return p.readyState.loadEventFired && p.readyState.frameStartedLoading && p.readyState.frameStoppedLoading
}

/**
等待标签页加载完成
传参：
	timeOut：最长等待时间，单位秒
	flag：如果flag为空字符串，则会等待到页面转圈结束为止，如果flag有值，则会等待到网页源码存在该值为止
返回：
	页面加载完成返回true，否则返回false
*/
func (p *Tag) TagLoadWaitEnd(timeOut int, flag string) bool {
	time.Sleep(1 * time.Second)
	timeOut--
	for i := 1; i <= timeOut; i++ {
		if flag == "" && p.TagLoadIsEnd() {
			time.Sleep(1 * time.Second)
			return true
		} else if flag != "" && p.TagFrameUpdate() {
			res, _ := p.EvalJs("document.getElementsByTagName('html')[0].innerHTML.indexOf('"+flag+"')", 0)
			if res.Value != "-1" {
				time.Sleep(1 * time.Second)
				return true
			}
		}
		time.Sleep(1 * time.Second)
	}
	return p.TagLoadIsEnd()
}

/**
取标签页面源码
传参：
	contextId：指定框架上下文id，传0表示默认主框架
返回：
	网页源码
*/
func (p *Tag) TagHtmlGet(contextId int) string {
	res, err := p.EvalJs("document.getElementsByTagName('html')[0].innerHTML", contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
提取网页资源，比如提取网页图片音视频、JS、CSS等文件内容
传参：
	frameId：指定框架的id，传空字符串表示默认主框架
	url：本对象.Frames[0].Resources[]结构体中的url
	base64：是否base64编码，如提取图片音视频则建议base64编码
返回：
	提取的结果
*/
func (p *Tag) TagResourceContentGet(frameId string, url string, base64 bool) string {
	if frameId == "" {
		frameId = p.Frames[0].Id
	}
	parm := make(map[string]interface{})
	parm["frameId"] = frameId
	parm["url"] = url
	res, err := p.Call("Page.getResourceContent", parm)
	if res == "" || err != nil {
		return ""
	}
	sjson, err := simplejson.NewJson([]byte(res))
	if err != nil {
		return ""
	}
	sjson = sjson.Get("result")
	content, _ := sjson.Get("content").String()
	if !strings.Contains(res, "base64Encoded\":true") && base64 {
		content = big.EnCodeBase64Str(content)
	}
	return content
}

/**
执行JS代码【精简版】
传参：
	jscode：JS代码，可以多行。
	contextId：指定框架上下文id，传0表示默认主框架
返回：
	EvalRes：EvalRes结构体用于保存执行结果，具体结果查看该结构体字段属性
	error：用于记录执行在Go中发生的错误
*/
func (p *Tag) EvalJs(jscode string, contextId int) (EvalRes, error) {
	if contextId == 0 {
		contextId = p.Frames[0].ContextId
	}
	return p.Evaluate(jscode, "ChromeRemoteObject00000000000000000000000000000000", true, true, contextId, true, false, false, false)
}

/**
拦截标签页对话框事件，当标签页弹出或关闭对话框(alert,confirm,prompt,beforeunload)时自动触发；
若想取消拦截，则再次调用本方法，dopen和dclose参数传nil即可取消拦截
传参：
	dopen：传入一个func函数，当对话框弹出时自动调用该函数，并将chrome.DialogOpen结构体对象作为形参传入，函数需定义形参，格式：func demo(d DialogOpen) {}
	dclose：传入一个func函数，当对话框关闭时自动调用该函数，并将chrome.DialogClose结构体对象作为形参传入，函数需定义形参，格式：func demo(d DialogClose) {}
注意：如果两个事件都拦截，那么不想拦截的事件函数可传nil
*/
func (p *Tag) TagDialogHook(dopen func(d DialogOpen), dclose func(d DialogClose)) {
	p.dialogOpenEvent = dopen
	p.dialogCloseEvent = dclose
}

/**
检查当前页面是否已弹出alert,confirm,prompt,beforeunload等提示框
返回：
	已弹出返回true，否则返回false
*/
func (p *Tag) TagDialogIsHave() bool {
	return p.haveDialog
}

/**
反馈对话框，接受或解除JavaScript启动的对话框（alert，confirm，prompt或onbeforeunload）
传参：
	accept：接收或解除，true为接收对话框，false为解除对话框
	promptText：输入文本，可空
返回：
	成功返回true，失败返回false
*/
func (p *Tag) TagDialogHandle(accept bool, promptText string) bool {
	parm := make(map[string]interface{})
	parm["accept"] = accept
	if promptText != "" {
		parm["promptText"] = promptText
	}
	res, err := p.Call("Page.handleJavaScriptDialog", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
CPU限速：启用CPU限制来模拟缓慢的CPU
传参：
	rate：节流率为减速因子（1为无油门，2为2倍减速等）
返回：
	成功返回true，失败返回false
*/
func (p *Tag) TagCPUThrottlingRateSet(rate float64) bool {
	parm := make(map[string]interface{})
	parm["rate"] = rate
	res, err := p.Call("Emulation.setCPUThrottlingRate", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
模拟地理位置，虚拟定位
传参：
	longitude：经度
	latitude：纬度
	accuracy：精度
返回：
	设置成功返回true，否则返回false
*/
func (p *Tag) TagGeolocationOverrideSet(longitude float64, latitude float64, accuracy float64) bool {
	parm := make(map[string]interface{})
	parm["longitude"] = longitude
	parm["latitude"] = latitude
	parm["accuracy"] = accuracy
	res, err := p.Call("Emulation.setGeolocationOverride", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
置标签页用户代理标识UA
传参：
	ua：需要设置的UserAgent
返回：
	设置成功返回true，否则返回false
*/
func (p *Tag) TagUserAgentSet(ua string) bool {
	parm := make(map[string]interface{})
	parm["userAgent"] = ua
	res, err := p.Call("Network.setUserAgentOverride", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
网页快照截图，拍摄一张当前页面的渲染图像
传参：
	format：存储格式jpeg或png
	quality：压缩质量，仅为jpeg格式下有效，取值范围0-100
	fromSurface：从表面(Surface)获取屏幕截图,而不是视图(View)
	x：截取指定区域，从x坐标开始，x、y、width、height均为0表示截取全网页
	y：截取指定区域，从y坐标开始，x、y、width、height均为0表示截取全网页
	width：截图指定区域的宽度，x、y、width、height均为0表示截取全网页
	height：截图指定区域的高度，x、y、width、height均为0表示截取全网页
返回：
	成功返回base64编码后的图片内容，失败返回空文本
*/
func (p *Tag) TagCaptureScreenshot(format string, quality int, fromSurface bool, x int, y int, width int, height int) string {
	parm := make(map[string]interface{})
	parm["format"] = format
	parm["quality"] = quality
	parm["fromSurface"] = fromSurface
	if x != 0 && y != 0 && width != 0 && height != 0 {
		parm["clip"] = map[string]interface{}{
			"x":      x,
			"y":      y,
			"width":  width,
			"height": height,
			"scale":  1,
		}
	}
	res, err := p.Call("Page.captureScreenshot", parm)
	if res == "" || err != nil {
		return ""
	}
	return big.StrGetSub(res, "{\"data\":\"", "\"}}")
}

/**
启用仿真模拟触点设备
传参：
	enable：是否启用仿真触摸事件，true为启用，false为不启用
	configuration：手势事件类型，可选值mobile, desktop
返回：
	成功返回true，否则返回false
*/
func (p *Tag) TagTouchEmulationEnabledSet(enable bool, configuration string) bool {
	parm := make(map[string]interface{})
	parm["enabled"] = enable
	parm["configuration"] = configuration
	res, err := p.Call("Emulation.setEmitTouchEventsForMouse", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
清除缓存
返回：
	成功返回true，否则返回false
*/
func (p *Tag) TagCacheClear() bool {
	//先判断浏览器是否支持清除缓存
	res, err := p.Call("Network.canClearBrowserCache", nil)
	if !strings.Contains(res, "true") {
		//不支持本操作
		return false
	}
	res, err = p.Call("Network.clearBrowserCache", nil)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
取控制台日志信息
传参：
	clear：是否获取后清理日志，清理后下次获取将返回新的日志，不清理下次获取将返回继续累加的日志。
返回：
	返回[]chrome.ConsoleLog切片
*/
func (p *Tag) TagConsoleLogsGet(clear bool) []ConsoleLog {
	p.logsLock.Lock()
	defer p.logsLock.Unlock()
	logs := p.logs[:]
	if clear {
		p.logs = p.logs[0:0]
	}
	return logs
}

/**
设置屏幕
覆盖设备屏幕尺寸的值（window.screen.width，window.screen.height，window.innerWidth，
window.innerHeight和“device-width”/“device-height”）相关的CSS媒体查询结果）
传参：
	width：宽度，覆盖的宽度值,像素单位,允许的范围(最小 0,最大 10000000),设置为0时,禁用覆盖
	height：高度，覆盖的高度值,像素单位,允许的范围(最小 0,最大 10000000),设置为0时,禁用覆盖
	deviceScaleFactor：设备比例因子，覆盖设备的比例因子值,设置为0时,禁用覆盖
	mobile：移动模式，是否模拟移动设备,这包含视口元标记,覆盖 滚动条,文本自动调整等
	flScale：缩放比例，应用于缩放生成的视图图像,忽略|fitWindow|模式
	screenWidth：屏幕宽度，覆盖屏幕宽度值（以像素为单位）（最小值0，最大10000000）。 只用于| mobile == true |
	screenHeight：屏幕高度，覆盖屏幕高度值（以像素为单位）（最小0，最大10000000）。 只用于| mobile == true |
	positionX：视图位置X，在屏幕上覆盖视图X位置（以像素为单位）（最小值0，最大值10000000）。 只用于| mobile == true |
	positionY：视图位置Y，在屏幕上覆盖视图Y位置（以像素为单位）（最小值0，最大值10000000）。 只用于| mobile == true |
	screenOrientationType：屏幕方向类型，设置屏幕方向,可空,可选的值: portraitPrimary, portraitSecondary, landscapePrimary, landscapeSecondary
	screenOrientationAngle：屏幕角度，所处方向的角度
返回：
	成功返回true，否则返回false
*/
func (p *Tag) TagScreenSet(width int, height int, deviceScaleFactor float64, mobile bool, flScale float64, screenWidth int, screenHeight int, positionX int, positionY int, screenOrientationType string, screenOrientationAngle int) bool {
	parm := make(map[string]interface{})
	parm["width"] = width
	parm["height"] = height
	parm["deviceScaleFactor"] = deviceScaleFactor
	parm["mobile"] = mobile
	parm["scale"] = flScale
	parm["screenWidth"] = screenWidth
	parm["screenHeight"] = screenHeight
	parm["positionX"] = positionX
	parm["positionY"] = positionY
	parm["fitWindow"] = true //超过可用浏览器窗口区域的视图是否应缩小以适应，默认为True
	screenOrientation := make(map[string]interface{})
	screenOrientation["angle"] = screenOrientationAngle
	if screenOrientationType != "" {
		screenOrientation["type"] = screenOrientationType
	}
	parm["screenOrientation"] = screenOrientation
	res, err := p.Call("Emulation.setDeviceMetricsOverride", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
清除设置屏幕的指标值
返回：
	成功返回true，否则返回false
*/
func (p *Tag) TagScreenClear() bool {
	res, err := p.Call("Emulation.clearDeviceMetricsOverride", nil)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
浏览器窗口位置大小状态设置
传参：
	left：设置浏览器离屏幕左边的位置，单位px，传-1表示不设置
	top：设置浏览器离屏幕顶边的位置，单位px，传-1表示不设置
	width：设置浏览器宽度，单位px，传-1表示不设置
	height：设置浏览器高度，单位px，传-1表示不设置
	windowState：窗口状态，可选值normal（普通）, minimized（最小化）, maximized（最大化）, fullscreen（全屏），传空字符串表示不设置
返回：
	成功返回true，否则返回false
*/
func (p *Tag) WindowSet(left int, top int, width int, height int, windowState string) bool {
	parm := make(map[string]interface{})
	bounds := make(map[string]interface{})
	if left != -1 {
		bounds["left"] = left
	}
	if top != -1 {
		bounds["top"] = top
	}
	if width != -1 {
		bounds["width"] = width
	}
	if height != -1 {
		bounds["height"] = height
	}
	if windowState != "" {
		bounds["windowState"] = height
	}
	parm["windowId"] = 1
	parm["bounds"] = bounds
	res, err := p.Call("Browser.setWindowBounds", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
窗口滚动条距离设置
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	xNum：文档向右滚动的像素数
	yNum：文档向下滚动的像素数
*/
func (p *Tag) WindowScrollBySet(contextId int, xNum int, yNum int) bool {
	_, err := p.EvalJs(fmt.Sprintf("window.scrollBy(%d,%d)", xNum, yNum), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
窗口滚动条位置设置
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	x：要在窗口文档显示区左上角显示的文档的x坐标
	y：要在窗口文档显示区左上角显示的文档的y坐标
*/
func (p *Tag) WindowScrollToSet(contextId int, x int, y int) bool {
	_, err := p.EvalJs(fmt.Sprintf("window.scrollTo(%d,%d)", x, y), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
Cookies清空，清除当前页面中Cookies数据
返回：
	成功返回true，否则返回false
*/
func (p *Tag) CookiesClear() bool {
	//先判断浏览器是否支持清除缓存
	res, err := p.Call("Network.canClearBrowserCookies", nil)
	if !strings.Contains(res, "true") {
		//不支持本操作
		return false
	}
	res, err = p.Call("Network.clearBrowserCookies", nil)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
Cookies获取
传参：
	domain：域名，如果要获取顶级域名的Cookies请填写：http://.baidu.com，不区分http和https，两者结果都一样，传空字符串表示获取获取所有域名的cookies
返回2参数：
	参数1：为Cookie结构体的切片，表示多个Cookie
	参数2：为Cookie的拼接字符串，可用于浏览器或本big包下的HttpSend的Cookies字段,如果想获取单个Cookie值，请用big.HttpGetCookie函数获取
	参数3：为错误信息
*/
func (p *Tag) CookiesGet(domain string) ([]Cookie, string, error) {
	var res string
	var err error
	parm := make(map[string]interface{})
	if domain != "" {
		parm["urls"] = [1]string{domain}
		res, err = p.Call("Network.getCookies", parm)
	} else {
		res, err = p.Call("Network.getAllCookies", nil)
	}
	if err != nil {
		return nil, "", err
	}
	sjson, err := simplejson.NewJson([]byte(res))
	if err != nil {
		return nil, "", err
	}
	bjson, err := sjson.Get("result").Get("cookies").MarshalJSON()
	if err != nil {
		return nil, "", err
	}
	var cookies []Cookie
	err = json.Unmarshal(bjson, &cookies)
	if err != nil {
		return nil, "", err
	}
	res = ""
	for _, v := range cookies {
		res = res + v.Name + "=" + v.Value + "; "
	}
	if res != "" {
		res = res[0 : len(res)-2]
	}
	return cookies, res, nil
}

/**
Cookies删除单个
传参：
	url：domain+Path的字符串拼接，也可以只传domain
	name：名称，要删除的Cookie名称
返回：
	成功返回true，否则返回false
*/
func (p *Tag) CookiesDel(url string, name string) bool {
	parm := make(map[string]interface{})
	parm["url"] = url
	parm["name"] = name
	res, err := p.Call("Network.deleteCookies", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
Cookies单个设置，如不存在，则创建
传参：
	cookie：欲设置的Cookie，请使用chrome.Cookie赋值并传入
返回：
	成功返回true，否则返回false
*/
func (p *Tag) CookiesSet(cookie Cookie) bool {
	parm := make(map[string]interface{})
	bres, _ := json.Marshal(&cookie)
	json.Unmarshal(bres, &parm)
	res, err := p.Call("Network.setCookie", parm)
	if !strings.Contains(res, "true") || err != nil {
		return false
	}
	return true
}

/**
Cookies批量设置【文本形式】
传参：
	url：domain+Path的字符串拼接，也可以只传domain，需要加http。
	domain：域名，如果要设置顶级域名的Cookies请填写：.baidu.com，不需要加http。
	path：路径，一般为“/”
	secure：是否为https协议，true为只能用于https协议
	expires：过期时间，单位毫秒，可空，传0表示365天过期
	httponly：是否禁止通过JS获取该Cookie，true为禁止，false为允许
	cookies：字符串格式的cookies，类似于：name=value; name=value这种格式
返回：
	成功返回true，否则返回false
*/
func (p *Tag) CookiesSetStr(url string, domain string, path string, secure bool, expires float64, httponly bool, cookies string) bool {
	cookies = strings.TrimSpace(cookies)
	arr := strings.Split(cookies, "; ")
	for _, v := range arr {
		if expires == 0 {
			_, exp := big.TimeStamp(13)
			expires = float64(exp + 31536000000)
		}
		cookie := Cookie{
			Name:     big.StrGetLeft(v, "="),
			Value:    big.StrGetRight(v, "="),
			Domain:   domain,
			Path:     path,
			Expires:  expires,
			Size:     len(v) - 1,
			HTTPOnly: httponly,
			Secure:   secure,
			Session:  false,
			Priority: "Medium",
			Url:      url,
			SameSite: "Lax",
		}
		if !p.CookiesSet(cookie) {
			return false
		}
	}
	return true
}

/**
Cookies批量设置【[]chrome.Cookie结构体切片形式】
传参：
	cookies：[]chrome.Cookie结构体切片
返回：
	成功返回true，否则返回false
*/
func (p *Tag) CookiesSetStu(cookies []Cookie) bool {
	for _, v := range cookies {
		if !p.CookiesSet(v) {
			return false
		}
	}
	return true
}

/**
CSS取表单（FORM）提交地址
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径
返回：
	结果
*/
func (p *Tag) CssFormUrlGet(contextId int, selector string) string {
	if contextId < 0 || selector == "" {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').action", selector), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
CSS置表单（FORM）提交地址
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径
	url：欲设置的表单提交地址
返回：
	设置成功，返回设置的url表单提交地址
*/
func (p *Tag) CssFormUrlSet(contextId int, selector string, url string) string {
	if contextId < 0 || selector == "" || url == "" {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').action='%s'", selector, url), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
CSS表单（FORM）重置
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径
返回：
	成功返回true，否则返回false
*/
func (p *Tag) CssFormReset(contextId int, selector string) bool {
	if contextId < 0 || selector == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').reset()", selector), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
CSS表单（FORM）提交
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径
返回：
	成功返回true，否则返回false
*/
func (p *Tag) CssFormSubmit(contextId int, selector string) bool {
	if contextId < 0 || selector == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').submit()", selector), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
CSS下拉菜单（SELECT）取表项文本
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	index：表项下标
返回：
	结果
*/
func (p *Tag) CssSelectTextGet(contextId int, selector string, index int) string {
	if contextId < 0 || selector == "" || index == -1 {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').options[%d].text", selector, index), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
CSS下拉菜单（SELECT）取表项数
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	表项总数
*/
func (p *Tag) CssSelectIndexLength(contextId int, selector string) int {
	if contextId < 0 || selector == "" {
		return 0
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').length", selector), contextId)
	if err != nil {
		return 0
	}
	ok, err := strconv.Atoi(res.Value)
	if err != nil {
		return 0
	}
	return ok
}

/**
CSS下拉菜单（SELECT）取现行选中项
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	现在选中的表项下标
*/
func (p *Tag) CssSelectNowIndexGet(contextId int, selector string) int {
	if contextId < 0 || selector == "" {
		return 0
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').selectedIndex", selector), contextId)
	if err != nil {
		return 0
	}
	ok, err := strconv.Atoi(res.Value)
	if err != nil {
		return 0
	}
	return ok
}

/**
CSS下拉菜单（SELECT）置现行选中项
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	index：欲设置的选中项下标
返回：
	成功返回true，否则返回false
*/
func (p *Tag) CssSelectNowIndexSet(contextId int, selector string, index int) bool {
	if contextId < 0 || selector == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').selectedIndex=%d", selector, index), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
CSS表格（Table）取单元格文本
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	row：第几行
	cell：第几列
返回：
	单元格文本
*/
func (p *Tag) CssTableCellTextGet(contextId int, selector string, row int, cell int) string {
	if contextId < 0 || selector == "" || row == -1 || cell == -1 {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').rows[%d].cells[%d].innerText", selector, row, cell), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
CSS表格（Table）取单元格源码
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	row：第几行
	cell：第几列
返回：
	单元格源码
*/
func (p *Tag) CssTableCellHtmlGet(contextId int, selector string, row int, cell int) string {
	if contextId < 0 || selector == "" || row == -1 || cell == -1 {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').rows[%d].cells[%d].innerHTML", selector, row, cell), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
CSS表格（Table）取单元格行数
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	单元格行数
*/
func (p *Tag) CssTableRows(contextId int, selector string) int {
	if contextId < 0 || selector == "" {
		return 0
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').rows.length", selector), contextId)
	if err != nil {
		return 0
	}
	ok, err := strconv.Atoi(res.Value)
	if err != nil {
		return 0
	}
	return ok
}

/**
CSS表格（Table）取单元格列数
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	单元格列数
*/
func (p *Tag) CssTableCells(contextId int, selector string) int {
	if contextId < 0 || selector == "" {
		return 0
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').cells.length", selector), contextId)
	if err != nil {
		return 0
	}
	ok, err := strconv.Atoi(res.Value)
	if err != nil {
		return 0
	}
	return ok
}

/**
元素触发单击事件
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	成功返回true，否则返回false
*/
func (p *Tag) DomClick(contextId int, selector string) bool {
	if contextId < 0 || selector == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').click()", selector), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
元素定位
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	x：元素x坐标
	y：元素y坐标
*/
func (p *Tag) DomPosition(contextId int, selector string) (x int, y int) {
	if contextId < 0 || selector == "" {
		return 0, 0
	}
	const jscode string = `function taptap(){
var n = document.querySelector("%s").getBoundingClientRect();
var x = (n.left + document.documentElement.scrollLeft).toFixed();
var y = (n.top + document.documentElement.scrollTop).toFixed();
return x+","+y;
}
taptap();`
	res, err := p.EvalJs(fmt.Sprintf(jscode, selector), contextId)
	if err != nil {
		return 0, 0
	}
	res.Value = strings.ReplaceAll(res.Value, "\"", "")
	resArr := strings.Split(res.Value, ",")
	x, _ = strconv.Atoi(resArr[0])
	y, _ = strconv.Atoi(resArr[1])
	return
}

/**
元素焦点激活
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	成功返回true，否则返回false
*/
func (p *Tag) DomFocus(contextId int, selector string) bool {
	if contextId < 0 || selector == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').focus()", selector), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
元素焦点失去
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	成功返回true，否则返回false
*/
func (p *Tag) DomBlur(contextId int, selector string) bool {
	if contextId < 0 || selector == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').blur()", selector), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
元素取HTML
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	HTML源码
*/
func (p *Tag) DomHtmlGet(contextId int, selector string) string {
	if contextId < 0 || selector == "" {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').innerHTML", selector), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
元素置HTML
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	html：欲设置的html源码
返回：
	成功返回true，否则返回false
*/
func (p *Tag) DomHtmlSet(contextId int, selector string, html string) bool {
	if contextId < 0 || selector == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').innerHTML='%s'", selector, html), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
元素取属性值
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	name：属性名
返回：
	结果
*/
func (p *Tag) DomAttributeGet(contextId int, selector string, name string) string {
	if contextId < 0 || selector == "" || name == "" {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').getAttribute('%s')", selector, name), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
元素置属性值
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	name：属性名
	val：属性值
返回：
	成功返回true，否则返回false
*/
func (p *Tag) DomAttributeSet(contextId int, selector string, name string, val string) bool {
	if contextId < 0 || selector == "" || name == "" || val == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').setAttribute('%s','%s')", selector, name, val), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
元素取文本
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	结果
*/
func (p *Tag) DomTextGet(contextId int, selector string) string {
	if contextId < 0 || selector == "" {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').innerText", selector), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
元素置文本
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	text：欲设置的元素文本
返回：
	成功返回true，否则返回false
*/
func (p *Tag) DomTextSet(contextId int, selector string, text string) bool {
	if contextId < 0 || selector == "" || text == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').innerText='%s'", selector, text), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
元素取值
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
返回：
	结果
*/
func (p *Tag) DomValGet(contextId int, selector string) string {
	if contextId < 0 || selector == "" {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').value", selector), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
元素置值
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	val：欲设置的元素值
返回：
	成功返回true，否则返回false
*/
func (p *Tag) DomValSet(contextId int, selector string, val string) bool {
	if contextId < 0 || selector == "" || val == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').value='%s'", selector, val), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
元素执行指定事件
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	name：欲执行的事件，需自己加括号，例如：click()
返回：
	执行返回结果
*/
func (p *Tag) DomOnEvent(contextId int, selector string, name string) string {
	if contextId < 0 || selector == "" || name == "" {
		return ""
	}
	res, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').%s", selector, name), contextId)
	if err != nil {
		return ""
	}
	return res.Value
}

/**
元素复选框置状态
传参：
	contextId：指定框架上下文id，传0表示默认主框架
	selector：CSS选择器路径，用于选择SELECT标签
	check：是否选中，true为选中，false为未选中
返回：
	成功返回true，否则返回false
*/
func (p *Tag) DomCheckboxSet(contextId int, selector string, check string) bool {
	if contextId < 0 || selector == "" || check == "" {
		return false
	}
	_, err := p.EvalJs(fmt.Sprintf("document.querySelector('%s').checked=%s", selector, check), contextId)
	if err != nil {
		return false
	}
	return true
}

/**
发送按键事件
传参：
	typ：按键类型,可选值: keyDown, keyUp, rawKeyDown, char
	modifiers：功能键，可选值：Alt = 1，Ctrl = 2，Meta/Command = 4，Shift = 8，默认为 = 0
	windowsVirtualKeyCode：Windows虚拟键代码（默认值：0）
	isKeypad：事件是否从小键盘生成，true为是，false为否，一般为false
	isSysttemKey：事件是否是系统键事件，true为是，false为否，一般为false
返回：
	成功返回true，否则返回false
*/
func (p *Tag) InputSendKey(typ string, modifiers int, windowsVirtualKeyCode int, isKeypad bool, isSysttemKey bool) bool {
	return p.keyEvent(typ, modifiers, windowsVirtualKeyCode, "", isKeypad, isSysttemKey)
}

/**
发送文本事件
传参：
	typ：按键类型,可选值: keyDown, keyUp, rawKeyDown, char
	modifiers：功能键，可选值：Alt = 1，Ctrl = 2，Meta/Command = 4，Shift = 8，默认为 = 0
	text：欲发送的文本
	isKeypad：事件是否从小键盘生成，true为是，false为否，一般为false
	isSysttemKey：事件是否是系统键事件，true为是，false为否，一般为false
返回：
	成功返回true，否则返回false
*/
func (p *Tag) InputSendText(typ string, modifiers int, text string, isKeypad bool, isSysttemKey bool) bool {
	return p.keyEvent(typ, modifiers, 0, text, isKeypad, isSysttemKey)
}

/**
发送鼠标事件
传参：
	typ：鼠标类型,可选值: mousePressed, mouseReleased, mouseMoved, mouseWheel
	x：事件的X坐标相对于CSS像素中的主框架的视口
	y：事件的Y坐标相对于CSS像素中的主框架视口，0表示视口的顶部，Y随着进入视口底部而增加。
	modifiers：功能键，可选值：Alt = 1，Ctrl = 2，Meta/Command = 4，Shift = 8，默认为 = 0
	button：鼠标按键,可选值: none, left, middle, right
	clickCount：点击次数，单击鼠标按钮的次数（默认值：0）
	deltaX：鼠标滚轮事件的CSS像素中的X delta（默认值：0）
	deltaY：鼠标滚轮事件的CSS像素中的Y delta（默认值：0）
返回：
	成功返回true，否则返回false
*/
func (p *Tag) InputSendMouse(typ string, x int, y int, modifiers int, button string, clickCount int, deltaX int, deltaY int) bool {
	parm := make(map[string]interface{})
	parm["type"] = typ
	parm["x"] = x
	parm["y"] = y
	parm["modifiers"] = modifiers
	_, timestamp := big.TimeStamp(10)
	parm["timestamp"] = timestamp
	parm["button"] = button
	parm["clickCount"] = clickCount
	parm["deltaX"] = deltaX
	parm["deltaY"] = deltaY
	res, err := p.Call("Input.dispatchMouseEvent", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
模拟鼠标轨迹移动到指定坐标
传参：
	x：事件的X坐标相对于CSS像素中的主框架的视口
	y：事件的Y坐标相对于CSS像素中的主框架视口，0表示视口的顶部，Y随着进入视口底部而增加。
*/
func (p *Tag) InputMouseMove(x int, y int) {
	xArr := make([]int, 0)
	yArr := make([]int, 0)
	x = x - p.px
	y = y - p.py
	positiveX := true
	positiveY := true
	if x < 0 {
		positiveX = false
	}
	if y < 0 {
		positiveY = false
	}
	x = int(math.Abs(float64(x)))
	y = int(math.Abs(float64(y)))
	p.InputSendMouse("mouseMoved", x, y, 0, "none", 0, 0, 0)
	for {
		if x > 0 {
			x--
			if positiveX {
				p.px++
			} else {
				p.px--
			}
		}

		if y > 0 {
			y--
			if positiveY {
				p.py++
			} else {
				p.py--
			}
		}
		xArr = append(xArr, p.px)
		yArr = append(yArr, p.py)
		if x == 0 && y == 0 {
			break
		}
	}
	for i := 0; i < len(xArr); i++ {
		p.InputSendMouse("mouseMoved", xArr[i], yArr[i], 0, "none", 0, 0, 0)
	}
}

/**
发送触摸点事件
传参：
	typ：触摸事件的类型,TouchEnd和TouchCancel不能包含任何触摸点,而TouchStart和TouchMove必须至少包含一个,可选的值:touchStart，touchEnd，touchMove，touchCancel
	touchs：触摸设备上的活动触摸点。每个任何改变点（与序列中的先前触摸事件相比）产生一个事件，逐个模拟按压/移动/释放点。
	modifiers：功能键，可选值：Alt = 1，Ctrl = 2，Meta/Command = 4，Shift = 8，默认为 = 0
返回：
	成功返回true，否则返回false
*/
func (p *Tag) InputSendTouch(typ string, touchs []Touch, modifiers int) bool {
	_, timestamp := big.TimeStamp(10)
	parm := make(map[string]interface{})
	parm["type"] = typ
	parm["touchPoints"] = touchs
	parm["modifiers"] = modifiers
	parm["timestamp"] = timestamp
	res, err := p.Call("Input.dispatchTouchEvent", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
发送合成捏合手势【实验性功能】
传参：
	x：X坐标,相当于CSS像素,相对于浏览器视口左上角
	y：Y坐标,相当于CSS像素,相对于浏览器视口左上角
	scaleFactor：缩放后的相对缩放因子（> 1.0放大，<1.0缩小）
	relativeSpeed：相对指针速度（以像素为单位）（默认值：800）
	gestureSourceType：要生成哪种类型的输入事件（默认值：“default”，它会查询平台的首选输入类型),可选值:default, touch, mouse
返回：
	成功返回true，否则返回false
*/
func (p *Tag) InputSendPinch(x float64, y float64, scaleFactor float64, relativeSpeed int, gestureSourceType string) bool {
	parm := make(map[string]interface{})
	parm["x"] = x
	parm["y"] = y
	parm["scaleFactor"] = scaleFactor
	parm["relativeSpeed"] = relativeSpeed
	parm["gestureSourceType"] = gestureSourceType
	res, err := p.Call("Input.synthesizePinchGesture", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
发送合成滚动手势
传参：
	x：X坐标，相当于CSS像素，相对于浏览器视口左上角
	y：Y坐标，相当于CSS像素，相对于浏览器视口左上角
	xDistance：x滚动距离，沿X轴滚动的距离（正向左滚动）
	yDistance：y滚动距离，沿Y轴滚动的距离（向上滚动）
	xOverscroll：x滚动增量，除了给定距离之外，沿X轴向后滚动的附加像素数
	yOverscroll：y滚动增量，除了给定距离之外，沿Y轴向后滚动的附加像素数
	preventFling：不清楚意义，默认值为true，建议为true
	speed：速度，以秒为单位的扫描速度（默认值：800）
	gestureSourceType：手势类型，要生成哪种类型的输入事件（默认值：“default”，它会查询平台的首选输入类型）,可选值:default, touch, mouse
	repeatCount：重复次数，重复手势的次数（默认值：0）
	repeatDelayMs：每次重复之间的毫秒数延迟（默认值：250）
返回：
	成功返回true，否则返回false
*/
func (p *Tag) InputSendRoll(x float64, y float64, xDistance float64, yDistance float64, xOverscroll float64, yOverscroll float64, preventFling bool, speed int, gestureSourceType string, repeatCount int, repeatDelayMs int) bool {
	parm := make(map[string]interface{})
	parm["x"] = x
	parm["y"] = y
	parm["xDistance"] = xDistance
	parm["yDistance"] = yDistance
	parm["xOverscroll"] = xOverscroll
	parm["yOverscroll"] = yOverscroll
	parm["preventFling"] = preventFling
	parm["speed"] = speed
	parm["gestureSourceType"] = gestureSourceType
	parm["repeatCount"] = repeatCount
	parm["repeatDelayMs"] = repeatDelayMs
	res, err := p.Call("Input.synthesizeScrollGesture", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
发送合成点击手势
传参：
	x：X坐标，相当于CSS像素，相对于浏览器视口左上角
	y：Y坐标，相当于CSS像素，相对于浏览器视口左上角
	duration：持续时间，达阵和触摸事件之间的持续时间（ms）（默认值：50）
	tapCount：Tap次数，执行Tap的次数（例如2次双击，默认值为1）
	gestureSourceType：手势类型，要生成哪种类型的输入事件（默认值：“default”，它会查询平台的首选输入类型）,可选值:default, touch, mouse
返回：
	成功返回true，否则返回false
*/
func (p *Tag) InputSendClick(x float64, y float64, duration int, tapCount int, gestureSourceType string) bool {
	parm := make(map[string]interface{})
	parm["x"] = x
	parm["y"] = y
	parm["duration"] = duration
	parm["tapCount"] = tapCount
	parm["gestureSourceType"] = gestureSourceType
	res, err := p.Call("Input.synthesizeTapGesture", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
开启拦截网络请求，目前仅支持拦截不支持修改数据
传参：
	req：传入格式为func(tag *chrome.Tag, request chrome.HookHttpRequest)的函数，当有请求时会自动触发
	resp：传入格式为func(tag *chrome.Tag, response HookHttpResponse)的函数，当有响应时会自动触发
*/
func (p *Tag) HookHttpEn(req func(tag *Tag, request HookHttpRequest), resp func(tag *Tag, response HookHttpResponse)) {
	p.hookReqEvent = req
	p.hookRespEvent = resp
	p.Call("Network.enable", nil)
}

/**
禁用拦截网络请求
*/
func (p *Tag) HookHttpDis() {
	p.Call("Network.disable", nil)
}

/**
获取POST数据内容
传参：
	requestId：该参数在拦截的网络请求形参中有，传入对应相同名的参数即可
返回：
	返回chrome.HookHttpBody结构体对象，具体字段属性含义查看源码定义时的注释
*/
func (p *Tag) HookGetBody(requestId string) HookHttpBody {
	entity := HookHttpBody{}
	parm := make(map[string]interface{})
	parm["requestId"] = requestId
	res, err := p.Call("Network.getResponseBody", parm)
	if res == "" || err != nil {
		return entity
	}
	json.Unmarshal([]byte(res), &entity)
	return entity
}
