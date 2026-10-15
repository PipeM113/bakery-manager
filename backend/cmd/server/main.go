package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
)

// Local development server. In production the same routes run as a Vercel function
// (see api/index.go).
func main() {
	// .env is only for local development; deployed environments set real variables.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatalf("Configuración inválida:\n%v", err)
	}

	handler, closeDB, err := app.Build(context.Background(), cfg, true)
	if err != nil {
		log.Fatalf("Error iniciando la aplicación: %v", err)
	}
	defer closeDB()
	log.Println("DB conectada correctamente")

	log.Printf("Servidor corriendo en puerto %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatal(err)
	}
}
