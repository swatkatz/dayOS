package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"dayos/auth"
	"dayos/calendar"
	"dayos/cors"
	"dayos/db"
	"dayos/graph"
	"dayos/planner"

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

	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
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

	// Initialize sqlc queries, planner, and GraphQL server
	queries := db.New(pool)
	aiClient := planner.NewAnthropicClient()
	plannerSvc := planner.New(aiClient)

	// Initialize calendar service (optional — works without Google env vars)
	var calendarSvc graph.CalendarService
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
	if googleClientID != "" && googleClientSecret != "" {
		var cache calendar.Cache
		if memcachedURL := os.Getenv("MEMCACHED_URL"); memcachedURL != "" {
			cache = calendar.NewMemcachedCache(memcachedURL)
		} else {
			cache = &calendar.NullCache{}
		}
		calendarSvc = &calendar.Service{
			Store:     queries,
			Cache:     cache,
			API:       calendar.NewGoogleCalendarAPI(),
			Refresher: calendar.NewGoogleTokenRefresher(googleClientID, googleClientSecret),
			Exchanger: calendar.NewGoogleOAuthExchanger(googleClientID, googleClientSecret, googleRedirectURI),
		}
	}

	cfg := graph.Config{
		Resolvers: &graph.Resolver{
			RoutineStore:          queries,
			TaskStore:             queries,
			ContextStore:          queries,
			DayPlanStore:          queries,
			TaskConversationStore: queries,
			Planner:               plannerSvc,
			Calendar:              calendarSvc,
		},
	}
	cfg.Directives.Validate = graph.ValidateDirective()
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(cfg))

	authMiddleware := auth.RequireAuth(appSecret)
	corsMiddleware := cors.AllowFrontend(frontendURL)

	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		// Dev mode: serve playground, but behind auth
		http.Handle("/", authMiddleware(playground.Handler("DayOS", "/graphql")))
	}

	// GraphQL endpoint: CORS wraps auth (preflight OPTIONS has no Authorization header)
	http.Handle("/graphql", corsMiddleware(authMiddleware(srv)))

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
