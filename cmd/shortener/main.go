package main

import (
	"context"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/server"
	"go-yandex/internal/app/storage"
	"log"
)

func main() {
	s := server.New(storage.New(), config.GetConfig())

	ctx := context.Background()

	if e := s.Start(ctx); e != nil {
		log.Print(e)
	}
}
