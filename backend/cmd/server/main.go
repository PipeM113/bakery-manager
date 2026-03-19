package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"github.com/PipeM113/bakery-manager/internal/auth/handler"
	"github.com/PipeM113/bakery-manager/internal/auth/service"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
)

func main() {
	// carga .env solo en desarrollo, en Railway las vars vienen del entorno
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	db, err := kernel.NewDB()
	if err != nil {
		log.Fatalf("Error conectando a la DB: %v", err)
	}
	defer db.Close()
	log.Println("DB conectada correctamente")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "https://*.vercel.app"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})
	authSvc := service.NewAuthService(db)
	authHandler := handler.NewAuthHandler(authSvc)

	r.Post("/auth/login", authHandler.Login)

	// rutas protegidas van así:
	r.Group(func(r chi.Router) {
		r.Use(mid.AuthMiddleware)
		r.Get("/ingredients", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"message":"ok"}`))
		})
	})

	log.Printf("Servidor corriendo en puerto %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
