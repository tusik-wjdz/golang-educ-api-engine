package db

import (
    "context"
    "fmt"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
)

// specify all migrations IN VALID ORDER
var Migrations = []MigrateMe{RunCoreMigration260624, RunDomainMigration260624}

type MigrateMe func(ctx context.Context, pool *pgxpool.Pool, dbOwner string) error;

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dbOwner string) error {
    // specify all migrations in order
    for _, m := range Migrations {
        err := m(ctx, pool, dbOwner)
        if err != nil {
            log.Println(err)
            return fmt.Errorf(
                "Migration process incompleted. Reason: %w. Check logs and configuration.",
                err,
            )
        }
    }
    return nil
}