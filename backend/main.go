package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"dayos/auth"
	"dayos/db"
	"dayos/graph"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed db/migrations/*.sql
var migrations embed.FS

func main() {
	appSecret := os.Getenv("APP_SECRET")
	if appSecret == "" {
		log.Fatal("APP_SECRET environment variable is required")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Run migrations
	if err := runMigrations(dbURL); err != nil {
		log.Fatalf("running migrations: %v", err)
	}

	// Initialize connection pool
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	// Initialize sqlc queries and GraphQL server
	queries := db.New(pool)
	cfg := graph.Config{
		Resolvers: &graph.Resolver{
			RoutineStore: queries,
			TaskStore:    queries,
			DayPlanStore: queries,
		},
	}
	cfg.Directives.Validate = graph.ValidateDirective()
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(cfg))

	authMiddleware := auth.RequireAuth(appSecret)

	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		// Dev mode: serve playground, but behind auth
		http.Handle("/", authMiddleware(playground.Handler("DayOS", "/graphql")))
	}
	// Production: no playground — static files would be served here (owned by deployment spec)

	// GraphQL endpoint always requires auth
	http.Handle("/graphql", authMiddleware(srv))

	log.Printf("DayOS server running on :%s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func runMigrations(dbURL string) error {
	source, err := iofs.New(migrations, "db/migrations")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}

	return nil
}
