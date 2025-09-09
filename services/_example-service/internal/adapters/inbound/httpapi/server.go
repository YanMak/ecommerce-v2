package httpapi

import (
	"context"
	"lesson2/adapters/inbound/httpapi/search"
	appsearch "lesson2/app/search"
	"net/http"
	"time"

	_ "lesson2/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	// 	httpSwagger "github.com/swaggo/http-swagger"
	// _ "lesson2/docs"
	"lesson2/adapters/outbound/postgres"
	pgrepo "lesson2/adapters/outbound/repo/postgres"
)

type Server struct{ router chi.Router }

// @title       Architecture-1 API
// @version     1.0
// @description Training project: super-endpoint covering full HTTP inputs with validation.
// @BasePath    /v1
// @schemes     http
func NewServer() *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.Logger, middleware.Compress(5))

	// --- PG + миграции ---
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := postgres.NewPool(ctx)
	if err != nil {
		panic(err)
	}
	if err := postgres.Migrate(ctx); err != nil {
		panic(err)
	}
	// ----------------------

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/v1", func(r chi.Router) {
		repo := pgrepo.New(db)
		uc := appsearch.NewUseCase(repo)
		search.New(uc).Routes(r)
	})

	// тут позже смонтируем /v1 и контроллеры
	return &Server{router: r}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
