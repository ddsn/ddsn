package main

import (
	"fmt"
	"flag"
	"os"
	"io/ioutil"

	"encoding/xml"
)

func main() {
	fmt.Println("Welcome to DDSN 1.0")

	// <Load configuration>

	var configFile = flag.String("config", "config.xml", "file which contains configuration")
	fmt.Println("Loading config information from " +(*configFile)+"...")

	xmlFile, err := os.Open(*configFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	b, _ := ioutil.ReadAll(xmlFile)

	xmlFile.Close()

	type server struct {
		HttpPort  int
		DdsnPort  int
		TmlDir    string
	}

	type client struct {
	}

	type config struct {
		Server  server
		Client  client
	}

	var c config
	xml.Unmarshal(b, &c)

	// </Load configuration>
}