package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/seed"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/service"
)

func main() {
	dbEnv := os.Getenv("DB_ENV")
	if dbEnv == "" {
		dbEnv = "prod"
	}
	if dbEnv != "prod" && dbEnv != "test" {
		log.Fatalf("DB_ENV must be \"prod\" or \"test\", got %q", dbEnv)
	}

	dbURL := selectDatabase(dbEnv)
	log.Printf("DB_ENV=%s", dbEnv)
	handler.SetDbTarget(dbEnv)

	dataServiceURL := os.Getenv("DATA_SERVICE_URL")
	if dataServiceURL == "" {
		log.Fatal("DATA_SERVICE_URL is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Printf("Connected to database (%s)", dbEnv)

	// Run migrations
	runMigrations(db)

	// Reseed test database if requested
	if dbEnv == "test" && os.Getenv("RESEED_TEST_DB") == "true" {
		if err := seed.ReseedDatabase(db); err != nil {
			log.Fatalf("Failed to reseed database: %v", err)
		}
	} else if dbEnv == "test" {
		if err := seed.SeedIfEmpty(db); err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
	}

	// Repositories
	stockRepo := &repository.StockRepo{DB: db}
	txnRepo := &repository.TransactionRepo{DB: db}
	portfolioRepo := &repository.PortfolioRepo{DB: db}
	priceCacheRepo := &repository.PriceCacheRepo{DB: db}
	importRepo := &repository.ImportRepo{DB: db}
	stagingRepo := &repository.StagingRepo{DB: db}

	// Clients
	dataClient := client.NewDataServiceClient(dataServiceURL)
	historyCache := service.NewHistoryCache(15 * time.Minute)

	// Services
	txnSvc := &service.TransactionService{
		StockRepo: stockRepo,
		TxnRepo:   txnRepo,
	}
	portfolioSvc := &service.PortfolioService{
		PortfolioRepo:  portfolioRepo,
		PriceCacheRepo: priceCacheRepo,
		DataClient:     dataClient,
		HistoryCache:   historyCache,
	}
	importSvc := &service.ImportService{
		StockRepo:  stockRepo,
		ImportRepo: importRepo,
		DataClient: dataClient,
	}
	stagingSvc := &service.StagingService{
		StagingRepo: stagingRepo,
		StockRepo:   stockRepo,
		ImportRepo:  importRepo,
		DataClient:  dataClient,
		UploadDir:   "/tmp/imports",
	}

	// Handlers
	txnHandler := &handler.TransactionHandler{Svc: txnSvc}
	portfolioHandler := &handler.PortfolioHandler{Svc: portfolioSvc}
	importHandler := &handler.ImportHandler{Svc: importSvc}
	uploadHandler := &handler.UploadHandler{Svc: stagingSvc}

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", handler.Health)

	r.Route("/api", func(r chi.Router) {
		r.Post("/transactions", txnHandler.Create)
		r.Get("/transactions", txnHandler.List)
		r.Get("/transactions/{id}", txnHandler.GetByID)
		r.Put("/transactions/{id}", txnHandler.Update)
		r.Delete("/transactions/{id}", txnHandler.Delete)

		r.Get("/portfolio", portfolioHandler.GetPortfolio)
		r.Get("/prices/{ticker}", portfolioHandler.GetPrice)
		r.Get("/prices/{ticker}/history", portfolioHandler.GetPriceHistory)

		r.Post("/import/alpaca", importHandler.ImportAlpaca)

		r.Post("/imports/upload", uploadHandler.Upload)
		r.Get("/imports/{importId}", uploadHandler.GetImport)
		r.Patch("/imports/{importId}/rows/{rowId}", uploadHandler.PatchRow)
		r.Post("/imports/{importId}/confirm", uploadHandler.Confirm)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting backend on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migrations complete")
}

// selectDatabase returns the connection string for the given environment.
func selectDatabase(env string) string {
	prodURL := os.Getenv("DATABASE_URL_PROD")
	testURL := os.Getenv("DATABASE_URL_TEST")

	if prodURL == "" {
		log.Fatal("DATABASE_URL_PROD is required")
	}
	if testURL == "" {
		log.Fatal("DATABASE_URL_TEST is required")
	}

	if env == "prod" {
		return prodURL
	}
	return testURL
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
