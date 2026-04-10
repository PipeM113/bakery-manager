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
	analyticsHand "github.com/PipeM113/bakery-manager/internal/analytics/handler"
	analyticsSvc "github.com/PipeM113/bakery-manager/internal/analytics/service"
	costHandler "github.com/PipeM113/bakery-manager/internal/costs/handler"
	costService "github.com/PipeM113/bakery-manager/internal/costs/service"
	fcHand "github.com/PipeM113/bakery-manager/internal/fixed_costs/handler"
	fcRepo "github.com/PipeM113/bakery-manager/internal/fixed_costs/repository"
	fcSvc "github.com/PipeM113/bakery-manager/internal/fixed_costs/service"
	ingHand "github.com/PipeM113/bakery-manager/internal/ingredients/handler"
	ingRepo "github.com/PipeM113/bakery-manager/internal/ingredients/repository"
	ingSvc "github.com/PipeM113/bakery-manager/internal/ingredients/service"
	cldSvc "github.com/PipeM113/bakery-manager/internal/cloudinary"
	quoteHand "github.com/PipeM113/bakery-manager/internal/quotations/handler"
	recHand "github.com/PipeM113/bakery-manager/internal/recipes/handler"
	recRepo "github.com/PipeM113/bakery-manager/internal/recipes/repository"
	expHand "github.com/PipeM113/bakery-manager/internal/operational_expenses/handler"
	expRepo "github.com/PipeM113/bakery-manager/internal/operational_expenses/repository"
	expSvc "github.com/PipeM113/bakery-manager/internal/operational_expenses/service"
	saleHand "github.com/PipeM113/bakery-manager/internal/sales/handler"
	saleRepo "github.com/PipeM113/bakery-manager/internal/sales/repository"
	saleSvc "github.com/PipeM113/bakery-manager/internal/sales/service"
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

	allowedOrigins := []string{"http://localhost:5173"}
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
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

	ingredientRepo := ingRepo.NewIngredientRepository(db)
	ingredientImportSvc := ingSvc.NewIngredientImportService(db)
	ingredientHandler := ingHand.NewIngredientHandler(ingredientRepo, ingredientImportSvc)

	var cloudinaryUploader recHand.PhotoUploader
	if cld, cldErr := cldSvc.NewService(); cldErr != nil {
		log.Printf("Cloudinary no configurado (fotos deshabilitadas): %v", cldErr)
	} else {
		cloudinaryUploader = cld
	}

	recipeRepo := recRepo.NewRecipeRepository(db)
	recipeHandler := recHand.NewRecipeHandler(recipeRepo, cloudinaryUploader)
	costSvc := costService.NewCostService(db)
	costHandler := costHandler.NewCostHandler(costSvc, recipeRepo)
	fixedCostRepo := fcRepo.NewFixedCostRepository(db)
	fixedCostSvc := fcSvc.NewFixedCostService(fixedCostRepo)
	fixedCostHandler := fcHand.NewFixedCostHandler(fixedCostSvc)

	r.Group(func(r chi.Router) {
		r.Use(mid.AuthMiddleware)
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
		saleSvc := saleSvc.NewSaleService(saleRepository)
		saleHandler := saleHand.NewSaleHandler(saleSvc)
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

	log.Printf("Servidor corriendo en puerto %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
