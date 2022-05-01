//编码操作
package big

import (
	"bytes"
	"encoding/base64"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
)

// 编码GBK到UTF8(GBK 字节集) 返回UTF8 字节集, 错误信息 error
func EnCodeGbkToUtf8(orig []byte) ([]byte, error) {
	I := bytes.NewReader(orig)
	O := transform.NewReader(I, simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(O)
	if e != nil {
		return nil, e
	}
	return d, nil
}

// 编码UTF8到GBK(UTF8 字节集) 返回GBK 字节集, 错误信息 error
func EnCodeUtf8ToGbk(orig []byte) ([]byte, error) {
	I := bytes.NewReader(orig)
	O := transform.NewReader(I, simplifiedchinese.GBK.NewEncoder())
	d, e := ioutil.ReadAll(O)
	if e != nil {
		return nil, e
	}
	return d, nil
}

// 编码BIG5到UTF8(BIG5 字节集) 返回UTF8 字节集, 错误信息 error
func EnCodeBig5ToUtf8(orig []byte) ([]byte, error) {
	I := bytes.NewReader(orig)
	O := transform.NewReader(I, traditionalchinese.Big5.NewDecoder())
	d, e := ioutil.ReadAll(O)
	if e != nil {
		return nil, e
	}
	return d, nil
}

// 编码UTF-8到BIG5(UTF-8 字节集) 返回BIG5 字节集, 错误信息 error
func EnCodeUtf8ToBig5(orig []byte) ([]byte, error) {
	I := bytes.NewReader(orig)
	O := transform.NewReader(I, traditionalchinese.Big5.NewEncoder())
	d, e := ioutil.ReadAll(O)
	if e != nil {
		return nil, e
	}
	return d, nil
}

// 将字符串进行URL编码 返回编码后的字符串
func EnCodeUrl(str string) string {
	return url.QueryEscape(str)
}

// 将字符串进行URL解码，返回解码后的字符串
func EnCodeUrlUn(str string) string {
	res, _ := url.QueryUnescape(str)
	return res
}

// 将字节集进行Base64编码，返回编码后的字节集
func EnCodeBase64(dst []byte) []byte {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(dst)))
	base64.StdEncoding.Encode(buf, dst)
	return buf
}

// 将Base64编码的字节集进行解码，返回解码后的字节集
func EnCodeBase64Un(s []byte) ([]byte, error) {
	dbuf := make([]byte, base64.StdEncoding.DecodedLen(len(s)))
	n, err := base64.StdEncoding.Decode(dbuf, []byte(s))
	return dbuf[:n], err
}

// 将字符串进行Base64编码，返回编码后的字符串
func EnCodeBase64Str(dst string) string {
	dstByte := []byte(dst)
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(dstByte)))
	base64.StdEncoding.Encode(buf, dstByte)
	return string(buf)
}

// 将Base64编码的字符串进行解码，返回解码后的字符串
func EnCodeBase64StrUn(s string) (string, error) {
	dbuf := make([]byte, base64.StdEncoding.DecodedLen(len(s)))
	n, err := base64.StdEncoding.Decode(dbuf, []byte(s))
	return string(dbuf[:n]), err
}

// 编码Ansi到Unicode(ansi的字符串) 返回Unicode字符串
func EnCodeAnsiToUnicode(ansi string) string {
	if ansi == "" {
		return ""
	}
	res := ""
	a := []rune(ansi)
	for _, v := range a {
		res += "&#" + strconv.Itoa(int(v))
	}
	return res
}

// 编码Unicode到Ansi(unicode的字符串) 返回Ansi字符串
func EnCodeUnicodeToAnsi(unicode string) string {
	if unicode == "" || !strings.Contains(unicode, "&#") {
		return ""
	}
	ansi := []rune("")
	uarr := strings.Split(unicode, "&#")
	for _, v := range uarr {
		if v != "" {
			a, _ := strconv.Atoi(v)
			ansi = append(ansi, rune(a))
		}
	}
	res := string(ansi)
	return res
}

// 编码Ansi转Usc2(ansi的字符串) 返回Usc2字符串
func EnCodeAnsiToUsc2(ansi string) string {
	textQuoted := strconv.QuoteToASCII(ansi)
	return textQuoted[1 : len(textQuoted)-1]
}

// 编码Usc2转Ansi(usc2的字符串) 返回Ansi字符串
func EnCodeUsc2ToAnsi(usc2 string) string {
	str, err := strconv.Unquote(strings.Replace(strconv.Quote(usc2), `\\u`, `\u`, -1))
	if err != nil {
		return ""
	}
	return str
}
