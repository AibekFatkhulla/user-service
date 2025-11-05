package main

import (
	"database/sql"
	"net/http"
	"os"

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
	log.SetLevel(log.DebugLevel)

	if err := godotenv.Load(); err != nil {
		log.Warn("Could not load .env file.")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("FATAL: DATABASE_URL environment variable is not set")
	}

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
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.WithField("error", err).Fatal("Could not ping the database")
	}
	log.Info("Successfully connected to the PostgreSQL database.")

	// Create repository
	userRepository := repository.NewPostgresUserRepository(db)

	// Create service
	userService := service.NewUserService(userRepository)

	// Create server
	srv := server.NewServer(userService, db)

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.WithField("port", port).Info("User service is starting with Echo")

	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		log.WithField("error", err).Fatal("Echo server failed to start")
	}
}
