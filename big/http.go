//网络请求操作
package big

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//Http请求结构体参数
type HttpParms struct {
	Url             string      //请求地址
	Mode            string      //提交方式：GET POST HEAD PUT OPTIONS DELETE TRACE CONNECT，为空默认为GET
	DataStr         string      //提交字符串数据，POST方式本参数有效，Data与DataByte参数二选一传入即可。
	DataByte        []byte      //提交字节集数据，POST方式本参数有效，Data与DataByte参数二选一传入即可。
	Cookies         string      //附加Cookies，把浏览器中开发者工具中Cookies复制传入即可
	Headers         string      //附加协议头，直接将浏览器抓包的协议头复制下来传入即可，无需调整格式，User-Agent也是在此处传入，如果为空默认为Chrome的UA。
	RetHeaders      http.Header //返回协议头，http.Header类型，需导入"net/http"包，返回协议头的参数通过本变量.Get(参数名 string)获取
	RetStatusCode   int         //返回状态码
	Redirect        bool        //是否禁止重定向，true为禁止重定向
	ProxyIP         string      //代理IP，格式IP:端口，如：127.0.0.1:8888
	ProxyUser       string      //代理IP账户
	ProxyPwd        string      //代理IP密码
	TimeOut         int         //超时时间，单位：秒，默认30秒，如果提供大于0的数值，则修改操作超时时间
	AutoFormatEnter bool        //是否将提交的数据内容的换行强制转为\r\n格式，当提交有换行数据有问题时，将此项设为true
}

