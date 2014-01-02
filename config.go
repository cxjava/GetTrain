package main

type config struct {
	Login                 login
	Query                 query
	Submit                submit
	System                system
	LeftTicket            map[string]string `toml:"left_ticket"`
	OrderRequest          map[string]string `toml:"order_request"`
	GetQueueCountRequest  map[string]string `toml:"get_queue_count"`
	ConfirmSingleForQueue map[string]string `toml:"confirm_single_for_queue"`
}

type login struct {
	Cookie    string
	UserAgent string `toml:"user_agent"`
}

type query struct {
	TrainDate    string `toml:"train_date"`
	FromStation  string `toml:"from_station"`
	ToStation    string `toml:"to_station"`
	PurposeCodes string `toml:"purpose_codes"`
}

type submit struct {
	TrainCode          string `toml:"train_code"`
	PassengerTicketStr string `toml:"passenger_ticket_str"`
	OldPassengerStr    string `toml:"old_passenger_str"`
}

type system struct {
	Proxy    bool
	ProxyUrl string `toml:"proxy_url"`
	Cdn      []string
}
