package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	submitChannel      = make(chan int, 1)       // 提交池
	queryChannel       = make(chan int, 10)      // 查询池
	availableCDN       = make(map[string]string) //可用cdn
	timeout            = 10 * time.Second        //超时时间
)

func main() {
	//日志
	SetLogInfo()
	//读取配置文件
	if err := ReadConfig(); err != nil {
		log.Println(err)
		return
	}
	//获取站点
	parseStationNames()
	//设置日志
	if Config.System.LogLevel > 0 {
		SetLevel(Config.System.LogLevel)
	}
	//设置提交订单线程大小
	if Config.System.OrderSize > 1 {
		submitChannel = make(chan int, Config.System.OrderSize)
	}
	//设置查询订单线程大小
	if Config.System.QuerySize > 1 {
		queryChannel = make(chan int, Config.System.QuerySize)
	}
	//设置超时时间
	if Config.System.TimeOut > 0 {
		timeout = time.Duration(Config.System.TimeOut) * time.Second
	}
	//设置CDN
	for _, v := range Config.System.Cdn {
		availableCDN[v] = ""
	}
	Info("==========乘客信息===========")
	Info("从", Config.OrderInfo.FromStation, "到", Config.OrderInfo.ToStation)
	Info("日期", Config.OrderInfo.TrainDate)
	Info("车次", Config.OrderInfo.TrainCode)
	Info("席别", Config.OrderInfo.SeatTypeName)
	Info("乘客", Config.OrderInfo.PassengerName)
	Info("==========乘客信息===========")
	//设置CPU
	//runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	//获取联系人
	go getPassengerDTO()

	if Config.System.ShowCDN {
		go getAllCDN()
	}
	//查询间隔时间
	timer := time.NewTicker(time.Duration(Config.System.RefreshTime) * time.Millisecond)
	for {
		select {
		case <-timer.C:
			Info("查询余票")
			//去多个CDN查询
			for _, date := range Config.OrderInfo.TrainDate { //轮询日期
				for k, _ := range availableCDN {
					go Order(k, date)
				}
			}

		}
	}

}

