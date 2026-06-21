package main

import (
	"flag"

	"cryptorates/coin/internal/adapters/config"
	"cryptorates/coin/pkg/application"
)

func main() {
	var path string
	flag.StringVar(&path, "config", "", "Path to configuration file")
	flag.Parse()

	cfg := config.NewConfig(path)

	app := application.New(cfg)
	app.Run()
}
