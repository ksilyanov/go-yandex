package main

import (
	"context"
	"go-yandex/internal/app/server"
	"go-yandex/internal/app/storage"
	"log"
)

func main() {
	s := server.New("localhost:8080", storage.New())

	ctx := context.Background()

	if e := s.Start(ctx); e != nil {
		log.Print(e)
	}
}
