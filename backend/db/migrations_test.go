package db_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var pool *pgxpool.Pool
var databaseURL string

func TestMain(m *testing.M) {
	dockerPool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("connecting to docker: %v", err)
	}

	resource, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15-alpine",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=dayos_test",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("starting postgres container: %v", err)
	}

	hostPort := resource.GetHostPort("5432/tcp")
	databaseURL = fmt.Sprintf("postgres://postgres:postgres@%s/dayos_test?sslmode=disable", hostPort)

	// Wait for postgres to be ready
	if err := dockerPool.Retry(func() error {
		var retryErr error
		pool, retryErr = pgxpool.New(context.Background(), databaseURL)
		if retryErr != nil {
			return retryErr
		}
		return pool.Ping(context.Background())
	}); err != nil {
		log.Fatalf("waiting for postgres: %v", err)
	}

	// Run migrations
	mig, err := migrate.New("file://../db/migrations", databaseURL)
	if err != nil {
		log.Fatalf("creating migrator: %v", err)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("running migrations: %v", err)
	}

	code := m.Run()

	pool.Close()
	if err := dockerPool.Purge(resource); err != nil {
		log.Fatalf("purging docker resource: %v", err)
	}

	os.Exit(code)
}

func TestAllTablesAcceptInserts(t *testing.T) {
	ctx := context.Background()

	// Insert into each table in dependency order
	inserts := []struct {
		name  string
		query string
	}{
		{"routines", `INSERT INTO routines (title, category, frequency) VALUES ('Morning run', 'exercise', 'daily') RETURNING id`},
		{"tasks", `INSERT INTO tasks (title, category, priority) VALUES ('Buy groceries', 'admin', 'medium') RETURNING id`},
		{"context_entries", `INSERT INTO context_entries (category, key, value) VALUES ('test', 'test_key', 'test_value') RETURNING id`},
		{"day_plans", `INSERT INTO day_plans (plan_date, blocks) VALUES ('2099-01-01', '[]'::jsonb) RETURNING id`},
		{"plan_messages", `INSERT INTO plan_messages (plan_id, role, content) VALUES ((SELECT id FROM day_plans LIMIT 1), 'user', 'hello') RETURNING id`},
		{"task_conversations", `INSERT INTO task_conversations (parent_task_id) VALUES ((SELECT id FROM tasks LIMIT 1)) RETURNING id`},
		{"task_messages", `INSERT INTO task_messages (conversation_id, role, content) VALUES ((SELECT id FROM task_conversations LIMIT 1), 'user', 'hello') RETURNING id`},
	}

	for _, tc := range inserts {
		t.Run(tc.name, func(t *testing.T) {
			var id string
			err := pool.QueryRow(ctx, tc.query).Scan(&id)
			if err != nil {
				t.Fatalf("insert into %s failed: %v", tc.name, err)
			}
			if id == "" {
				t.Fatalf("insert into %s returned empty id", tc.name)
			}
		})
	}

	// Cleanup test-inserted rows (leave seed data intact)
	_, _ = pool.Exec(ctx, "DELETE FROM task_messages WHERE content = 'hello'")
	_, _ = pool.Exec(ctx, "DELETE FROM task_conversations")
	_, _ = pool.Exec(ctx, "DELETE FROM plan_messages WHERE content = 'hello'")
	_, _ = pool.Exec(ctx, "DELETE FROM day_plans WHERE plan_date = '2099-01-01'")
	_, _ = pool.Exec(ctx, "DELETE FROM tasks WHERE title = 'Buy groceries'")
	_, _ = pool.Exec(ctx, "DELETE FROM routines WHERE title = 'Morning run'")
	_, _ = pool.Exec(ctx, "DELETE FROM context_entries WHERE key = 'test_key'")
}

