package main

import (
	"bufio"
	//"compress/gzip"
	//"crypto/tls"
	"fmt"
	//"io/ioutil"
	"net"
	"net/http"
	//"net/url"
	"runtime"
	"time"
)

func dial(netw, addr string) (net.Conn, error) {
	fmt.Printf("dial %v %v\n", netw, addr)
	return net.Dial(netw, addr)
}
func main() {

	SetLogger("file", `{"filename":"logs.log"}`)
	SetLevel(0)

	runtime.GOMAXPROCS(1)
	fmt.Println("1")
	timeout := 10 * time.Second
	c, err := net.DialTimeout("tcp", "124.254.47.187:80", timeout)
	if err != nil {
		fmt.Println(err)
		return
	}
	//buf_forward_conn *bufio.Reader
	buf_forward_conn := bufio.NewReader(c)

	//pr, err := url.Parse("113.57.187.29:443")
	//pr, err := url.Parse("http://127.0.0.1:8087")
	if err != nil {
		fmt.Println(err)
		return
	}
	//tr := &http.Transport{
	//	Proxy:           http.ProxyURL(pr),
	//	Dial:            dial,
	//	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//}
	fmt.Println("2")
	//http.DefaultTransport.(*http.Transport).Dial = dial
	//client := &http.Client{Transport: tr}
	fmt.Println("3")

	req, err := http.NewRequest("POST", "https://kyfw.12306.cn/otn/confirmPassenger/getPassengerDTOs", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	AddReqestHeader(req)
	var erra error
	erra = req.Write(c)
	if erra != nil {
		fmt.Println(erra)
		return
	}

	resp, err := http.ReadResponse(buf_forward_conn, req)

	//resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	//resp, err := client.Get("https://113.57.187.29/otn/")
	//resp, err := client.Get("https://golang.org/doc/")
	//resp, err := client.Get("https://74.125.128.141/doc/")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		html := ParseResponseBody(resp)
		fmt.Println("response body:", html)
		fmt.Println("dd")
	} else {
		fmt.Println("StatusCode:", resp.StatusCode)
		fmt.Println("ee")
	}
	fmt.Println("aaa")
}
