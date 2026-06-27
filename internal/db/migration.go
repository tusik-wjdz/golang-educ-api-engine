package db

import (
    "context"
    "fmt"
    "log"	
    "github.com/jackc/pgx/v5/pgxpool"
)

type (
    Migration struct {
        pool 			*pgxpool.Pool
        DbOwnerName		string
        lMigrationName  string
    }

    MigrateMe func(ctx context.Context, pool *pgxpool.Pool, dbOwner string) error;

    PGMigrationHandler func(
        ctx context.Context,
        m *Migration,
    ) error
)

// main migration "runner"
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dbOwner string) error {
    // run migrations in order
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
// run single migration
// API db_name (even if not exist) and user_name, host and port MUST BE CONFIGURED before launch this
func RunMigration(
    ctx context.Context,
    pool *pgxpool.Pool,
    user string,
    handlers []PGMigrationHandler,
) error {
    m := Migration{pool: pool, DbOwnerName: user}
    return m.Start(ctx, handlers)
}
// main func. for start migrations
func (m *Migration) Start(ctx context.Context, handlers []PGMigrationHandler) error {
    msg	    := "%d from %d declared steps has been finished. Check logs.\n"
    tNum    := len(handlers)
    counter := 0    
    for _, h := range handlers {
        err     := h(ctx, m)
        name    := m.GetLastMigrationName()
        if err != nil {
            log.Printf("Migration [`%s`] failed. Reason: %s\n", name, err)
            log.Printf(msg, counter, tNum)
            return fmt.Errorf("Migration terminated. See logs.")
        }
        log.Printf("[`%s`] has been migrated.", name)
        counter++
    }
    log.Printf(msg, counter, tNum)
    return nil
}
// small helper for exec single query on DB
func (m *Migration) Migrate(
    ctx context.Context, 
    name string,
    q string,
    args... any,
) error {
    // set last migration name
    m.lMigrationName = name
    // run query
    _, err := m.pool.Exec(ctx, q, args...)
    return err
}
// get last migration name (empty string if it doesn't happen)
func (m *Migration) GetLastMigrationName() string {
    return m.lMigrationName
}
// other helpers
func (m *Migration) BuildCreateSequenceSql(seqName string) string {
    return fmt.Sprintf(`
    CREATE SEQUENCE IF NOT EXISTS %s 
        INCREMENT 1
        START 1
        MINVALUE 1
        MAXVALUE 2147483647
        CACHE 1;
    `, seqName) + "\n"
}