func TestSeedContextEntries(t *testing.T) {
	ctx := context.Background()

	rows, err := pool.Query(ctx, "SELECT category, key, value FROM context_entries WHERE key IN ('baby', 'family', 'work_window', 'location', 'energy', 'dinner_prep', 'evening_window', 'kitchen', 'planning_style') ORDER BY key")
	if err != nil {
		t.Fatalf("querying context_entries: %v", err)
	}
	defer rows.Close()

	type entry struct {
		category, key, value string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.category, &e.key, &e.value); err != nil {
			t.Fatalf("scanning row: %v", err)
		}
		entries = append(entries, e)
	}

	if len(entries) != 9 {
		t.Fatalf("expected 9 seed entries, got %d", len(entries))
	}

	// Verify a few specific entries
	expected := map[string]string{
		"baby":           "life",
		"work_window":    "constraints",
		"kitchen":        "equipment",
		"planning_style": "preferences",
	}
	entryMap := make(map[string]string)
	for _, e := range entries {
		entryMap[e.key] = e.category
	}
	for key, wantCat := range expected {
		if gotCat, ok := entryMap[key]; !ok {
			t.Errorf("missing seed entry with key %q", key)
		} else if gotCat != wantCat {
			t.Errorf("entry %q: expected category %q, got %q", key, wantCat, gotCat)
		}
	}
}

func TestCascadeDeleteParentTask(t *testing.T) {
	ctx := context.Background()

	// Create parent task
	var parentID string
	err := pool.QueryRow(ctx, `INSERT INTO tasks (title, category, priority) VALUES ('Parent', 'admin', 'high') RETURNING id`).Scan(&parentID)
	if err != nil {
		t.Fatalf("inserting parent: %v", err)
	}

	// Create subtasks
	for i := range 3 {
		_, err := pool.Exec(ctx, `INSERT INTO tasks (title, category, priority, parent_id) VALUES ($1, 'admin', 'low', $2)`,
			fmt.Sprintf("Child %d", i), parentID)
		if err != nil {
			t.Fatalf("inserting child %d: %v", i, err)
		}
	}

	// Verify subtasks exist
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM tasks WHERE parent_id = $1", parentID).Scan(&count); err != nil {
		t.Fatalf("counting subtasks: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 subtasks, got %d", count)
	}

	// Delete parent
	_, err = pool.Exec(ctx, "DELETE FROM tasks WHERE id = $1", parentID)
	if err != nil {
		t.Fatalf("deleting parent: %v", err)
	}

	// Verify subtasks are gone
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM tasks WHERE parent_id = $1", parentID).Scan(&count); err != nil {
		t.Fatalf("counting subtasks after delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 subtasks after cascade delete, got %d", count)
	}
}

func TestSetNullOnRoutineDelete(t *testing.T) {
	ctx := context.Background()

	// Create routine
	var routineID string
	err := pool.QueryRow(ctx, `INSERT INTO routines (title, category, frequency) VALUES ('Yoga', 'exercise', 'daily') RETURNING id`).Scan(&routineID)
	if err != nil {
		t.Fatalf("inserting routine: %v", err)
	}

	// Create task linked to routine
	var taskID string
	err = pool.QueryRow(ctx, `INSERT INTO tasks (title, category, priority, is_routine, routine_id) VALUES ('Do yoga', 'exercise', 'medium', true, $1) RETURNING id`, routineID).Scan(&taskID)
	if err != nil {
		t.Fatalf("inserting task: %v", err)
	}

	// Delete routine
	_, err = pool.Exec(ctx, "DELETE FROM routines WHERE id = $1", routineID)
	if err != nil {
		t.Fatalf("deleting routine: %v", err)
	}

	// Verify task still exists with NULL routine_id
	var routineIDPtr *string
	err = pool.QueryRow(ctx, "SELECT routine_id FROM tasks WHERE id = $1", taskID).Scan(&routineIDPtr)
	if err != nil {
		t.Fatalf("querying task: %v", err)
	}
	if routineIDPtr != nil {
		t.Fatalf("expected routine_id to be NULL, got %v", *routineIDPtr)
	}

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM tasks WHERE id = $1", taskID)
}

func TestUniquePlanDate(t *testing.T) {
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO day_plans (plan_date, blocks) VALUES ('2026-03-05', '[]'::jsonb)`)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = pool.Exec(ctx, `INSERT INTO day_plans (plan_date, blocks) VALUES ('2026-03-05', '[]'::jsonb)`)
	if err == nil {
		t.Fatal("expected unique constraint violation, got nil")
	}

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM day_plans WHERE plan_date = '2026-03-05'")
}
