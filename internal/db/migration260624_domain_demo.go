package db

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
)

func weatherStationsUp(ctx context.Context, m *Migration) error {
    sql := `%s
    CREATE TABLE IF NOT EXISTS public.dm_weather_stations
    (
        id integer NOT NULL DEFAULT nextval('dm_weather_stations_id_seq'::regclass),
        city character varying(1000) COLLATE pg_catalog."default" NOT NULL,
        temperature real,
        humidity smallint,
        wind real,
        wind_gust real,
        sigw character varying(200) COLLATE pg_catalog."default" NOT NULL DEFAULT 'NO_DATA'::character varying,
        precipitation real,
        pressure smallint,
        low_clouds_coverage smallint,
        mid_clouds_coverage smallint,
        high_clouds_coverage smallint,
        created_at bigint NOT NULL,
        updated_at bigint,
        CONSTRAINT pk_dm_weatherstations PRIMARY KEY (id),
        CONSTRAINT chk_high_clouds CHECK (high_clouds_coverage >= 0 AND high_clouds_coverage <= 100) NOT VALID,
        CONSTRAINT chk_humidity CHECK (humidity >= 0 AND humidity <= 100),
        CONSTRAINT chk_low_clouds CHECK (low_clouds_coverage >= 0 AND low_clouds_coverage <= 100) NOT VALID,
        CONSTRAINT chk_mid_clouds CHECK (mid_clouds_coverage >= 0 AND mid_clouds_coverage <= 100) NOT VALID
    )
    TABLESPACE pg_default;
    ALTER TABLE IF EXISTS public.dm_weather_stations
        OWNER to %s;`
    // prepare final sql
    sql = fmt.Sprintf(sql, m.BuildCreateSequenceSql("public.dm_weather_stations_id_seq"), m.DbOwnerName)
    return m.Migrate(ctx, "dm_weather_stations", sql)
}

func productsUp(ctx context.Context, m *Migration) error {    
    sql := `%s
    CREATE TABLE IF NOT EXISTS public.dm_products
    (
        id integer NOT NULL DEFAULT nextval('dm_products_id_seq'::regclass),
        name character varying(500) COLLATE pg_catalog."default" NOT NULL,
        price bigint NOT NULL DEFAULT 0,
        qty integer NOT NULL DEFAULT 0,
        description character varying(5000) COLLATE pg_catalog."default" DEFAULT NULL::character varying,
        color character varying(255) COLLATE pg_catalog."default" DEFAULT NULL::character varying,
        created_at bigint NOT NULL,
        updated_at bigint,
        created_by integer NOT NULL,
        updated_by integer,
        checksum character varying(255) COLLATE pg_catalog."default" DEFAULT NULL::character varying,
        CONSTRAINT pk_products PRIMARY KEY (id),
        CONSTRAINT checksum_unq UNIQUE (checksum),
        CONSTRAINT fk_products_creator FOREIGN KEY (created_by)
        REFERENCES public.users (id) MATCH SIMPLE
            ON UPDATE CASCADE
            ON DELETE RESTRICT,
        CONSTRAINT fk_products_updater FOREIGN KEY (updated_by)
            REFERENCES public.users (id) MATCH SIMPLE
            ON UPDATE CASCADE
            ON DELETE NO ACTION
    )
    TABLESPACE pg_default;
    ALTER TABLE IF EXISTS public.dm_products
        OWNER to %s;`
    // prepare final sql
    sql = fmt.Sprintf(sql, m.BuildCreateSequenceSql("public.dm_products_id_seq"), m.DbOwnerName)
    return m.Migrate(ctx, "dm_products", sql)
}

// just a demo
func RunDomainMigration260624(ctx context.Context, pool *pgxpool.Pool, dbOwner string) error {
    mHandlers := []PGMigrationHandler{weatherStationsUp, productsUp}
    // try run migration
    return RunMigration(ctx, pool, dbOwner, mHandlers)
}
