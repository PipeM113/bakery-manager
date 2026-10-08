// Package app assembles the HTTP API. It is shared by the local server (cmd/server) and
// the Vercel function (api/), so both serve exactly the same routes.
package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	analyticsHand "github.com/PipeM113/bakery-manager/internal/analytics/handler"
	analyticsSvc "github.com/PipeM113/bakery-manager/internal/analytics/service"
	"github.com/PipeM113/bakery-manager/internal/auth/handler"
	"github.com/PipeM113/bakery-manager/internal/auth/service"
	cldSvc "github.com/PipeM113/bakery-manager/internal/cloudinary"
	"github.com/PipeM113/bakery-manager/internal/config"
	costHandler "github.com/PipeM113/bakery-manager/internal/costs/handler"
	costService "github.com/PipeM113/bakery-manager/internal/costs/service"
	fcHand "github.com/PipeM113/bakery-manager/internal/fixed_costs/handler"
	fcRepo "github.com/PipeM113/bakery-manager/internal/fixed_costs/repository"
	fcSvc "github.com/PipeM113/bakery-manager/internal/fixed_costs/service"
	ingHand "github.com/PipeM113/bakery-manager/internal/ingredients/handler"
	ingRepo "github.com/PipeM113/bakery-manager/internal/ingredients/repository"
	ingSvc "github.com/PipeM113/bakery-manager/internal/ingredients/service"
	expHand "github.com/PipeM113/bakery-manager/internal/operational_expenses/handler"
	expRepo "github.com/PipeM113/bakery-manager/internal/operational_expenses/repository"
	expSvc "github.com/PipeM113/bakery-manager/internal/operational_expenses/service"
	quoteHand "github.com/PipeM113/bakery-manager/internal/quotations/handler"
	recHand "github.com/PipeM113/bakery-manager/internal/recipes/handler"
	recRepo "github.com/PipeM113/bakery-manager/internal/recipes/repository"
	saleHand "github.com/PipeM113/bakery-manager/internal/sales/handler"
	saleRepo "github.com/PipeM113/bakery-manager/internal/sales/repository"
	saleSvc "github.com/PipeM113/bakery-manager/internal/sales/service"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
)

// PhotoUploader stores recipe photos. It may be nil: then the upload route answers 503.
type PhotoUploader = recHand.PhotoUploader

