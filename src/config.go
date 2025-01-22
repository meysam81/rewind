package requestrewind

import "flag"

type Config struct {
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	ServerPort       string
	TestDomain       string
	DBMaxConnections int
}

func NewConfig() *Config {
	serverPort := flag.String("p", "8080", "Port to run the server on")
	dbHost := flag.String("dbhost", "localhost", "Database host")
	dbPort := flag.String("dbport", "5432", "Database port")
	dbUser := flag.String("dbuser", "requestrewind", "Database user")
	dbPass := flag.String("dbpass", "requestrewind", "Database password")
	dbName := flag.String("dbname", "requestrewind", "Database name")
	testDomain := flag.String("d", "", "Domain to test against")
	dbMaxConn := flag.Int("dbmaxconn", 10, "Maximum number of database connections")
	flag.Parse()

	config := &Config{
		DBHost:           *dbHost,
		DBPort:           *dbPort,
		DBUser:           *dbUser,
		DBPassword:       *dbPass,
		DBName:           *dbName,
		ServerPort:       *serverPort,
		TestDomain:       *testDomain,
		DBMaxConnections: *dbMaxConn,
	}

	return config
}
