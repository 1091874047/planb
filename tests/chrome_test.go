package tests

import (
	"b/chrome"
	"fmt"
	"testing"
	"time"
)

func TestChrome(t *testing.T) {

	//Browser结构体每个属性含义，可点进去看，有注释
	//br := chrome.Browser{
	//	Ip:          "localhost",
	//	Port:        0,
	//	Path:        "",
	//	DefaultUrl:  "https://gitee.com/nanqis/bigtires",
	//	Width:       0,
	//	Height:      0,
	//	Maximized:   false,
	//	Hide:        true,
	//	DataDir:     "",
	//	Proxy:       "",
	//	Headless:    false,
	//	RemoteDebug: false,
	//	DisSecurity: false,
	//}
	br := chrome.Browser{
		Ip:          "localhost",
		Port:        10236,
		Path:        "",
		DefaultUrl:  "https://gitee.com/nanqis/bigtires",
		Width:       0,
		Height:      0,
		Maximized:   false,
		Hide:        false,
		DataDir:     "",
		Proxy:       "",
		Headless:    false,
		RemoteDebug: false,
		DisSecurity: false,
	}
	//打开浏览器
	//br.OpenBrowser()
	//关闭浏览器
	//defer br.CloseBrowser()
	//获取浏览器标签页对象
	tag := br.GetTagList("page")[0]
	//请求拦截
	tag.HookHttpEn(hkreq, hkResp)
	time.Sleep(2 * time.Second)
	//标签页地址跳转
	tag.TagJump("https://www.southwest.com/air/booking/select.html?adultPassengersCount=1&departureDate=2021-05-30&departureTimeOfDay=ALL_DAY&destinationAirportCode=LGA&fareType=USD&int=HOMEQBOMAIR&originationAirportCode=ORD&passengerType=ADULT&reset=true&returnDate=&returnTimeOfDay=ALL_DAY&tripType=oneway", "", 10, "")
	//更新页面框架信息，当页面发生改动后需调用本方法更新框架信息，框架信息包含了网页音视频图资源，框架ID等
	tag.TagFrameUpdate()
	//tag.TagJump("https://browserleaks.com/canvas","",10,"")
	//println(tag.Url, tag.Title)
	//取网页源码
	//html := tag.TagHtmlGet(0)
	//println(html)
	//tag.TagJump("https://gitee.com/nanqis/bigtires", "", 10)
	//println(tag.Url, tag.Title)
	//拦截页面对话框事件
	//tag.TagDialogHook(openDia, CloseDia)
	//res, err := tag.EvalJs("alert('测试内容')", 0)
	//使用标签页环境执行JS代码，可多行
	//res, err := tag.EvalJs("true", 0)
	//res, err := tag.EvalJs("document.querySelector('.am-dialog-text').textContent", 0)
	//if err == nil {
	//	println(res.Value)
	//}
	//清除缓存
	//tag.TagCacheClear()
	//清除Cookies
	//tag.CookiesClear()
	//获取Cookies
	//cookieArr, cookies, _ := tag.CookiesGet("http://.baidu.com")
	//for i, v := range cookieArr {
	//	println(i, v.Name, v.Value)
	//}
	//设置Cookies
	//cookies := "lang_type=zh-CN; Hm_lvt_b71d23d488cc70a042ad30fdcb0e962d=1618191547; remember-me-token=; OZ_SI_2074=sTime=1619484329&sIndex=2; OZ_1U_2074=vid=vfd1a659834f3e.0&ctime=1619484330&ltime=1619484329; OZ_1Y_2074=erefer=https%3A//www.baidu.com/s%3Fwd%3D%25E8%25A5%25BF%25E9%2583%25A8%25E8%2588%25AA%25E7%25A9%25BA%26ie%3DUTF-8&eurl=http%3A//www.westair.cn/user/register&etime=1619484329&ctime=1619484330&ltime=1619484329&compid=2074; JSESSIONID=V8vhXTle6JoSSgJq3nP5fL8q.CMS_MHServer2; user-token=eyJhbGciOiJIUzUxMiJ9.eyJ1c2VyVHlwZSI6IlVTRVJOQU1FIiwiYWNjb3VudFR5cGUiOiIxIiwiY2FjaGVLZXkiOiIyMTczNGZhZi0zYTMzLTRmNmQtYTE2OS1lODA2MWVhZjczYTkiLCJzdWIiOiJ0b25nMTIxNiIsImNyZWF0ZWRUaW1lIjoxNjE5NDg0NTE5NDIxLCJleHAiOjE2MTk1NzA5MTl9.HlvjpFo4eUzEquNwYEiWiGzi-OchrszTg-wFccBfA5X8UQy_VYpkqAvfwP2mKVzy4qTkboT5fCZ2L9LOw1m_9w; user-token=eyJhbGciOiJIUzUxMiJ9.eyJ1c2VyVHlwZSI6IlVTRVJOQU1FIiwiYWNjb3VudFR5cGUiOiIxIiwiY2FjaGVLZXkiOiIyMTczNGZhZi0zYTMzLTRmNmQtYTE2OS1lODA2MWVhZjczYTkiLCJzdWIiOiJ0b25nMTIxNiIsImNyZWF0ZWRUaW1lIjoxNjE5NDg0NTE5NDIxLCJleHAiOjE2MTk1NzA5MTl9.HlvjpFo4eUzEquNwYEiWiGzi-OchrszTg-wFccBfA5X8UQy_VYpkqAvfwP2mKVzy4qTkboT5fCZ2L9LOw1m_9w"
	//tag.CookiesSetStr("http://.westair.cn", ".westair.cn", "/", false, 0, false, cookies)
	//tag.TagJump("https://new.westair.cn/profile/inforsetup", "", 10, "")
	//println(cookies)
	//tag.CookiesDel("http://.baidu.com","BIDUPSID")
	//取控制台输出日志
	//log := tag.TagConsoleLogsGet(true)
	//for i, v := range log {
	//	println(i, v.Value.(string))
	//}
	//刷新网页
	//tag.ReLoad(true, "", 5)
	//time.Sleep(1 * time.Second)
	//tag.TagFrameUpdate()
	//设置滚动条位置，以此来设置浏览器显示的位置
	//tag.WindowScrollToSet(0, 0, 2000)
	//定位元素位置
	//x, y := tag.DomPosition(0, "#kw")
	//输入字符
	//tag.DomFocus(0, "#kw")
	//tag.InputKeySend("keyDown", 0, big.SysKeyCode("I"), false, false)
	//tag.InputKeySend("keyUp", 0, big.SysKeyCode("I"), false, false)
	//println(x,y)
	//鼠标操作
	//tag.InputSendMouse("mouseMoved", 100,100,0, "none",0,0,0)
	//鼠标轨迹模拟
	//tag.InputMouseMove(50,50)
	//模拟触摸点
	//touchs := make([]chrome.Touch,0)
	//for i := 0; i < 5; i++ {
	//	t := chrome.Touch{
	//		X:             float64(i),
	//		Y:             float64(i),
	//		RadiusX:       1.0,
	//		RadiusY:       1.0,
	//		RotationAngle: 0.0,
	//		Force:         1.0,
	//		Id:            float64(i),
	//	}
	//	touchs = append(touchs, t)
	//}
	//tag.InputSendTouch("TouchMove", touchs, 0)
	//提取支付宝验证码
	//codeurl := ""
	//for _, v := range tag.Frames[0].Resources{
	//	if strings.Contains(v.Url, "omeo.alipay.com/service/checkcode") {
	//		codeurl = v.Url
	//		break
	//	}
	//}
	//println(tag.TagResourceContentGet("", codeurl, true))
	//设置浏览器位置和大小
	//tag.WindowSet(-1,-1,500,500,"")
	//br.CloseBrowser()
	time.Sleep(9999 * time.Second)
	println("执行完成")
}

func openDia(d chrome.DialogOpen) {
	println(d.Msg)
}

func CloseDia(d chrome.DialogClose) {
	println(d.Result)
	println(d.UserInput)
}

func hkreq(tag *chrome.Tag, req chrome.HookHttpRequest) {
	if req.Params.Request.Url == "https://www.southwest.com/api/air-booking/v1/air-booking/page/air/booking/shopping" {
		res := ""
		for k, v := range req.Params.Request.Headers {
			res += k + ": " + v + "\r\n"
		}
		fmt.Println(res)
	}
}

func hkResp(tag *chrome.Tag, resp chrome.HookHttpResponse) {
	if resp.Params.Response.Url == "https://www.southwest.com/api/air-booking/v1/air-booking/page/air/booking/shopping" {
		res := tag.HookGetBody(resp.Params.RequestId)
		fmt.Println(res.Result.Body)
	}
}
