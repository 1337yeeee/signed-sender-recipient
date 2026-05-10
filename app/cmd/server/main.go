package main

import (
	"log"

	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/container"
	"electronic-digital-signature/internal/app/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	appContainer, err := container.New(cfg)
	if err != nil {
		log.Fatalf("create app container: %v", err)
	}

	if err := server.New(cfg, appContainer).Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
