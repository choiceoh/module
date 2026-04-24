package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"module-backend/internal/ai"
	"module-backend/internal/handler"
)

func main() {
	baseURL := envOr("VLLM_BASE_URL", "http://100.105.145.6:8000/v1")
	model := envOr("VLLM_MODEL", "google/gemma-4-27b-a4b-it")
	addr := envOr("ADDR", ":8080")

	aiClient := ai.New(baseURL, model)
	h := handler.New(aiClient)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(180 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Content-Disposition"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/api/health", h.Health)
	r.Post("/api/extract", h.Extract)
	r.Post("/api/export", h.Export)

	log.Printf("module backend listening on %s (vllm=%s model=%s)", addr, baseURL, model)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
