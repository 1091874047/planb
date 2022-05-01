package chrome

//标签上下文
type TagContext struct {
	Id      int    `json:"id"`
	Origin  string `json:"origin"`
	Name    string `json:"name"`
	FrameId string `json:"frameId"`
}

//调用浏览器方法的参数
type CallParm struct {
	Id     int                    `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

//页面框架
type frame struct {
	Id                             string     `json:"id"`                //当前框架ID
	ParentId                       string     `json:"parentId"`          //父框架ID，顶级框架此项空字符串
	IoaderId                       string     `json:"loaderId"`          //加载ID
	Name                           string     `json:"name"`              //框架名：顶级框架此项空字符串
	Url                            string     `json:"url"`               //页面地址
	DomainAndRegistry              string     `json:"domainAndRegistry"` //域名
	SecurityOrigin                 string     `json:"securityOrigin"`
	MimeType                       string     `json:"mimeType"` //MIME类型：诸如application/javascript、image/jpeg、text/html等
	AdFrameType                    string     `json:"adFrameType"`
	SecureContextType              string     `json:"secureContextType"`
	CrossOriginIsolatedContextType string     `json:"crossOriginIsolatedContextType"`
	ContextId                      int        `json:"contextId"` //标签上下文ID
	Resources                      []resource `json:"resources"` //框架页面资源，比如图片、音频、视频、JS、CSS、字体等文件
}

//框架资源
type resource struct {
	Url         string `json:"url"`         //资源地址，图片资源地址有些是base64编码后的data:image/webp;base64,xxxx格式
	Typ         string `json:"type"`        //资源类型：Script、Stylesheet、Image
	MimeType    string `json:"mimeType"`    //MIME类型：诸如application/javascript、image/jpeg、text/html等
	ContentSize int    `json:"contentSize"` //URL所指向的内容文件大小
}

//执行JS表达式结果
type EvalRes struct {
	ClassName        string `json:"className"`        //类名
	ResType          string `json:"type"`             //结果主类型，目前已知类型有：undefined=未定义，number=整数值，string=字符串，double=浮点数，boolean=布偶
	Subtype          string `json:"subtype"`          //子类型
	Value            string `json:"value"`            //执行结果值
	Description      string `json:"description"`      //结果描述，一般就是将其他类型强制转为string保存到本属性
	ExceptionDetails string `json:"exceptionDetails"` //异常信息，当执行出现错误后本属性有值
}

//框架页面加载进度
type frameLoadStep struct {
	loadEventFired      bool //页面开始加载
	frameStartedLoading bool //框架开始加载
	frameStoppedLoading bool //框架停止加载
}

//对话框创建时的信息
type DialogOpen struct {
	Url           string `json:"url"`           //发生的网页地址
	Msg           string `json:"message"`       //消息内容
	Typ           string `json:"type"`          //弹窗类型：alert,confirm,prompt,beforeunload
	DefaultPrompt string `json:"defaultPrompt"` //默认文本
}

//对话框关闭时的信息
type DialogClose struct {
	Result    bool   `json:"result"`    //结果，具体含义不知
	UserInput string `json:"userInput"` //对话框输入的信息
}

//控制台输出日志信息
type ConsoleLog struct {
	Typ       string      `json:"type"`  //输出数据类型，目前已知类型有：undefined=未定义，number=整数值，string=字符串，double=浮点数，boolean=布偶
	Value     interface{} `json:"value"` //日志内容，需转换为真实类型，转换代码：.(真实类型)，比如Typ为number，那么Go对应的为int类型，转换代码为：Value.(int)
	ScriptUrl string      `json:"url"`   //触发该日志输出的JS代码所在的文件路径
}

//Cookie信息
type Cookie struct {
	Name     string  `json:"name"`     //Cookie名称
	Value    string  `json:"value"`    //Cookie值
	Domain   string  `json:"domain"`   //所指域名
	Path     string  `json:"path"`     //所指路径
	Expires  float64 `json:"expires"`  //过期时间，单位毫秒
	Size     int     `json:"size"`     //名称+值拼接的大小
	HTTPOnly bool    `json:"httpOnly"` //是否禁止通过JS获取该Cookie，true为禁止，false为允许
	Secure   bool    `json:"secure"`   //是否为https协议，true为只能用于https协议
	Session  bool    `json:"session"`  //是否为会话状态保持的Cookie，true表示该Cookie用于会话保持
	Priority string  `json:"priority"` //优先级，chrome的提案，定义了三种优先级，Low/Medium/High，当cookie数量超出时，低优先级的cookie会被优先清除
	Url      string  `json:"url"`      //Domain+Path的字符串拼接
	SameSite string  `json:"sameSite,omitempty"`
	//SameSite为限制第三方cookie，有3个值：Strict/Lax/None
	//Strict: 仅允许发送同站点请求的的cookie
	//Lax: 允许部分第三方请求携带cookie，即导航到目标网址的get请求。包括超链接<a href='...' />，预加载<link rel="prerender" />和get表单<form method="GET" />三种形式发送cookie
	//None: 任意发送cookie，设置为None，需要同时设置Secure，意味着网站必须采用https，若同时支持http和https，可以将http用307跳转到https
}

//触摸点结构体
type Touch struct {
	X             float64 `json:"x"`             //X坐标，相当于CSS像素，相对于浏览器视口左上角
	Y             float64 `json:"y"`             //Y坐标，相当于CSS像素，相对于浏览器视口左上角，0表示视口的顶部，Y随着进入视口底部而增加
	RadiusX       float64 `json:"radiusX"`       //触摸区域的X半径（默认值：1.0）
	RadiusY       float64 `json:"radiusY"`       //触摸区域的Y半径（默认值：1.0）
	RotationAngle float64 `json:"rotationAngle"` //旋转角度（默认值：0.0）
	Force         float64 `json:"force"`         //强制（默认值：1.0）
	Id            float64 `json:"id"`            //用于跟踪事件之间的触摸源的标识符在事件中必须是唯一的
}

//拦截的请求POST数据主体
type HookHttpBody struct {
	Id     int `json:"id"`
	Result struct {
		Body          string `json:"body"`          //POST数据主体内容
		Base64Encoded bool   `json:"base64Encoded"` //主体内容是否Base64编码
	} `json:"result"`
}

//拦截请求的内容
type HookHttpRequest struct {
	Method string `json:"method"` //固定内容：Network.requestWillBeSent
	Params struct {
		RequestId   string   `json:"requestId"`   //请求ID（和响应ID配对一致）
		LoaderId    string   `json:"loaderId"`    //加载页框架ID
		DocumentURL string   `json:"documentURL"` //发起请求所在的网页路径
		Request     struct { //请求主体内容
			Url             string            `json:"url"`         //请求URL
			Method          string            `json:"method"`      //请求模式：GET POST
			Headers         map[string]string `json:"headers"`     //Head信息
			PostData        string            `json:"postData"`    //POST数据内容
			HasPostData     bool              `json:"hasPostData"` //是否有POST数据
			PostDataEntries []struct {
				Bytes string `json:"bytes"` //POST数据Base64编码后的内容
			} `json:"postDataEntries"`
			MixedContentType string `json:"mixedContentType"`
			InitialPriority  string `json:"initialPriority"`
			ReferrerPolicy   string `json:"referrerPolicy"`
		} `json:"request"`
		Timestamp float64  `json:"timestamp"`
		WallTime  float64  `json:"wallTime"`
		Initiator struct { //发起请求源
			Type  string   `json:"type"` //源类型
			Stack struct { //堆栈信息
				CallFrames []struct {
					FunctionName string `json:"functionName"`
					ScriptId     string `json:"scriptId"`
					Url          string `json:"url"`
					LineNumber   int    `json:"lineNumber"`
					ColumnNumber int    `json:"columnNumber"`
				} `json:"callFrames"`
				Parent struct {
					Description string `json:"description"`
					CallFrames  []struct {
						FunctionName string `json:"functionName"`
						ScriptId     string `json:"scriptId"`
						Url          string `json:"url"`
						LineNumber   int    `json:"lineNumber"`
						ColumnNumber int    `json:"columnNumber"`
					} `json:"callFrames"`
				} `json:"parent"`
			} `json:"stack"`
		} `json:"initiator"`
		Type           string `json:"type"`    //请求类型：XHR Script Document等
		FrameId        string `json:"frameId"` //页面框架ID
		HasUserGesture bool   `json:"hasUserGesture"`
	} `json:"params"`
}

//拦截响应的内容
type HookHttpResponse struct {
	Method string `json:"method"` //固定内容：Network.responseReceived
	Params struct {
		RequestId string   `json:"requestId"` //响应ID（和请求ID配对一致）
		LoaderId  string   `json:"loaderId"`  //加载页框架ID
		Timestamp float64  `json:"timestamp"`
		Type      string   `json:"type"` //请求类型：XHR Script Document等
		Response  struct { //响应主体
			Url               string            `json:"url"`    //请求URL
			Status            int               `json:"status"` //响应状态码
			StatusText        string            `json:"statusText"`
			Headers           map[string]string `json:"headers"`  //响应的头部信息
			MimeType          string            `json:"mimeType"` //文档类型：比如application/json
			ConnectionReused  bool              `json:"connectionReused"`
			ConnectionId      int               `json:"connectionId"`
			RemoteIPAddress   string            `json:"remoteIPAddress"` //远程IP地址
			RemotePort        int               `json:"remotePort"`      //远程IP地址端口
			FromDiskCache     bool              `json:"fromDiskCache"`
			FromServiceWorker bool              `json:"fromServiceWorker"`
			FromPrefetchCache bool              `json:"fromPrefetchCache"` //是否从缓存读取
			EncodedDataLength int               `json:"encodedDataLength"` //返回数据长度
			Timing            struct {
				RequestTime              float64 `json:"requestTime"`
				ProxyStart               int     `json:"proxyStart"`
				ProxyEnd                 int     `json:"proxyEnd"`
				DnsStart                 int     `json:"dnsStart"`
				DnsEnd                   int     `json:"dnsEnd"`
				ConnectStart             int     `json:"connectStart"`
				ConnectEnd               int     `json:"connectEnd"`
				SslStart                 int     `json:"sslStart"`
				SslEnd                   int     `json:"sslEnd"`
				WorkerStart              int     `json:"workerStart"`
				WorkerReady              int     `json:"workerReady"`
				WorkerFetchStart         int     `json:"workerFetchStart"`
				WorkerRespondWithSettled int     `json:"workerRespondWithSettled"`
				SendStart                float64 `json:"sendStart"`
				SendEnd                  float64 `json:"sendEnd"`
				PushStart                int     `json:"pushStart"`
				PushEnd                  int     `json:"pushEnd"`
				ReceiveHeadersEnd        float64 `json:"receiveHeadersEnd"`
			} `json:"timing"`
			ResponseTime    float64 `json:"responseTime"`
			Protocol        string  `json:"protocol"`
			SecurityState   string  `json:"securityState"`
			SecurityDetails struct {
				Protocol                       string   `json:"protocol"`
				KeyExchange                    string   `json:"keyExchange"`
				KeyExchangeGroup               string   `json:"keyExchangeGroup"`
				Cipher                         string   `json:"cipher"`
				CertificateId                  int      `json:"certificateId"`
				SubjectName                    string   `json:"subjectName"`
				SanList                        []string `json:"sanList"`
				Issuer                         string   `json:"issuer"`
				ValidFrom                      int      `json:"validFrom"`
				ValidTo                        int      `json:"validTo"`
				SignedCertificateTimestampList []struct {
					Status             string `json:"status"`
					Origin             string `json:"origin"`
					LogDescription     string `json:"logDescription"`
					LogId              string `json:"logId"`
					Timestamp          int64  `json:"timestamp"`
					HashAlgorithm      string `json:"hashAlgorithm"`
					SignatureAlgorithm string `json:"signatureAlgorithm"`
					SignatureData      string `json:"signatureData"`
				} `json:"signedCertificateTimestampList"`
				CertificateTransparencyCompliance string `json:"certificateTransparencyCompliance"`
			} `json:"securityDetails"`
		} `json:"response"`
		FrameId string `json:"frameId"` //页面框架ID
	} `json:"params"`
}
