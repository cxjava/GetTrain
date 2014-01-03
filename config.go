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
	RefreshTime int64  `toml:"refresh_time"`
	SubmitTime  int64  `toml:"submit_time"`
	Cdn         []string
	ShowCDN     bool `toml:"show_cdn"`
	Mobile      string
}
