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
	"dayos/tz"

	clerk "github.com/clerk/clerk-sdk-go/v2"

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
	clerkSecretKey := os.Getenv("CLERK_SECRET_KEY")
	if clerkSecretKey == "" {
		log.Fatal("CLERK_SECRET_KEY environment variable is required")
	}
	clerk.SetKey(clerkSecretKey)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
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

	// Initialize sqlc queries and user-scoped wrapper
	queries := db.New(pool)
	scoped := db.NewScopedQueries(queries)

	// Initialize planner with usage cap decorator
	aiClient := planner.NewAnthropicClient()
	basePlanner := planner.New(aiClient)
	plannerSvc := planner.NewWithUserCap(basePlanner, queries, os.Getenv("ANTHROPIC_API_KEY"))

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
			Store:     scoped,
			Cache:     cache,
			API:       calendar.NewGoogleCalendarAPI(),
			Refresher: calendar.NewGoogleTokenRefresher(googleClientID, googleClientSecret),
			Exchanger: calendar.NewGoogleOAuthExchanger(googleClientID, googleClientSecret, googleRedirectURI),
		}
	}

	cfg := graph.Config{
		Resolvers: &graph.Resolver{
			RoutineStore:          scoped,
			TaskStore:             scoped,
			ContextStore:          scoped,
			DayPlanStore:          scoped,
			TaskConversationStore: scoped,
			Planner:               plannerSvc,
			Calendar:              calendarSvc,
		},
	}
	cfg.Directives.Validate = graph.ValidateDirective()
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(cfg))

	authMiddleware := auth.RequireClerk(queries)
	corsMiddleware := cors.AllowFrontend(frontendURL)

	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		// Dev mode: serve playground, but behind auth
		http.Handle("/", authMiddleware(playground.Handler("DayOS", "/graphql")))
	}

	// GraphQL endpoint: CORS wraps auth (preflight OPTIONS has no Authorization header)
	http.Handle("/graphql", corsMiddleware(authMiddleware(tz.Middleware(srv))))

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
