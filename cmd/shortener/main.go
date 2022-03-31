package main

import (
	"context"
	"flag"
	_ "github.com/jackc/pgx/stdlib"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/server"
	"go-yandex/internal/app/storage"
	"log"
)

var (
	serverURL       *string
	baseURL         *string
	fileStoragePath *string
	dbDsn           *string
)

func init() {
	serverURL = flag.String("a", "", "server address")
	baseURL = flag.String("b", "", "base app address")
	fileStoragePath = flag.String("f", "", "url storage file path")
	dbDsn = flag.String("d", "", "db url")
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
	if *dbDsn != "" {
		curConfig.FileStoragePath = *dbDsn
	}

	ctx := context.Background()

	s := server.New(storage.New(curConfig, ctx), curConfig)

	if err := s.Start(ctx); err != nil {
		log.Print(err)
	}
}