/**
发送Http请求
传参：
	hp：传递HttpParms对象指针，HttpParms对象属性字段用于填写请求参数
返回：
	resStr：响应文本结果
	resByte：响应字节集结果
	cookies：提交时的cookies和服务响应cookies合并后的最新cookies
	err：错误信息
*/
func HttpSend(hp *HttpParms) (resStr string, resByte []byte, cookies string, err error) {
	//设置超时时间
	client := &http.Client{}
	if hp.TimeOut > 0 {
		client.Timeout = time.Duration(hp.TimeOut) * time.Second
	}
	//if hp.TimeOut == 0 {
	//	hp.TimeOut = 30
	//}
	//client := &http.Client{Timeout: time.Duration(hp.TimeOut) * time.Second}
	//判断是否重定向
	if hp.Redirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	//判断是否有代理IP
	if hp.ProxyIP != "" {
		proxyAddr := ""
		if hp.ProxyUser == "" {
			proxyAddr = "http://" + hp.ProxyIP + "/"
		} else {
			proxyAddr = "http://" + hp.ProxyUser + ":" + hp.ProxyPwd + "@" + hp.ProxyIP + "/"
		}
		proxy, err := url.Parse(proxyAddr)
		if err != nil {
			log.Fatal(err)
		}
		netTransport := &http.Transport{
			Proxy: http.ProxyURL(proxy),
			//MaxIdleConnsPerHost:   -1,
			//ResponseHeaderTimeout: time.Duration(hp.TimeOut) * time.Second,
		}
		//client.Timeout = time.Duration(hp.TimeOut) * time.Second
		client.Transport = netTransport
	}
	if hp.Mode == "" {
		hp.Mode = "GET"
	}
	var req *http.Request
	if hp.Mode == "POST" || hp.Mode == "PUT" || hp.Mode == "OPTIONS" || hp.Mode == "DELETE" {
		if hp.DataStr == "" {
			if hp.AutoFormatEnter {
				hp.DataByte = bytes.ReplaceAll(hp.DataByte, []byte("\r\n"), []byte("\n"))
				hp.DataByte = bytes.ReplaceAll(hp.DataByte, []byte("\n"), []byte("\r\n"))
			}
			req, err = http.NewRequest(hp.Mode, hp.Url, bytes.NewReader(hp.DataByte))
			req.Header.Set("Content-Length", strconv.Itoa(len(hp.DataByte)))
		} else {
			if hp.AutoFormatEnter {
				hp.DataStr = strings.ReplaceAll(hp.DataStr, "\r\n", "\n")
				hp.DataStr = strings.ReplaceAll(hp.DataStr, "\n", "\r\n")
			}
			req, err = http.NewRequest(hp.Mode, hp.Url, strings.NewReader(hp.DataStr))
			req.Header.Set("Content-Length", strconv.Itoa(len(hp.DataStr)))
		}
	} else {
		req, err = http.NewRequest(hp.Mode, hp.Url, nil)
	}
	if err != nil {
		log.Println(err)
		return
	}
	//添加headers
	if strings.Index(hp.Headers, "User-Agent") == -1 && strings.Index(hp.Headers, "user-agent") == -1 {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.3")
	}
	hp.Headers = strings.ReplaceAll(hp.Headers, "\r\n", "\n")
	hp.Headers = strings.ReplaceAll(hp.Headers, "\n", "\r\n")
	strSplit := strings.Split(hp.Headers, "\r\n")
	for _, val := range strSplit {
		val = strings.Replace(strings.Replace(val, ": ", ":", 1), "\t", "", -1)
		if val != "" {
			req.Header.Set(StrGetLeft(val, ":"), StrGetRight(val, ":"))
		}
	}
	//添加Cookies
	if hp.Cookies != "" {
		req.Header.Set("Cookie", hp.Cookies)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	//合并Cookies
	hp.Cookies = HttpMergeCookies(hp.Cookies, HttpCookiesToStr(resp.Cookies()))
	resByte, err = ioutil.ReadAll(resp.Body)
	if err != nil && err.Error() != "gzip: invalid header" {
		log.Println(err)
		return
	}
	err = nil
	//判断是否需要Gzip解压
	if strings.Index(resp.Header.Get("Content-Encoding"), "gzip") != -1 {
		resByte = HttpGzipUn(resByte)
	}
	hp.RetHeaders = resp.Header
	hp.RetStatusCode = resp.StatusCode
	//判断是否需要转码，Golang默认UTF8编码，如果网站采用GBK则需要转换为UTF8后Golang才能识别
	resStr = string(resByte)
	if strings.Index(resp.Header.Get("Content-Type"), "charset=gb") != -1 || strings.Index(resStr, "charset=\"gb") != -1 || strings.Index(resStr, "charset=gb") != -1 || strings.Index(resp.Header.Get("Content-Type"), "charset=GB") != -1 || strings.Index(resStr, "charset=\"GB") != -1 || strings.Index(resStr, "charset=GB") != -1 {
		resByte, _ = EnCodeGbkToUtf8(resByte)
		resStr = string(resByte)
	}
	cookies = hp.Cookies
	return
}

//将http的[]Cookie类型转为Cookies字符串
func HttpCookiesToStr(cookies []*http.Cookie) string {
	res := ""
	for _, v := range cookies {
		res = res + v.Name + "=" + v.Value + "; "
	}
	if res != "" {
		res = res[0 : len(res)-2]
	}
	return res
}

//合并文本Cookies，返回合并后的文本Cookies
func HttpMergeCookies(oldCookies string, newCookies string) string {
	//初步格式化
	oldCookies = strings.TrimSpace(oldCookies)
	if oldCookies != "" && oldCookies[len(oldCookies)-1:len(oldCookies)] == ";" {
		oldCookies = oldCookies + " "
	}

	newCookies = strings.TrimSpace(newCookies)
	if newCookies != "" && newCookies[len(newCookies)-1:len(newCookies)] == ";" {
		newCookies = newCookies + " "
	}
	if newCookies == "" {
		return oldCookies
	}
	//开始合并Cookies
	oldArray := strings.Split(oldCookies, "; ")
	for _, val := range oldArray {
		if strings.Index(newCookies, StrGetLeft(val, "=")+"=") == -1 {
			newCookies = newCookies + "; " + val
		}
	}
	return strings.ReplaceAll(newCookies, "; ; ", "; ")
}

//Gzip压缩：传入准备压缩的数据，返回压缩后的数据
func HttpGzipPack(data []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	defer w.Close()
	w.Write(data)
	w.Flush()
	return b.Bytes()
}

//Gzip解压，传入准备解压的数据，返回解压后的数据
func HttpGzipUn(data []byte) []byte {
	var b bytes.Buffer
	b.Write(data)
	r, _ := gzip.NewReader(&b)
	defer r.Close()
	unRes, _ := ioutil.ReadAll(r)
	return unRes
}

/**
获取单个Cookie值
传参：
	cookies：全部Cookies字符串
	name：欲获取的Cookie名称
返回：
	cookie的值，若cookie不存在则返回空文本
*/
func HttpGetCookie(cookies string, name string) string {
	cookie := StrGetSub(cookies, name+"=", ";")
	if cookie == "" {
		cookie = StrGetRight(cookies, name+"=")
	}
	return cookie
}
