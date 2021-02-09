package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/sh4ggyy/Project/config"
	"github.com/sh4ggyy/Project/handler"
	"github.com/sh4ggyy/Project/web"
)

var configFile = flag.String("config", "config/config.json", "Configuration file in JSON-format")

var cfg config.Configuration

func main() {
	flag.Parse()
	var FilePath string
	if len(*configFile) > 0 {
		FilePath = *configFile
	}
	err := config.LoadConfiguration(FilePath)
	if err != nil {
		fmt.Println("Error loading configuration file ", "Error : ", err.Error())
		os.Exit(1)
	} else {
		cfg = config.Config
	}
	http.HandleFunc("/", web.HandleMain)
	http.HandleFunc("/login", handler.HandleGitHubLogin)
	fmt.Print("Started running on http://127.0.0.1:8080\n")
	fmt.Println(http.ListenAndServe(cfg.ListenURL, nil))
}
