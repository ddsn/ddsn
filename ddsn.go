package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
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
	HttpPort    int
	DdsnPort    int
	TmlDir      string
	ResDir      string
	SqlDir      string
	DbFile      string
	RsaKeyFile  string
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
var Key *rsa.PrivateKey
var Identity [32]byte
var PeerName string

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
			fmt.Println("Error reading database initialization from "+Config.Server.SqlDir+"/init.sql: "+err.Error())
			return
		}

		Database.Exec(string(bytes))

		fmt.Println("Initialized database")
	}

	// </Connect to sqlite database>

	// <RSA keys>

	if _, err := os.Stat(Config.Server.RsaKeyFile); os.IsNotExist(err) {
		Key, err = rsa.GenerateKey(rand.Reader, 1024)

		if err != nil {
			fmt.Println("Error generating RSA key: " + err.Error())
			return
		}

		block := pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(Key)}

		pemData := pem.EncodeToMemory(&block)

		err = ioutil.WriteFile(Config.Server.RsaKeyFile, pemData, 0644)

		if err != nil {
			fmt.Println("Error writing RSA key to "+Config.Server.RsaKeyFile+": "+err.Error())
			return
		}

		fmt.Println("Saved RSA key to "+Config.Server.RsaKeyFile)
	} else {
		bytes, err := ioutil.ReadFile(Config.Server.RsaKeyFile)

		if err != nil {
			fmt.Println("Error reading RSA key from "+Config.Server.RsaKeyFile+": "+err.Error())
			return
		}

		block, _ := pem.Decode(bytes)

		fmt.Println("Read RSA key from "+Config.Server.RsaKeyFile)

		Key, err = x509.ParsePKCS1PrivateKey(block.Bytes)

		if err != nil {
			fmt.Println("Error parsing key bytes: "+err.Error())
			return
		}
	}

	publicBytes, _ := x509.MarshalPKIXPublicKey(&Key.PublicKey)
	Identity = sha256.Sum256(publicBytes)
	PeerName = hex.EncodeToString(Identity[:])[0:6]

	fmt.Println("Your peer name is "+PeerName)

	// </RSA keys>

	// <Start HTTP server>

	fmt.Println("Listening for HTTP requests...")

	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":"+strconv.Itoa(Config.Server.HttpPort), nil)

	// </Start HTTP server>
}