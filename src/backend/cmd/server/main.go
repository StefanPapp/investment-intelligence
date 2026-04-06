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
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/service"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
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
	log.Println("Connected to database")

	// Run migrations
	runMigrations(db)

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
