package main

import (
	"context"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/server"
	"go-yandex/internal/app/storage"
	"log"
)

func main() {
	curConfig := config.GetConfig()
	s := server.New(storage.New(curConfig), curConfig)

	ctx := context.Background()

	if e := s.Start(ctx); e != nil {
		log.Print(e)
	}
}
