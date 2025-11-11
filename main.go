package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-service/internal/config"
	"user-service/internal/publisher"
	"user-service/internal/repository"
	"user-service/internal/server"
	"user-service/internal/service"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	log "github.com/sirupsen/logrus"

	"github.com/labstack/echo/v4"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	log.SetOutput(os.Stdout)

	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = "info"
	}

	level, err := log.ParseLevel(levelStr)
	if err != nil {
		log.Warnf("Invalid LOG_LEVEL '%s', using InfoLevel", levelStr)
		level = log.InfoLevel
	}

	log.SetLevel(level)
	log.WithField("level", level.String()).Info("Logger initialized")

	if err := godotenv.Load("../.env"); err != nil {
		log.Warn("Could not load .env file.")
	}
	cfg, err := config.Load()
	if err != nil {
		log.WithField("error", err).Fatal("Could not load configuration")
	}
	dbURL := cfg.DB.URL

	log.Info("Starting database migration...")
	m, err := migrate.New("file://db/migrations", dbURL)
	if err != nil {
		log.WithField("error", err).Fatal("Could not create migrate instance")
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.WithField("error", err).Fatal("Could not apply migration")
	}
	log.Info("Database migration finished successfully.")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.WithField("error", err).Fatal("Could not connect to the database")
	}
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.DB.ConnMaxIdleTime)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.WithField("error", err).Fatal("Could not ping the database")
	}

	log.Info("Successfully connected to the PostgreSQL database.")

	// Create repository
	userRepository := repository.NewPostgresUserRepository(db)

	// Create audit publisher
	kafkaBootstrap := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if kafkaBootstrap == "" {
		log.Fatal("FATAL: KAFKA_BOOTSTRAP_SERVERS environment variable is not set")
	}

	auditTopic := os.Getenv("KAFKA_AUDIT_TOPIC")
	if auditTopic == "" {
		auditTopic = "audit_events"
	}

	auditPublisher, err := publisher.NewAuditPublisher(kafkaBootstrap, auditTopic)
	if err != nil {
		log.WithField("error", err).Fatal("Could not create audit Kafka publisher")
	}
	defer auditPublisher.Close()

	auditService := service.NewAuditService(auditPublisher)

	// Create service
	userService := service.NewUserService(userRepository, auditService)

	// Create server
	srv := server.NewServer(userService, db)

	// Create product repositories
	categoryRepository := repository.NewPostgresProductCategoryRepository(db)
	productRepository := repository.NewPostgresProductRepository(db)

	// Create product services
	categoryService := service.NewProductCategoryService(categoryRepository)
	productService := service.NewProductService(productRepository)

	// Create product servers
	categoryServer := server.NewProductCategoryServer(categoryService)
	productServer := server.NewProductServer(productService)

	// Setup Echo
	e := echo.New()

	// Health check
	e.GET("/health", srv.HealthCheck)

	// CRUD endpoints
	api := e.Group("/api")
	users := api.Group("/users")
	users.POST("", srv.CreateUser)
	users.GET("/:id", srv.GetUser)
	users.GET("/email/:email", srv.GetUserByEmail)
	users.PUT("/:id", srv.UpdateUser)
	users.DELETE("/:id", srv.DeleteUser)
	users.GET("", srv.ListUsers)

	// Business logic endpoints
	users.POST("/:id/coins", srv.AddCoins)
	users.POST("/:id/coins/deduct", srv.DeductCoins)
	users.POST("/:id/subscription/activate", srv.ActivateSubscription)
	users.POST("/:id/subscription/renew", srv.RenewSubscription)
	users.GET("/:id/access", srv.HasAccess)

	// Catalog endpoints
	catalog := api.Group("/catalog")

	// Categories
	categories := catalog.Group("/categories")
	categories.GET("", categoryServer.ListCategories)
	categories.GET("/:id", categoryServer.GetCategoryByID)
	categories.GET("/slug/:slug", categoryServer.GetCategoryBySlug)
	categories.POST("", categoryServer.CreateCategory)
	categories.PUT("/:id", categoryServer.UpdateCategory)
	categories.DELETE("/:id", categoryServer.DeleteCategory)

	// Products
	products := catalog.Group("/products")
	products.GET("", productServer.ListProducts)
	products.GET("/:id", productServer.GetProductByID)
	products.GET("/slug/:slug", productServer.GetProductBySlug)
	products.POST("", productServer.CreateProduct)
	products.PUT("/:id", productServer.UpdateProduct)
	products.DELETE("/:id", productServer.DeleteProduct)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.WithField("port", port).Info("User service is starting with Echo")

	// Start server in goroutine
	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.WithField("error", err).Fatal("Echo server failed to start")
		}
	}()

	// Setup graceful shutdown
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	log.Info("User service started. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-sigchan
	log.Info("Shutting down user service...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.WithField("error", err).Error("Error shutting down server")
	}

	// Close resources explicitly
	if err := db.Close(); err != nil {
		log.WithError(err).Error("Error closing database")
	}

	log.Info("User service stopped")
}
