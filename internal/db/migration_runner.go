package db

// =====================================
// Specify all migrations IN VALID ORDER
// MigrateMe type (delegate): func(context.Context, *pgxpool.Pool, string) error
// =====================================
var Migrations = []MigrateMe{RunCoreMigration260624, RunDomainMigration260624}