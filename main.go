package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	passengerDTO       PassengerDTO
	passengerTicketStr string
	oldPassengerStr    string
	mainChannel        = make(chan int, 1) // 主线程
)

func main() {
	//日志
	SetLogInfo()
	//读取配置文件
	if err := ReadConfig(); err != nil {
		log.Println(err)
		return
	}
	if Config.System.OrderSize > 1 {
		mainChannel = make(chan int, Config.System.OrderSize) // 主线程
	}
	//runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	go getPassengerDTO(Config.System.Cdn[0])
	//见配置
	timer := time.NewTicker(time.Duration(Config.System.RefreshTime) * time.Millisecond)
	for {
		select {
		case <-timer.C:
			Info("查询余票")
			for _, v := range Config.System.Cdn {
				go Order(v)
			}

		}
	}
}
func ParsePassager() {
	if len(passengerDTO.Data.NormalPassengers) > 0 {
		for _, v := range passengerDTO.Data.NormalPassengers {
			for _, name := range Config.OrderInfo.PassengerName {
				if name == v.PassengerName {
					passengerTicketStr += Config.OrderInfo.SeatType + ",0,1," + name + "," + v.PassengerIdTypeCode + "," + v.PassengerIdNo + "," + v.Mobile + ",N_"
					oldPassengerStr += name + "," + v.PassengerIdTypeCode + "," + v.PassengerIdNo + ",1_"
				}
			}
		}
	}
	passengerTicketStr = passengerTicketStr[:len(passengerTicketStr)-1]
	Info(passengerTicketStr)
	Info(oldPassengerStr)
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
	AddReqestHeader(req, method)

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
	getPassCodeNew(cdn)

	params, _ := url.QueryUnescape(v.Encode())
	Info("getQueueCount Params", params)
	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/getQueueCount", strings.NewReader(params))
	Info("getQueueCount body:", body)

	//confirmSingleForQueue
	urlValuesForQueue := url.Values{}
	for k, v := range Config.ConfirmSingleForQueue {
		urlValuesForQueue.Add(k, v)
	}
	urlValuesForQueue.Add("key_check_isChange", values[1])
	urlValuesForQueue.Add("leftTicketStr", values[2])
	urlValuesForQueue.Add("passengerTicketStr", passengerTicketStr)
	urlValuesForQueue.Add("oldPassengerStr", oldPassengerStr)
	// time.Sleep(time.Second * 1)
	confirmSingleForQueue(urlValuesForQueue, cdn)
}

//再次确认？
func confirmSingleForQueue(v url.Values, cdn string) {
	Info("confirmSingleForQueue Params:", v)
	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/confirmSingleForQueue", strings.NewReader(v.Encode()))
	if strings.Contains(body, `"submitStatus":true`) {
		Info("confirmSingleForQueue body:", body)
	} else {
		Warn("confirmSingleForQueue body:", body)
	}

}

//提交订单
func submitOrderRequest(v url.Values, cdn string, t ticket) {
	defer func() {
		<-mainChannel
	}()
	mainChannel <- 1
	params, _ := url.QueryUnescape(v.Encode())
	Debug(params)

	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/autoSubmitOrderRequest", strings.NewReader(params))
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
			urlValues.Add("train_no", t.TrainNo)
			urlValues.Add("stationTrainCode", t.StationTrainCode)
			urlValues.Add("seatType", Config.OrderInfo.SeatType)
			urlValues.Add("fromStationTelecode", t.FromStationTelecode)
			urlValues.Add("toStationTelecode", t.ToStationTelecode)
			getQueueCount(urlValues, v, cdn)
		}
	} else if strings.Contains(body, `您还有未处理的订单`) {
		log.Println(body)
		log.Println("订票成功！！！！！")
		cmd := exec.Command(Config.System.Open, Config.System.OpenParams)
		cmd.Start()
		os.Exit(2)
	} else if strings.Contains(body, `用户未登录`) {
		log.Println(body)
		log.Println("用户未登录！！！！！")
		cmd := exec.Command(Config.System.Open, Config.System.OpenParams)
		cmd.Start()
		os.Exit(2)
	} else if strings.Contains(body, `取消次数过多`) {
		log.Println(body)
		log.Println("由于您取消次数过多！！！！！")
		cmd := exec.Command(Config.System.Open, Config.System.OpenParams)
		cmd.Start()
		os.Exit(1)
	} else {
		Warn(cdn, "订票请求警告:", body)
	}
}

