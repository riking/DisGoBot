package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"encoding/json"

	"github.com/riking/discourse/discourse"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "config.json", "configuration file to load")
}

func fatal(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func setup() {
	file, err := os.Open(configFile)
	fatal(err)
	jsonBlob, err := ioutil.ReadAll(file)
	fatal(err)

	var config discourse.Config
	err = json.Unmarshal(jsonBlob, &config)
	fatal(err)
}

func main() {
	log.Println("Starting up...")
	flag.Parse()

}