//转发
func DoForWardRequest(forwardAddress, method, requestUrl string, body io.Reader) string {
	if !strings.Contains(forwardAddress, ":") {
		forwardAddress = forwardAddress + ":80"
	}

	conn, err := net.DialTimeout("tcp", forwardAddress, timeout)
	if err != nil {
		Error("DoForWardRequest error:", err)
		return ""
	}
	defer conn.Close()
	//buf_forward_conn *bufio.Reader
	buf_forward_conn := bufio.NewReader(conn)

	req, err := http.NewRequest(method, requestUrl, body)
	if err != nil {
		Error("DoForWardRequest error:", err)
		return ""
	}
	//add header
	AddReqestHeader(req, method)

	var errWrite error

	errWrite = req.Write(conn)
	if errWrite != nil {
		Error("DoForWardRequest error:", errWrite)
		return ""
	}

	resp, err := http.ReadResponse(buf_forward_conn, req)

	if err != nil {
		Error("DoForWardRequest error:", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body := ParseResponseBody(resp)
		Debug("DoForWardRequest body:", body)
		return body
	} else {
		Error("StatusCode:", resp.StatusCode)
	}
	return ""
}

//获取队列
func getQueueCount(v url.Values, values []string, cdn string) {
	//获取下验证码
	getPassCodeNew(cdn)

	params, _ := url.QueryUnescape(v.Encode())
	Info("getQueueCount Params", params)

	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/getQueueCount", strings.NewReader(params))
	Info("getQueueCount body:", body)

	urlValuesForQueue := url.Values{}
	for k, v := range Config.ConfirmSingleForQueue {
		urlValuesForQueue.Add(k, v)
	}
	urlValuesForQueue.Add("key_check_isChange", values[1])
	urlValuesForQueue.Add("leftTicketStr", values[2])
	urlValuesForQueue.Add("train_location", values[0])
	urlValuesForQueue.Add("passengerTicketStr", passengerTicketStr)
	urlValuesForQueue.Add("oldPassengerStr", oldPassengerStr)
	// 需要延迟提交，提早好像要被踢！！！
	if Config.System.SubmitTime > 1000 {
		time.Sleep(time.Millisecond * time.Duration(Config.System.SubmitTime))
	}

	confirmSingleForQueue(urlValuesForQueue, cdn)
}

//再次确认？
func confirmSingleForQueue(v url.Values, cdn string) {
	Info("confirmSingleForQueue Params:", v.Encode())
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
		<-submitChannel
	}()
	submitChannel <- 1

	params, _ := url.QueryUnescape(v.Encode())
	Debug(params)

	body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/autoSubmitOrderRequest", strings.NewReader(params))
	Info("submitOrderRequest body:", body)

	if strings.Contains(body, `"submitStatus":true`) {
		orderResoult := new(OrderResoult)
		if err := json.Unmarshal([]byte(body), &orderResoult); err != nil {
			Error("submitOrderRequest", err)
			return
		} else {
			v := strings.Split(orderResoult.Data.Result, "#")
			//key_check_isChange=99F79C00DFB9BF8713D23EFA4A8CF06BCA8C412DAC19686DCE306476
			// leftTicketStr = 1002353600401115003110023507803007450039
			// for getQueueCount
			Info("key_check_isChange:", v[1], "leftTicket:", v[2])

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
		log.Println("订票成功！！！！！")
		sendMessage("订票成功！！！！！")
	} else if strings.Contains(body, `用户未登录`) {
		log.Println("用户未登录！！！！！")
		sendMessage("用户未登录！！！！！")
	} else if strings.Contains(body, `取消次数过多`) {
		log.Println("由于您取消次数过多！！！！！")
		sendMessage("由于您取消次数过多！！！！！")
	} else {
		Warn(cdn, "订票请求警告:", body)
	}
}

//queryjs
func queryJs(cdn string) {
	body := DoForWardRequest(cdn, "GET", "https://kyfw.12306.cn/otn/dynamicJs/queryJs", nil)
	Debug("body:", body)
}

//获取新验证码
func getPassCodeNew(cdn string) {
	body := DoForWardRequest(cdn, "GET", "https://kyfw.12306.cn/otn/passcodeNew/getPassCodeNew.do?module=login&rand=sjrand&0.2866508506704122", nil)
	Debug("body:", body)
}

//查询
func Order(cdn, date string) {
	//睡眠下，随机
	//time.Sleep(time.Millisecond * time.Duration(Config.System.SubmitTime))
	//time.Sleep(time.Millisecond * time.Duration(rand.Int63n(Config.System.RefreshTime)))

	// queryJs(cdn)
	defer func() {
		<-queryChannel
	}()
	queryChannel <- 1

	if tickets := queryLeftTicket(cdn, date); tickets != nil { //获取车次
		for _, trainCode := range Config.OrderInfo.TrainCode { //要预订的车次
			trainCode = strings.ToUpper(trainCode)
			for _, data := range tickets.Data { //每个车次
				//查询到的车次
				tkt := data.Ticket
				if tkt.StationTrainCode == trainCode { //是预订的车次
					//获取余票信息
					ticketNum := getTicketNum(tkt.YpInfo, tkt.YpEx)
					numOfTicket := ticketNum[Config.OrderInfo.SeatTypeName]
					if numOfTicket >= len(Config.OrderInfo.PassengerName) { //想要预订席别的余票大于等于订票人的人数
						Info(cdn, "开始订票", date, "车次", tkt.StationTrainCode, "余票", fmt.Sprintf("%v", ticketNum))
						urlValues := url.Values{}
						for k, v := range Config.OrderRequest {
							urlValues.Add(k, v)
						}
						urlValues.Add("secretStr", data.SecretStr)
						urlValues.Add("train_date", date)
						urlValues.Add("query_from_station_name", tkt.FromStationName)
						urlValues.Add("query_to_station_name", tkt.ToStationName)
						urlValues.Add("passengerTicketStr", passengerTicketStr)
						urlValues.Add("oldPassengerStr", oldPassengerStr)

						go submitOrderRequest(urlValues, cdn, tkt)

					} else if numOfTicket > 0 && numOfTicket < len(Config.OrderInfo.PassengerName) {
						Info("车次", tkt.StationTrainCode, "余票不足！！！", fmt.Sprintf("%v", ticketNum))
					}
				} else { //不是预订的车次
					//Debug(tkt.StationTrainCode, "余票", fmt.Sprintf("%v", getTicketNum(tkt.YpInfo, tkt.YpEx)))
				}
			}
		}
	} else {
		Error(cdn, "余票查询错误", tickets)
	}
}

//查询余票
func queryLeftTicket(cdn, trainDate string) *QueryLeftNewDTO {

	leftTicketUrl := "https://kyfw.12306.cn/otn/leftTicket/query?"

	leftTicketUrl += "leftTicketDTO.train_date=" + trainDate + "&"
	leftTicketUrl += "leftTicketDTO.from_station=" + stationMap[Config.OrderInfo.FromStation] + "&"
	leftTicketUrl += "leftTicketDTO.to_station=" + stationMap[Config.OrderInfo.ToStation] + "&"
	leftTicketUrl += "purpose_codes=ADULT"

	Debug("queryLeftTicket url:", leftTicketUrl)
	body := DoForWardRequest(cdn, "GET", leftTicketUrl, nil)
	Debug("queryLeftTicket body:", body)
	leftTicket := new(QueryLeftNewDTO)

	if !strings.Contains(body, "queryLeftNewDTO") {
		Error("查询余票出错，返回:", body, "查询链接:", leftTicketUrl)
		//删除废弃的CDN
		if len(availableCDN) > 5 {
			delete(availableCDN, cdn)
		}
		return nil
	}

	if err := json.Unmarshal([]byte(body), &leftTicket); err != nil {
		Error("queryLeftTicket", cdn, err)
		Error("queryLeftTicket", cdn, body)
		//删除废弃的CDN
		if len(availableCDN) > 5 {
			delete(availableCDN, cdn)
		}
		return nil
	}

	return leftTicket
}

//获取联系人
func getPassengerDTO() {
	success := ""
	for _, cdn := range Config.System.Cdn {
		Info("开始获取联系人！")
		body := DoForWardRequest(cdn, "POST", "https://kyfw.12306.cn/otn/confirmPassenger/getPassengerDTOs", nil)
		Debug("getPassengerDTO body:", body)

		if !strings.Contains(body, "passenger_name") {
			Error("获取联系人出错!!!!!!返回:", body)
			Error("貌似你还没有登录了,或者你的网速太慢了～～")
			success = "no"
			continue
		}

		if err := json.Unmarshal([]byte(body), &passengerDTO); err != nil {
			Error("getPassengerDTO", cdn, err)
			success = "no"
			continue
		} else {
			Info(cdn, "获取成功！")
			ParsePassager()
			success = "yes"
			break
		}
	}
	if success != "yes" {
		os.Exit(1)
	}
}

//解析联系人
func ParsePassager() {
	Debug(passengerDTO)
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

func sendMessage(infos string) {
	Info(infos)

	defer func() {
		cmd := exec.Command(Config.System.Open, Config.System.OpenParams)
		cmd.Start()
		os.Exit(2)
	}()

	if len(Config.System.Mobile) > 0 {
		client := &http.Client{}

		// urls := "http://mobile.alipay.com/main/smsdownload.json?action=smsJsonController&channel=5035&phoneNumber=" + Config.System.Mobile + "&productId=&_input_charset=utf-8&r=1388128497869&_callback=sendSmsCallback"
		urls := "http://mobile.alipay.com/main/smsdownload_constantAddr.json?_callback=jQuery17209914382044225931_1388127995939&serviceCode=PMDR&needCheckCode=false&phoneNumber=" + Config.System.Mobile + "&_=1388128337347"
		reqest, err := http.NewRequest("GET", urls, nil)
		if err != nil {
			Error(err)
			return
		}

		response, err := client.Do(reqest)
		if err != nil {
			Error(err)
			return
		}
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			body := ParseResponseBody(response)
			Info("DoForWardRequest body:", body)
			log.Println("DoForWardRequest body:", body)
		} else {
			Error("StatusCode:", response.StatusCode)
		}
	}
}

func getAllCDN() {
	timer := time.NewTicker(time.Duration(10*1000) * time.Millisecond)
	for {
		select {
		case <-timer.C:

			Info("可用CDN:", fmt.Sprintf("%v", availableCDN))

		}
	}
}
