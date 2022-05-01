package chrome

import (
	"b/big"
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"
	"strconv"
)

/**
调用浏览器方法
传参：
	method：方法名
	params：参数map类型，可用map传递多个参数
返回
	调用结果
*/
func (p *Tag) Call(method string, params map[string]interface{}) (string, error) {
	//判断标签是否连接
	if !p.connect {
		//未连接，则开始连接
		flag, err := p.Connect()
		if !flag {
			//连接失败
			return "", err
		}
	}
	p.taskLock.Lock()
	p.taskId++
	p.taskLock.Unlock()
	args := CallParm{
		Id:     p.taskId,
		Method: method,
		Params: params,
	}
	data, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	//发送消息
	err = p.ws.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return "", err
	}
	var message string
	if args.Method != "Page.enable" && args.Method != "Runtime.enable" && args.Method != "Network.enable" {
		//如果不是开启事件命令，则需要等待响应消息
		for {
			val, ok := p.taskRsp.Load(args.Id)
			if ok {
				message = val.(string)
				p.taskRsp.Delete(args.Id)
				break
			}
		}
	}
	return message, err
}

/**
执行表达式
传参：
	expression: 表达式文本
	objectGroup: 可用于释放多个对象的符号组名称。
	includeCommandLineAPI: 确定在评估期间Command Line API是否可用。
	silent: 在静默模式下，评估期间抛出的异常不报告，不要暂停执行。 覆盖setPauseOnException状态。
	executionContextId: 指定执行上下文执行评估。 如果省略参数，将在检查页面的上下文中执行评估。
	returnByValue: 结果是否应该是应该通过值发送的JSON对象。
	generatePreview: 是否应为结果生成预览。
	userGesture: 执行是否应该被视为用户在UI中发起的。
	awaitPromise: 执行是否应该等待承诺解决。 如果评估结果不是承诺，则认为是错误。
返回：
	EvalRes：EvalRes结构体用于保存执行结果，具体结果查看该结构体字段属性
	error：用于记录执行在Go中发生的错误
*/
func (p *Tag) Evaluate(expression string, objectGroup string, includeCommandLineAPI bool, silent bool,
	executionContextId int, returnByValue bool, generatePreview bool, userGesture bool, awaitPromise bool) (EvalRes, error) {
	parm := make(map[string]interface{})
	parm["expression"] = expression
	parm["objectGroup"] = objectGroup
	parm["includeCommandLineAPI"] = includeCommandLineAPI
	parm["silent"] = silent
	parm["contextId"] = executionContextId
	parm["returnByValue"] = returnByValue
	parm["generatePreview"] = generatePreview
	parm["userGesture"] = userGesture
	parm["awaitPromise"] = awaitPromise
	res, err := p.Call("Runtime.evaluate", parm)
	var evalres EvalRes
	if res == "" || err != nil {
		return evalres, err
	}
	jsonobj, err := simplejson.NewJson([]byte(res))
	if err != nil {
		return evalres, err
	}
	jsonobj = jsonobj.Get("result")
	errbyte, _ := jsonobj.Get("exceptionDetails").MarshalJSON()
	evalres.ExceptionDetails = string(errbyte)
	jsonobj = jsonobj.Get("result")
	evalres.ClassName, _ = jsonobj.Get("className").String()
	evalres.ResType, _ = jsonobj.Get("type").String()
	evalres.Subtype, _ = jsonobj.Get("subtype").String()
	evalres.Description, _ = jsonobj.Get("description").String()
	val, _ := jsonobj.Get("value").MarshalJSON()
	evalres.Value = string(val)
	if evalres.Value[:1] == "\"" && evalres.Value[len(evalres.Value)-1:] == "\"" {
		evalres.Value, _ = strconv.Unquote(evalres.Value)
	}
	return evalres, nil
}

/**
按键或文本输入事件
传参：
	typ：按键类型,可选值: keyDown, keyUp, rawKeyDown, char
	modifiers：功能键，可选值：Alt = 1, Ctrl = 2,Meta/Command = 4,Shift = 8,默认为 = 0
	windowsVirtualKeyCode：Windows虚拟键代码（默认值：0）
	text：欲输入的文本，默认为空字符串。
	isKeypad：事件是否从小键盘生成，true为是，false为否，一般为false
	isSysttemKey：事件是否是系统键事件，true为是，false为否，一般为false
返回：
	成功返回true，否则返回false
*/
func (p *Tag) keyEvent(typ string, modifiers int, windowsVirtualKeyCode int, text string, isKeypad bool, isSysttemKey bool) bool {
	parm := make(map[string]interface{})
	parm["type"] = typ
	parm["modifiers"] = modifiers
	_, timestamp := big.TimeStamp(19)
	parm["timestamp"] = timestamp
	parm["text"] = text                                   //通过使用键盘布局处理虚拟键代码生成的文本。 keyUp和rawKeyDown事件不需要（默认：“”）
	parm["unmodifiedText"] = ""                           //如果没有修改器被按下，则由键盘生成的文本（移位除外）。 用于快捷键（加速器）键处理（默认：“”）。
	parm["keyIdentifier"] = ""                            //唯一键标识符（例如，“U + 0041”）（默认值：“”）。
	parm["code"] = ""                                     //每个物理键的唯一DOM定义的字符串值（例如，“KeyA”）（默认：“”）。
	parm["key"] = ""                                      //唯一的DOM定义的字符串值，描述了活动修饰符，键盘布局等上下文中键的含义（例如，“AltGr”）（默认值：“”）。
	parm["windowsVirtualKeyCode"] = windowsVirtualKeyCode //Windows虚拟键代码（默认值：0）。
	parm["nativeVirtualKeyCode"] = 0                      //本地虚拟键代码（默认值：0）。
	parm["autoRepeat"] = false                            //事件是否由自动重复生成（默认值：false）。
	parm["isKeypad"] = isKeypad                           //事件是否从小键盘生成（默认值：false）。
	parm["isSystemKey"] = isSysttemKey                    //事件是否是系统键事件（默认值：false）。
	res, err := p.Call("Input.dispatchKeyEvent", parm)
	if res == "" || err != nil {
		return false
	}
	return true
}

