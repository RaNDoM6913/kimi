package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type UserPrivateRepo struct {
	pool *pgxpool.Pool
}

func NewUserPrivateRepo(pool *pgxpool.Pool) *UserPrivateRepo {
	return &UserPrivateRepo{pool: pool}
}
