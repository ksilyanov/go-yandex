package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go-yandex/internal/app/handlers"
	"go-yandex/internal/app/storage"
	"log"
	"net/http"
)

type Server interface {
	chi.Router
}

type server struct {
	url        string
	repository storage.URLRepository
}

func New(addr string, rep storage.URLRepository) *server {
	return &server{
		addr,
		rep,
	}
}

func (s *server) Start(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		if err := http.ListenAndServe(s.url, GetRouter(s.repository)); err != nil && err != http.ErrServerClosed {
			log.Printf("listener failed:+%v\n", err)
			cancel()
		}

	}()
	<-ctx.Done()

	return nil
}

func GetRouter(repository storage.URLRepository) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/", handlers.SaveURL(repository))
	r.Get("/{id}", handlers.GetURL(repository))
	r.Route("/api", func(r chi.Router) {
		r.Post("/shorten", handlers.SaveURLJson(repository))
	})

	return r
}
