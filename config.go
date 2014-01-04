package main

type config struct {
	Login                 login
	OrderInfo             orderInfo `toml:"order_info"`
	System                system
	OrderRequest          map[string]string `toml:"order_request"`
	GetQueueCountRequest  map[string]string `toml:"get_queue_count"`
	ConfirmSingleForQueue map[string]string `toml:"confirm_single_for_queue"`
}

type login struct {
	Cookie    string
	UserAgent string `toml:"user_agent"`
}

type orderInfo struct {
	TrainCode     []string `toml:"train_code"`
	TrainDate     []string `toml:"train_date"`
	FromStation   string   `toml:"from_station"`
	ToStation     string   `toml:"to_station"`
	PassengerName []string `toml:"passenger_name"`
	SeatType      string   `toml:"seat_type"`
	SeatTypeName  string   `toml:"seat_type_name"`
}

type system struct {
	Proxy       bool
	OpenParams  string `toml:"open_params"`
	Open        string `toml:"open"`
	ProxyUrl    string `toml:"proxy_url"`
	LogLevel    int    `toml:"log_level"`
	OrderSize   int    `toml:"order_size"`
	QuerySize   int    `toml:"query_size"`
	RefreshTime int64  `toml:"refresh_time"` //查询订单时间
	SubmitTime  int64  `toml:"submit_time"`  //提交订单的停顿时间
	Cdn         []string
	ShowCDN     bool   `toml:"show_cdn"` //是否显示CDN的过滤结果
	Mobile      string //成功后的短信提示
	TimeOut     int    `toml:"time_out"` //DoForWardRequest()的超时时间
	GoBoth      bool   `toml:"go_both"`  //是否并行执行 getQueueCount()和confirmSingleForQueue()
}
