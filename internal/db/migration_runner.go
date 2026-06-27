package db

// =====================================
// Specify all migrations IN VALID ORDER
// Migrate me type: func(context.Context, *pgxpool.Pool, string) error
// =====================================
var Migrations = []MigrateMe{RunCoreMigration260624, RunDomainMigration260624}