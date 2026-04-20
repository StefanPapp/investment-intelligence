package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	// Determine git branch to select the right database
	branch := gitBranch()
	dbURL, dbTarget := selectDatabase(branch)

	log.Printf("Branch: %s → database: %s", branch, dbTarget)
	handler.SetDbTarget(dbTarget)

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
	log.Printf("Connected to database (%s)", dbTarget)

	// Run migrations
	runMigrations(db)

	// Seed test database if empty
	if dbTarget == "test" {
		if err := seed.SeedIfEmpty(db); err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
	}

	// Repositories
	stockRepo := &repository.StockRepo{DB: db}
	txnRepo := &repository.TransactionRepo{DB: db}
	portfolioRepo := &repository.PortfolioRepo{DB: db}
	priceCacheRepo := &repository.PriceCacheRepo{DB: db}

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

	// Handlers
	txnHandler := &handler.TransactionHandler{Svc: txnSvc}
	portfolioHandler := &handler.PortfolioHandler{Svc: portfolioSvc}

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

// gitBranch returns the current git branch name, or "unknown" if git is
// unavailable (e.g. inside a Docker image with no .git directory).
func gitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		log.Printf("Warning: could not detect git branch: %v (defaulting to \"unknown\")", err)
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// selectDatabase picks the database URL based on the branch name.
// "main" uses DATABASE_URL_PROD; everything else uses DATABASE_URL_TEST.
func selectDatabase(branch string) (url, target string) {
	prodURL := os.Getenv("DATABASE_URL_PROD")
	testURL := os.Getenv("DATABASE_URL_TEST")

	if prodURL == "" {
		log.Fatal("DATABASE_URL_PROD is required")
	}
	if testURL == "" {
		log.Fatal("DATABASE_URL_TEST is required")
	}

	if branch == "main" {
		return prodURL, "prod"
	}
	return testURL, "test"
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
