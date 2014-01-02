package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
)

func main() {
	//日志
	SetLogInfo()
	//读取配置文件
	if err := ReadConfig(); err != nil {
		log.Println(err)
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	Info("aa")
	//5秒钟
	timer := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-timer.C:

			for _, v := range Config.System.Cdn {
				Info(v, "查询余票")
				go Order(v)
			}

		}
	}
	Info("Trigger")
	Info("for")
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

//获取队列
func getQueueCount(v url.Values, values []string, cdn string) {

	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/getQueueCount", strings.NewReader(v.Encode()))
	Info(body)

	//confirmSingleForQueue
	urlValuesForQueue := url.Values{}
	for k, v := range Config.GetQueueCountRequest {
		urlValuesForQueue.Add(k, v)
	}
	urlValuesForQueue.Add("key_check_isChange", values[1])
	urlValuesForQueue.Add("leftTicketStr", values[2])
	confirmSingleForQueue(urlValuesForQueue, cdn)
}

//再次确认？
func confirmSingleForQueue(v url.Values, cdn string) {

	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/confirmSingleForQueue", strings.NewReader(v.Encode()))
	if strings.Contains(body, `"submitStatus":true`) {
		Info(body)
	} else {
		Warn(body)
	}

}

//提交订单
func submitOrderRequest(v url.Values, cdn string) {

	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/autoSubmitOrderRequest", strings.NewReader(v.Encode()))
	Debug("body:", body)
	if strings.Contains(body, `"submitStatus":true`) {
		orderResoult := new(OrderResoult)
		if err := json.Unmarshal([]byte(body), &orderResoult); err != nil {
			Error(err)
			return
		} else {
			v := strings.Split(orderResoult.Data.Result, "#")
			//key_check_isChange=99F79C00DFB9BF8713D23EFA4A8CF06BCA8C412DAC19686DCE306476
			// leftTicketStr = 1002353600401115003110023507803007450039
			// for getQueueCount
			Info("key_check_isChange:", v[1])
			Info("leftTicket:", v[2])
			urlValues := url.Values{}
			for k, v := range Config.GetQueueCountRequest {
				urlValues.Add(k, v)
			}
			urlValues.Add("leftTicket", v[2])
			getQueueCount(urlValues, v, cdn)
		}
	} else {
		Warn("提交订单:", body)
	}
}

//order
func Order(cdn string) {

	if tickets := queryLeftTicket(cdn); tickets != nil {
		for _, d := range tickets.Data {
			if d.Ticket.StationTrainCode == Config.Submit.TrainCode && d.Ticket.YingWoNum != "*" && d.Ticket.YingWoNum != "--" && d.Ticket.YingWoNum != "无" {
				Debug(d)
				Info("硬卧:", d.Ticket.YingWoNum, "软卧:", d.Ticket.RuanWoNum, "硬座:", d.Ticket.YingZuoNum)
				urlValues := url.Values{}
				for k, v := range Config.OrderRequest {
					urlValues.Add(k, v)
				}
				urlValues.Add("secretStr", d.SecretStr)
				submitOrderRequest(urlValues, cdn)
			} else {
				Info(d.Ticket.StationTrainCode, "硬卧:", d.Ticket.YingWoNum, "软卧:", d.Ticket.RuanWoNum, "硬座:", d.Ticket.YingZuoNum)
			}
		}
	} else {
		Error("余票查询错误")
	}

}

//查询余票
func queryLeftTicket(cdn string) *QueryLeftNewDTO {
	leftTicketUrl := "https://kyfw.12306.cn/otn/leftTicket/query?"
	for k, v := range Config.LeftTicket {
		leftTicketUrl += k + "=" + v + "&"
	}
	Debug("request url:", leftTicketUrl)
	body := DoForWardRequest(cdn, "GET", leftTicketUrl[:len(leftTicketUrl)-1], nil)
	Debug("body:", body)
	leftTicket := new(QueryLeftNewDTO)

	if err := json.Unmarshal([]byte(body), &leftTicket); err != nil {
		Error(err)
		return nil
	} else {
		// Debug(leftTicket.HttpStatus)
		// Debug(len(leftTicket.Data))
		// Debug(leftTicket.Data)
		return leftTicket
	}
}

//获取联系人
func getPassengerDTO() {
	body := DoForWardRequest(Config.System.Cdn[1], "POST", "https://kyfw.12306.cn/otn/confirmPassenger/getPassengerDTOs", nil)
	Debug("body:", body)
	passenger := new(PassengerDTO)

	if err := json.Unmarshal([]byte(body), &passenger); err != nil {
		Error(err)
	} else {
		// Debug(passenger.Data.NormalPassengers[0])
	}
}