//order
func queryJs(cdn string) {
	body := DoForWardRequest(cdn, "GET", "https://kyfw.12306.cn/otn/dynamicJs/queryJs", nil)
	Debug("body:", body)
}

//order
func getPassCodeNew(cdn string) {
	body := DoForWardRequest(cdn, "GET", "https://kyfw.12306.cn/otn/passcodeNew/getPassCodeNew.do?module=login&rand=sjrand&0.2866508506704122", nil)
	Debug("body:", body)
}

func Order(cdn string) {

	time.Sleep(time.Millisecond * time.Duration(rand.Int31n(2000)))

	queryJs(cdn)

	if tickets := queryLeftTicket(cdn); tickets != nil {
		for _, d := range tickets.Data {
			for _, trainCode := range Config.OrderInfo.TrainCode {
				if d.Ticket.StationTrainCode == trainCode && d.Ticket.YingWoNum != "*" && d.Ticket.YingWoNum != "--" && d.Ticket.YingWoNum != "无" {
					Debug(d)
					Info("开始订票:", d.Ticket.StationTrainCode, "硬卧:", d.Ticket.YingWoNum)
					urlValues := url.Values{}
					for k, v := range Config.OrderRequest {
						urlValues.Add(k, v)
					}
					urlValues.Add("secretStr", d.SecretStr)
					urlValues.Add("train_date", Config.OrderInfo.TrainDate)
					urlValues.Add("query_from_station_name", d.Ticket.FromStationName)
					urlValues.Add("query_to_station_name", d.Ticket.ToStationName)
					urlValues.Add("passengerTicketStr", passengerTicketStr)
					urlValues.Add("oldPassengerStr", oldPassengerStr)
					go submitOrderRequest(urlValues, cdn, d.Ticket)
				} else {
					Debug(d.Ticket.StationTrainCode, "硬卧:", d.Ticket.YingWoNum, "软卧:", d.Ticket.RuanWoNum, "硬座:", d.Ticket.YingZuoNum)
				}
			}
		}
	} else {
		Error(cdn, "余票查询错误", tickets)
	}

}

//查询余票
func queryLeftTicket(cdn string) *QueryLeftNewDTO {
	leftTicketUrl := "https://kyfw.12306.cn/otn/leftTicket/query?"

	leftTicketUrl += "leftTicketDTO.train_date=" + Config.OrderInfo.TrainDate + "&"
	leftTicketUrl += "leftTicketDTO.from_station=" + Config.OrderInfo.FromStation + "&"
	leftTicketUrl += "leftTicketDTO.to_station=" + Config.OrderInfo.ToStation + "&"
	leftTicketUrl += "purpose_codes=ADULT"

	Debug("request url:", leftTicketUrl)
	body := DoForWardRequest(cdn, "GET", leftTicketUrl, nil)
	Debug("body:", body)
	leftTicket := new(QueryLeftNewDTO)

	if err := json.Unmarshal([]byte(body), &leftTicket); err != nil {
		Error(cdn, err)
		Error(cdn, body)
		return nil
	} else {
		// Debug(leftTicket.HttpStatus)
		// Debug(len(leftTicket.Data))
		// Debug(leftTicket.Data)
		return leftTicket
	}
}

//获取联系人
func getPassengerDTO(cdn string) {
	Info("获取联系人")
	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/getPassengerDTOs", nil)
	Debug("body:", body)

	if err := json.Unmarshal([]byte(body), &passengerDTO); err != nil {
		Error(cdn, err)
		return
	} else {
		Debug(passengerDTO.Data.NormalPassengers[0])
		Info(cdn, "success!")
		ParsePassager()
	}
}
