package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	mainChan = make(chan int, 20) //主线程
	wg       = sync.WaitGroup{}   // 用于等待所有 goroutine 结束
)

func main() {
	//日志
	SetLogInfo()
	//读取配置文件
	if err := ReadConfig(); err != nil {
		Error(err)
		return
	}

	//"https://kyfw.12306.cn/otn/confirmPassenger/getPassengerDTOs"
	runtime.GOMAXPROCS(1)
	wg.Add(1)
	wg.Add(1)
	go getPassengerDTO()
	go queryLeftTicket()
	Info("waiting!")
	wg.Wait()
	log.Println("finished!")
}

//转发
func DoForWardRequest(forwardAddress, method, requestUrl string, body io.Reader) string {
	if !strings.Contains(forwardAddress, ":") {
		forwardAddress = forwardAddress + ":80"
	}

	timeout := 10 * time.Second

	conn, err := net.DialTimeout("tcp", forwardAddress, timeout)
	if err != nil {
		Error(err)
		return ""
	}
	//buf_forward_conn *bufio.Reader
	buf_forward_conn := bufio.NewReader(conn)

	req, err := http.NewRequest(method, requestUrl, body)
	if err != nil {
		Error(err)
		return ""
	}
	//add header
	AddReqestHeader(req)

	var errWrite error

	errWrite = req.Write(conn)
	if errWrite != nil {
		Error(errWrite)
		return ""
	}
	defer conn.Close()
	resp, err := http.ReadResponse(buf_forward_conn, req)

	if err != nil {
		Error(err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body := ParseResponseBody(resp)
		Debug("response body:", body)
		return body
	} else {
		Error("StatusCode:", resp.StatusCode)
	}
	return ""
}

//获取联系人
func getPassengerDTO() {
	defer func() {
		<-mainChan
		wg.Done()
	}()
	mainChan <- 1

	body := DoForWardRequest(Config.System.Cdn[1], "POST", "https://kyfw.12306.cn/otn/confirmPassenger/getPassengerDTOs", nil)
	Debug("body:", body)
	passenger := new(PassengerDTO)

	if err := json.Unmarshal([]byte(body), &passenger); err != nil {
		Error(err)
	} else {
		Debug(passenger.Data.NoLogin)
		Debug(passenger.Data.NormalPassengers[0].PassengerName)
	}
}

func queryLeftTicket() {
	defer func() {
		<-mainChan
		wg.Done()
	}()
	mainChan <- 1
	leftTicketUrl := "https://kyfw.12306.cn/otn/leftTicket/query?"
	for k, v := range Config.LeftTicket {
		leftTicketUrl += k + "=" + v + "&"
	}
	Debug("request url:", leftTicketUrl)
	body := DoForWardRequest(Config.System.Cdn[2], "GET", leftTicketUrl[:len(leftTicketUrl)-1], nil)
	Debug("body:", body)
	leftTicket := new(QueryLeftNewDTO)

	if err := json.Unmarshal([]byte(body), &leftTicket); err != nil {
		Error(err)
	} else {
		Debug(leftTicket.Data[0].ticket.ArriveTime)
		Debug(leftTicket.Data[0].secretStr)
	}
}
