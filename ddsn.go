package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type server struct {
	HttpPort  int
	DdsnPort  int
	TmlDir    string
	ResDir    string
	SqlDir    string
	DbFile    string
}

type client struct {
}

type config struct {
	Server  server
	Client  client
}

var Config config
var Sessions map[string]int
var Database *sql.DB

func main() {
	Sessions = make(map[string]int)

	fmt.Println("Welcome to DDSN 1.0")

	// <Load configuration>

	var configFile = flag.String("config", "config.xml", "file which contains configuration")

	xmlFile, err := os.Open(*configFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	fmt.Println("Loaded config information from "+(*configFile))

	bytes, _ := ioutil.ReadAll(xmlFile)

	xmlFile.Close()

	xml.Unmarshal(bytes, &Config)

	// </Load configuration>

	// <Connect to sqlite database>

	initDb := false
	if _, err := os.Stat(Config.Server.DbFile); os.IsNotExist(err) {
		initDb = true
	}

	Database, err = sql.Open("sqlite3", Config.Server.DbFile)
	defer Database.Close()

	if err != nil {
		fmt.Println("Error connecting to database: "+err.Error())
		return
	}

	fmt.Println("Connected to database "+Config.Server.DbFile)

	if initDb {
		bytes, err := ioutil.ReadFile(Config.Server.SqlDir + "/init.sql")

		if err != nil {
			fmt.Println("Error opening database initialization file init.sql: " + err.Error())
			return
		}

		Database.Exec(string(bytes))

		fmt.Println("Initialized database")
	}

	// </Connect to sqlite database>

	// <Start HTTP server>

	fmt.Println("Listening for HTTP requests...")

	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":"+strconv.Itoa(Config.Server.HttpPort), nil)

	// </Start HTTP server>
}