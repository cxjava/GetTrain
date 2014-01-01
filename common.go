package main

import (
	//"bufio"
	"compress/gzip"
	//"io"
	"io/ioutil"
	"net/http"
	//"os"
	//"strings"

	"github.com/BurntSushi/toml"
)

var Config config

//添加头
func AddReqestHeader(request *http.Request) {
	request.Header.Set("Host", "kyfw.12306.cn")
	request.Header.Set("Connection", "keep-alive")
	request.Header.Set("Content-Length", "0")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Set("Origin", "https://kyfw.12306.cn")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("User-Agent", Config.Login.UserAgent)

	request.Header.Set("Referer", "https://kyfw.12306.cn/otn/leftTicket/init")
	request.Header.Set("Accept-Encoding", "gzip,deflate,sdch")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	request.Header.Set("Cookie", Config.Login.Cookie)
}

//读取响应
func ParseResponseBody(resp *http.Response) string {
	var body []byte
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			Error(err)
			return ""
		}
		defer reader.Close()
		bodyByte, err := ioutil.ReadAll(reader)
		if err != nil {
			Error(err)
			return ""
		}
		body = bodyByte
	default:
		bodyByte, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Error(err)
			return ""
		}
		body = bodyByte
	}
	return string(body)
}

//读取配置文件
func ReadConfig() error {
	if _, err := toml.DecodeFile("config.toml", &Config); err != nil {
		Error(err)
		return err
	}
	Debug(Config.System.Cdn)
	for k, v := range Config.LeftTicket {
		Debug(k, " = ", v)
	}
	return nil
}

//设置log相关
func SetLogInfo() {
	// debug 1, info 2
	SetLevel(1)
	SetLogger("console", "")
	SetLogger("file", `{"filename":"log.log"}`)
}
