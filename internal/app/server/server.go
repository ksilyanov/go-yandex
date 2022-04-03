package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/handlers"
	"go-yandex/internal/app/middlewares/compressor"
	"go-yandex/internal/app/middlewares/cookiemanager"
	"go-yandex/internal/app/storage"
	"log"
	"net/http"
)

type Server interface {
	chi.Router
}

type server struct {
	repository storage.URLRepository
	config     config.Config
}

func New(rep storage.URLRepository, config config.Config) *server {
	return &server{
		rep,
		config,
	}
}

func (s *server) Start(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		if err := http.ListenAndServe(s.config.ServerURL, GetRouter(s.repository, s.config)); err != nil && err != http.ErrServerClosed {
			log.Printf("listener failed:+%v\n", err)
			cancel()
		}

	}()
	<-ctx.Done()

	return nil
}

func GetRouter(repository storage.URLRepository, config config.Config) chi.Router {
	r := chi.NewRouter()
	r.Use(
		middleware.Logger,
		compressor.GzipHandler,
		cookiemanager.Handler,
	)

	r.Post("/", handlers.SaveURL(repository, config))
	r.Get("/{id}", handlers.GetURL(repository))
	r.Route("/api", func(r chi.Router) {
		r.Post("/shorten", handlers.SaveURLJson(repository, config))
		r.Post("/shorten/batch", handlers.SaveBatch(repository, config))
		r.Get("/user/urls", handlers.GetForUser(repository, config))
	})
	r.Get("/ping", handlers.GetDBStatus(repository))

	return r
}
