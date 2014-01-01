package main

import (
	"bufio"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func AddReqestHeader(request *http.Request) {
	request.Header.Set("Host", "kyfw.12306.cn")
	request.Header.Set("Connection", "keep-alive")
	request.Header.Set("Content-Length", "0")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Set("Origin", "https://kyfw.12306.cn")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.63 Safari/537.36")

	request.Header.Set("Referer", "https://kyfw.12306.cn/otn/leftTicket/init")
	request.Header.Set("Accept-Encoding", "gzip,deflate,sdch")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	request.Header.Set("Cookie", "JSESSIONID=5119B99B45C4C8BF0313851DD9991DEA; BIGipServerotn=2647916810.38945.0000; _jc_save_fromStation=%u4E4C%u9C81%u6728%u9F50%2CWMR; _jc_save_toStation=%u5E93%u5C14%u52D2%2CKLR; _jc_save_fromDate=2014-01-01; _jc_save_toDate=2014-01-01; _jc_save_wfdc_flag=dc")
}

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
func Check(err error) {
	if err != nil {
		Error(err)
	}
}
func ReadLines(path string) (lines []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		Error(err)
		return nil, err
	}
	buf := bufio.NewReader(file)
	for {
		//每次读取一行
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		} else {
			lines = append(lines, strings.TrimSpace(string(line)))
		}
	}
	return
}
