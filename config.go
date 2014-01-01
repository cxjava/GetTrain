package main

var Config config

type config struct {
	Login  login
	Query  query
	Submit submit
	System system
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
	PassengerTicketStr string `toml:"passenger_ticket_str"`
	OldPassengerStr    string `toml:"old_passenger_str"`
}

type system struct {
	Proxy    bool
	ProxyUrl string `toml:"proxy_url"`
	Cdn      []string
}