// New wires every route. db may be nil in tests that only exercise middleware: the
// constructors below just keep the pool, they do not use it.
func New(cfg config.Config, db *pgxpool.Pool, photos PhotoUploader) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// go-chi/cors treats an empty origin list as "allow everything", so the middleware is
	// only installed when origins are configured. Cookies are never allowed: the token
	// travels in the Authorization header.
	if len(cfg.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: false,
			MaxAge:           600,
		}))
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	authSvc := service.NewAuthService(db, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authSvc)
	r.Post("/auth/login", authHandler.Login)

	ingredientRepo := ingRepo.NewIngredientRepository(db)
	ingredientImportSvc := ingSvc.NewIngredientImportService(db)
	ingredientHandler := ingHand.NewIngredientHandler(ingredientRepo, ingredientImportSvc)

	recipeRepo := recRepo.NewRecipeRepository(db)
	recipeHandler := recHand.NewRecipeHandler(recipeRepo, photos)
	costSvc := costService.NewCostService(db)
	costHandler := costHandler.NewCostHandler(costSvc, recipeRepo)
	fixedCostRepo := fcRepo.NewFixedCostRepository(db)
	fixedCostSvc := fcSvc.NewFixedCostService(fixedCostRepo)
	fixedCostHandler := fcHand.NewFixedCostHandler(fixedCostSvc)

	r.Group(func(r chi.Router) {
		r.Use(mid.Auth(cfg.JWTSecret))
		r.Get("/ingredients", ingredientHandler.GetAll)
		r.Get("/ingredients/export", ingredientHandler.Export)
		r.Post("/ingredients/import", ingredientHandler.Import)
		r.Post("/ingredients", ingredientHandler.Create)
		r.Put("/ingredients/{id}", ingredientHandler.Update)
		r.Delete("/ingredients/{id}", ingredientHandler.Delete)
		r.Get("/ingredients/{id}/history", ingredientHandler.GetPriceHistory)
		r.Get("/recipes", recipeHandler.GetAll)
		r.Get("/recipes/{id}", recipeHandler.GetByID)
		r.Post("/recipes", recipeHandler.Create)
		r.Post("/recipes/{id}/scale", recipeHandler.Scale)
		r.Post("/recipes/{id}/save-scaled", recipeHandler.SaveScaled)
		r.Post("/recipes/{id}/save-as", recipeHandler.SaveAs)
		r.Put("/recipes/{id}", recipeHandler.Update)
		r.Delete("/recipes/{id}", recipeHandler.Delete)
		r.Post("/recipes/{id}/photo", recipeHandler.UploadPhoto)
		r.Get("/recipes/{id}/cost", costHandler.GetBreakdown)
		r.Post("/recipes/{id}/cost/simulate", costHandler.Simulate)
		r.Get("/recipes/{id}/costs", costHandler.GetCosts)
		r.Put("/recipes/{id}/costs", costHandler.UpdateCosts)
		r.Get("/fixed-costs", fixedCostHandler.List)
		r.Post("/fixed-costs", fixedCostHandler.Create)
		r.Put("/fixed-costs/{id}", fixedCostHandler.Update)
		r.Delete("/fixed-costs/{id}", fixedCostHandler.Delete)
		quotationHandler := quoteHand.NewQuotationHandler(costSvc, db)
		r.Post("/quotations/generate", quotationHandler.Generate)
		r.Get("/quotations", quotationHandler.List)
		r.Put("/quotations/{id}/confirm", quotationHandler.Confirm)
		r.Put("/quotations/{id}/cancel", quotationHandler.Cancel)

		saleRepository := saleRepo.NewSaleRepository(db)
		saleService := saleSvc.NewSaleService(saleRepository)
		saleHandler := saleHand.NewSaleHandler(saleService)
		r.Post("/sales", saleHandler.Create)
		r.Get("/sales", saleHandler.List)
		r.Delete("/sales/{id}", saleHandler.Delete)

		expenseRepository := expRepo.NewExpenseRepository(db)
		expenseService := expSvc.NewExpenseService(expenseRepository)
		expenseHandler := expHand.NewExpenseHandler(expenseService)
		r.Post("/expenses", expenseHandler.Create)
		r.Get("/expenses", expenseHandler.List)
		r.Put("/expenses/{id}", expenseHandler.Update)
		r.Delete("/expenses/{id}", expenseHandler.Delete)

		analyticsService := analyticsSvc.NewAnalyticsService(db)
		analyticsHandler := analyticsHand.NewAnalyticsHandler(analyticsService)
		r.Get("/analytics/monthly", analyticsHandler.Monthly)
		r.Get("/analytics/recipes", analyticsHandler.Recipes)
		r.Get("/analytics/trends", analyticsHandler.Trends)
	})

	return r
}

// Build opens the database pool and the optional photo service and returns the API plus a
// function that releases the pool. With pingDB the database must answer now (local server:
// fail fast); without it the first request connects (serverless: never fail a cold start).
func Build(ctx context.Context, cfg config.Config, pingDB bool) (http.Handler, func(), error) {
	pool, err := kernel.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return nil, nil, err
	}
	if pingDB {
		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("la base de datos no responde")
		}
	}

	var photos PhotoUploader
	if cfg.CloudinaryURL == "" {
		log.Println("Cloudinary no configurado: la subida de fotos queda deshabilitada")
	} else if svc, err := cldSvc.NewServiceFromURL(cfg.CloudinaryURL); err != nil {
		log.Printf("Cloudinary no disponible (fotos deshabilitadas): %v", err)
	} else {
		photos = svc
	}

	return New(cfg, pool, photos), pool.Close, nil
}

// NewLazyHandler builds the application on the first request, from the environment read
// through getenv, and reuses it afterwards. If the configuration is invalid every request
// gets the same generic 500; the details go to the log only.
func NewLazyHandler(getenv func(string) string) http.HandlerFunc {
	var (
		once sync.Once
		h    http.Handler
	)
	return func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() {
			cfg, err := config.Load(getenv)
			if err == nil {
				h, _, err = Build(context.Background(), cfg, false)
			}
			if err != nil {
				log.Printf("el servicio no pudo iniciar: %v", err)
				h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					httputil.JSONError(w, "configuración del servidor inválida", http.StatusInternalServerError)
				})
			}
		})
		h.ServeHTTP(w, r)
	}
}
