package rewind

import "flag"

type Config struct {
	DSN        string
	ServerPort string
	TestDomain string
}

func NewConfig() *Config {
	serverPort := flag.String("p", "8080", "Port to run the server on")
	dsn := flag.String("dsn", "postgres://rewind:rewind@localhost/rewind?sslmode=disable", "Postgres DSN")
	testDomain := flag.String("d", "", "Domain to test against")
	flag.Parse()

	config := &Config{
		DSN:        *dsn,
		ServerPort: *serverPort,
		TestDomain: *testDomain,
	}

	return config
}
