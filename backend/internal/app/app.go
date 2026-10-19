// Package app assembles the HTTP API. It is shared by the local server (cmd/server) and
// the Vercel function (api/), so both serve exactly the same routes.
package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	bizHand "github.com/PipeM113/bakery-manager/internal/business/handler"
	bizRepo "github.com/PipeM113/bakery-manager/internal/business/repository"
	"github.com/PipeM113/bakery-manager/internal/config"
	costHandler "github.com/PipeM113/bakery-manager/internal/costs/handler"
	costService "github.com/PipeM113/bakery-manager/internal/costs/service"
	catHand "github.com/PipeM113/bakery-manager/internal/expense_categories/handler"
	catRepo "github.com/PipeM113/bakery-manager/internal/expense_categories/repository"
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
	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	saHand "github.com/PipeM113/bakery-manager/internal/stock_adjustments/handler"
	saRepo "github.com/PipeM113/bakery-manager/internal/stock_adjustments/repository"
	"github.com/PipeM113/bakery-manager/internal/storage"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
)

// maxJSONBody is the largest JSON body any endpoint reads (1 MiB).
const maxJSONBody = 1 << 20

// PhotoUploader stores recipe photos. It may be nil: then the upload route answers 503.
type PhotoUploader = recHand.PhotoUploader

// New wires every route. db may be nil in tests that only exercise middleware: the
// constructors below just keep the pool, they do not use it.
func New(cfg config.Config, db *pgxpool.Pool, photos PhotoUploader) http.Handler {
	return NewWithClock(cfg, db, photos, clock.System)
}

// NewWithClock is New with the clock the handlers use for "today" and "this month".
func NewWithClock(cfg config.Config, db *pgxpool.Pool, photos PhotoUploader, clk clock.Clock) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// JSON bodies are small; only the photo upload (multipart, with its own limit) may be bigger.
	r.Use(mid.MaxBody(maxJSONBody, func(r *http.Request) bool {
		return r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/recipes/") && strings.HasSuffix(r.URL.Path, "/photo")
	}))

	// go-chi/cors treats an empty origin list as "allow everything", so the middleware is
	// only installed when origins are configured. Cookies are never allowed: the token
	// travels in the Authorization header.
	if len(cfg.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Retry-After"}, // the login page shows how long to wait
			AllowCredentials: false,
			MaxAge:           600,
		}))
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	authSvc := service.NewAuthService(db, cfg.JWTSecret, clk)
	authHandler := handler.NewAuthHandler(authSvc)
	r.Post("/auth/login", authHandler.Login)

	ingredientRepo := ingRepo.NewIngredientRepository(db, clk)
	ingredientImportSvc := ingSvc.NewIngredientImportService(db, clk)
	ingredientHandler := ingHand.NewIngredientHandler(ingredientRepo, ingredientImportSvc, clk)

	recipeRepo := recRepo.NewRecipeRepository(db)
	recipeHandler := recHand.NewRecipeHandler(recipeRepo, photos)
	costSvc := costService.NewCostService(db)
	costHandler := costHandler.NewCostHandler(costSvc, recipeRepo)
	stockAdjustmentHandler := saHand.New(saRepo.New(db), clk)
	fixedCostRepo := fcRepo.NewFixedCostRepository(db)
	fixedCostSvc := fcSvc.NewFixedCostService(fixedCostRepo)
	fixedCostHandler := fcHand.NewFixedCostHandler(fixedCostSvc, clk)

	r.Group(func(r chi.Router) {
		r.Use(mid.Auth(cfg.JWTSecret))
		r.Put("/auth/password", authHandler.ChangePassword)
		r.Get("/ingredients", ingredientHandler.GetAll)
		r.Get("/ingredients/export", ingredientHandler.Export)
		r.Post("/ingredients/import", ingredientHandler.Import)
		r.Post("/ingredients", ingredientHandler.Create)
		r.Put("/ingredients/{id}", ingredientHandler.Update)
		r.Delete("/ingredients/{id}", ingredientHandler.Delete)
		r.Get("/ingredients/{id}/history", ingredientHandler.GetPriceHistory)
		r.Get("/ingredients/{id}/stock-adjustments", stockAdjustmentHandler.ForIngredient)
		r.Post("/ingredients/{id}/stock-adjustments", stockAdjustmentHandler.Create)
		r.Get("/stock-adjustments", stockAdjustmentHandler.ForMonth)
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
		businessRepository := bizRepo.New(db)
		businessHandler := bizHand.NewBusinessHandler(businessRepository, cfg.Business)
		r.Get("/business", businessHandler.Get)
		r.Put("/business", businessHandler.Save)
		quotationHandler := quoteHand.NewQuotationHandler(costSvc, db, clk, businessRepository, cfg.Business)
		r.Post("/quotations", quotationHandler.Create)
		r.Get("/quotations", quotationHandler.List)
		r.Get("/quotations/{id}", quotationHandler.Get)
		r.Get("/quotations/{id}/pdf", quotationHandler.PDF)
		r.Put("/quotations/{id}/confirm", quotationHandler.Confirm)
		r.Put("/quotations/{id}/cancel", quotationHandler.Cancel)

		saleRepository := saleRepo.NewSaleRepository(db)
		saleService := saleSvc.NewSaleService(saleRepository)
		saleHandler := saleHand.NewSaleHandler(saleService, clk)
		r.Post("/sales", saleHandler.Create)
		r.Get("/sales", saleHandler.List)
		r.Delete("/sales/{id}", saleHandler.Delete)
		orderHandler := saleHand.NewOrderHandler(saleService, clk)
		r.Post("/orders", orderHandler.Create)
		r.Get("/orders", orderHandler.List)
		r.Get("/orders/{id}", orderHandler.Get)
		r.Delete("/orders/{id}", orderHandler.Delete)

		expenseRepository := expRepo.NewExpenseRepository(db)
		expenseService := expSvc.NewExpenseService(expenseRepository)
		expenseHandler := expHand.NewExpenseHandler(expenseService, clk)
		categoryHandler := catHand.NewCategoryHandler(catRepo.NewCategoryRepository(db))
		r.Get("/expense-categories", categoryHandler.List)
		r.Post("/expense-categories", categoryHandler.Create)
		r.Put("/expense-categories/{id}", categoryHandler.Rename)
		r.Delete("/expense-categories/{id}", categoryHandler.Delete)
		r.Post("/expenses", expenseHandler.Create)
		r.Get("/expenses", expenseHandler.List)
		r.Put("/expenses/{id}", expenseHandler.Update)
		r.Delete("/expenses/{id}", expenseHandler.Delete)

		analyticsService := analyticsSvc.NewAnalyticsService(db)
		analyticsHandler := analyticsHand.NewAnalyticsHandler(analyticsService, clk)
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
	if cfg.SupabaseURL == "" {
		log.Println("Supabase Storage no configurado: la subida de fotos queda deshabilitada")
	} else {
		photos = storage.New(storage.Config{URL: cfg.SupabaseURL, SecretKey: cfg.SupabaseSecretKey, Bucket: cfg.SupabaseBucket}, nil)
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
