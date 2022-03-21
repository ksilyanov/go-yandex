package main

import (
	"context"
	"flag"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/server"
	"go-yandex/internal/app/storage"
	"log"
)

var (
	serverURL       *string
	baseURL         *string
	fileStoragePath *string
)

func init() {
	serverURL = flag.String("a", "", "server address")
	baseURL = flag.String("b", "", "base app address")
	fileStoragePath = flag.String("f", "", "url storage file path")
}

func main() {
	curConfig, err := config.GetConfig()
	if err != nil {
		log.Print(err.Error())
		return
	}

	flag.Parse()
	if *serverURL != "" {
		curConfig.ServerURL = *serverURL
	}
	if *baseURL != "" {
		curConfig.BaseURL = *baseURL
	}
	if *fileStoragePath != "" {
		curConfig.FileStoragePath = *fileStoragePath
	}

	s := server.New(storage.New(curConfig), curConfig)

	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		log.Print(err)
	}
}
