package graph

import "github.com/jackc/pgx/v5/pgxpool"

type Resolver struct {
	Pool *pgxpool.Pool
}