/**
监听Chrome返回的消息
*/
func (p *Tag) onListenerMsg() {
	//等待回复消息
	for {
		_, message, err := p.ws.ReadMessage()
		if err != nil || p.connect == false {
			//连接已彻底中断
			p.connect = false
			return
		}
		if len(message) > 0 {
			jsonobj, err := simplejson.NewJson(message)
			if err == nil {
				_, flag := jsonobj.CheckGet("id")
				if flag {
					//如果id字段存在，说明是普通操作的响应结果
					id, err := jsonobj.Get("id").Int()
					if err == nil && id > 0 {
						p.taskRsp.Store(id, string(message))
					}
				} else {
					//如果id字段不存在，说明是事件的响应结果
					method, _ := jsonobj.Get("method").String()
					switch method {
					case "Network.requestWillBeSent":
						//请求拦截
						if p.hookReqEvent != nil {
							req := HookHttpRequest{}
							json.Unmarshal(message, &req)
							go p.hookReqEvent(p, req)
						}
					case "Network.responseReceived":
						if p.hookRespEvent != nil {
							resp := HookHttpResponse{}
							json.Unmarshal(message, &resp)
							go p.hookRespEvent(p, resp)
						}
					case "Runtime.executionContextCreated":
						//V8引擎创建完毕事件，更新标签上下文ID
						key, _ := jsonobj.Get("params").Get("context").Get("auxData").Get("frameId").String()
						value, _ := jsonobj.Get("params").Get("context").Get("id").Int()
						p.contextIds.Store(key, value)
					case "Runtime.executionContextDestroyed":
						//V8引擎销毁事件，删除上下文ID
						id, _ := jsonobj.Get("params").Get("executionContextId").Int()
						p.contextIds.Range(func(key, value interface{}) bool {
							if value == id {
								p.contextIds.Delete(key)
								return false
							}
							return true
						})
					case "Runtime.executionContextsCleared":
						p.contextIds.Range(func(key, value interface{}) bool {
							p.contextIds.Delete(key)
							return true
						})
					case "Page.loadEventFired":
						p.readyState.loadEventFired = true
						p.readyState.frameStoppedLoading = false
						//页面加载完成时间，格式：
						//{
						//	"message": {
						//	"method": "Page.loadEventFired",
						//		"params": {
						//		"timestamp": 42133.108263
						//	}
						//},
						//	"webview": "28DAFE9FE90E9292F1B8EDB3315608EC"
						//}
					case "Page.frameStartedLoading":
						p.readyState.frameStartedLoading = true
						p.readyState.frameStoppedLoading = false
						//框架开始加载，格式：
						//{"message":{"method":"Page.frameStartedLoading","params":{"frameId":"49C70573CE1145CEB5B38A270213A48"}},"webview":"28DAFE9FE90E9292F1B8EDB3315608EC"}
					case "Page.frameStoppedLoading":
						p.readyState.frameStoppedLoading = true
						p.pageInfoUpdate()
						//框架停止加载
					case "Page.javascriptDialogOpening":
						p.haveDialog = true
						if p.dialogOpenEvent != nil {
							jbyte, _ := jsonobj.Get("params").MarshalJSON()
							var dl DialogOpen
							json.Unmarshal(jbyte, &dl)
							p.dialogOpenEvent(dl)
						}
						//页面弹窗触发事件
						//{"method":"Page.javascriptDialogOpening","params": {"url":"http://xss.php","message":"1","type":"alert","hasBrowserHandler":false,"defaultPrompt":""} }
					case "Page.javascriptDialogClosed":
						//页面弹窗关闭事件
						if p.dialogCloseEvent != nil {
							jbyte, _ := jsonobj.Get("params").MarshalJSON()
							var dl DialogClose
							json.Unmarshal(jbyte, &dl)
							p.dialogCloseEvent(dl)
						}
						p.haveDialog = false
					case "Runtime.consoleAPICalled":
						//拦截控制台消息输出事件
						p.logsLock.Lock()
						scriptUrl, _ := jsonobj.Get("params").Get("stackTrace").Get("callFrames").GetIndex(0).Get("url").String()
						args, _ := jsonobj.Get("params").Get("args").Array()
						for i, _ := range args {
							jbyte, _ := jsonobj.Get("params").Get("args").GetIndex(i).MarshalJSON()
							var log ConsoleLog
							json.Unmarshal(jbyte, &log)
							log.ScriptUrl = scriptUrl
							p.logs = append(p.logs, log)
						}
						p.logsLock.Unlock()
					}
				}

			}
		}
	}
}

/**
更新url和title属性和浏览器的x、y值
*/
func (p *Tag) pageInfoUpdate() {
	p.px = 0
	p.py = 0
	_, res, _, _ := big.HttpSend(&big.HttpParms{Url: "http://" + big.StrGetSub(p.WebSocketDebuggerUrl, "ws://", "/") + "/json"})
	tags := make([]Tag, 0)
	err := json.Unmarshal(res, &tags)
	if err != nil {
		return
	}
	for _, v := range tags {
		if v.Id == p.Id {
			p.FaviconUrl = v.FaviconUrl
			p.Title = v.Title
			p.Url = v.Url
			return
		}
	}
}
