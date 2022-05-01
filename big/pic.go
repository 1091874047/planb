package big

import (
	"strings"
)

var (
	baiduExpiresIn   int64 = 0  //百度OCR识别AccessToken过期时间
	baiduAccessToken       = "" //百度OCR AccessToken值
)

/**
图片识别文字【有道接口】
传参：
	imgBase64：需要识别的图片，请传入Base64编码后的字符串
返回：
	识别结果，识别失败返回空字符串
*/
func PicOcrYouDao(imgBase64 string) string {
	data := strings.ReplaceAll(imgBase64, "+", "%2B")
	data = strings.ReplaceAll(data, "/", "%2F")
	data = strings.ReplaceAll(data, "=", "%3D")
	data = "imgBase=data%3Aimage%2Fjpeg%3Bbase64%2C" + data + "&lang=auto&company="
	res, _, _, _ := HttpSend(&HttpParms{
		Url:     "http://aidemo.youdao.com/ocrapi1",
		Mode:    "POST",
		DataStr: data,
		Headers: `Accept: */*
Accept-Encoding: gzip, deflate
Accept-Language: zh-CN,zh;q=0.9
Content-Type: application/x-www-form-urlencoded; charset=UTF-8
Proxy-Connection: keep-alive
Referer: http://aidemo.youdao.com/ocrdemo
User-Agent: Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36
X-Requested-With: XMLHttpRequest`,
	})
	return StrGetSub(res, "words\":\"", "\"}]}")
}

/**
图片识别文字【百度接口】
注册登录后打开创建APP：https://console.bce.baidu.com/ai/#/ai/ocr/overview/index
传参：
	apiKey：百度的API Key
	secretKey：百度的Secret Key
	imgBase64：需要识别的图片，请传入Base64编码后的字符串
返回：
	识别结果，识别失败返回空字符串
*/
func PicOcrBaidu(apiKey string, secretKey string, imgBase64 string) string {
	if _, nowtime := TimeStamp(10); nowtime-baiduExpiresIn > 2592000 {
		//判断accessToken是否过期
		res, _, _, _ := HttpSend(&HttpParms{Url: "https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=" + apiKey + "&client_secret=" + secretKey})
		baiduAccessToken = StrGetSub(res, "access_token\":\"", "\"")
		if baiduAccessToken == "" {
			//获取Token失败，返回空文本
			return ""
		}
		_, baiduExpiresIn = TimeStamp(10)
	}
	data := strings.ReplaceAll(imgBase64, "+", "%2B")
	data = strings.ReplaceAll(data, "/", "%2F")
	data = "image=" + strings.ReplaceAll(data, "=", "%3D")
	res, _, _, _ := HttpSend(&HttpParms{
		Url:     "https://aip.baidubce.com/rest/2.0/ocr/v1/general_basic?access_token=" + baiduAccessToken,
		Mode:    "POST",
		DataStr: data,
		Headers: "Content-Type: application/x-www-form-urlencoded",
	})
	return StrGetSub(res, "words\":\"", "\"}")
}
