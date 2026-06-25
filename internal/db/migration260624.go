package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func usersUp(ctx context.Context, m *Migration) error {
    sql := `%s
    CREATE TABLE IF NOT EXISTS public.users
    (
        id integer NOT NULL DEFAULT nextval('users_id_seq'::regclass),
        first_name character varying(255) COLLATE pg_catalog."default" NOT NULL,
        last_name character varying(255) COLLATE pg_catalog."default" NOT NULL,
        email character varying(255) COLLATE pg_catalog."default",
        nickname character varying(255) COLLATE pg_catalog."default" NOT NULL,
        passphrase character varying(255) COLLATE pg_catalog."default" NOT NULL,
        is_verified boolean NOT NULL DEFAULT false,
        last_seen bigint,
        is_active boolean DEFAULT false,
        created_at bigint NOT NULL,
        updated_at bigint,
        CONSTRAINT pkusers PRIMARY KEY (id),
        CONSTRAINT unq_email UNIQUE (email)
    )
    TABLESPACE pg_default;
    ALTER TABLE IF EXISTS public.users
        OWNER to %s;`
    sql = fmt.Sprintf(sql, m.BuildCreateSequenceSql("public.users_id_seq"), m.DbOwnerName)
    return m.Migrate(ctx, "users", sql)
}

func rolesUp(ctx context.Context, m *Migration) error {
    sql := `%s
    CREATE TABLE IF NOT EXISTS public.roles
    (
        id integer NOT NULL DEFAULT nextval('roles_id_seq'::regclass),
        name character varying(200) COLLATE pg_catalog."default" NOT NULL,
        description character varying(1000) COLLATE pg_catalog."default" DEFAULT NULL::character varying,
        is_admin boolean NOT NULL DEFAULT false,
        is_system boolean NOT NULL DEFAULT false,
        can_login boolean NOT NULL DEFAULT false,
        created_at bigint NOT NULL,
        updated_at bigint,
        CONSTRAINT pkroles PRIMARY KEY (id)
    )
    TABLESPACE pg_default;
    ALTER TABLE IF EXISTS public.roles
        OWNER to %s;`
    sql = fmt.Sprintf(sql, m.BuildCreateSequenceSql("public.roles_id_seq"), m.DbOwnerName)
    return m.Migrate(ctx, "roles", sql)
}

func user2rolesUp(ctx context.Context, m *Migration) error {
    sql := `%s
    CREATE TABLE IF NOT EXISTS public.user2roles
    (
        id integer NOT NULL DEFAULT nextval('user2roles_id_seq'::regclass),
        user_id integer NOT NULL,
        role_id integer NOT NULL,
        created_at bigint NOT NULL,
        updated_at bigint,
        CONSTRAINT pku2r PRIMARY KEY (id),
        CONSTRAINT u2r_unique_combo UNIQUE (user_id, role_id),
        CONSTRAINT "fk_roleId" FOREIGN KEY (role_id)
            REFERENCES public.roles (id) MATCH SIMPLE
            ON UPDATE CASCADE
            ON DELETE CASCADE,
        CONSTRAINT "fk_userId" FOREIGN KEY (user_id)
            REFERENCES public.users (id) MATCH SIMPLE
            ON UPDATE CASCADE
            ON DELETE CASCADE
    )
    TABLESPACE pg_default;
    ALTER TABLE IF EXISTS public.user2roles
        OWNER to %s;`
    sql = fmt.Sprintf(sql, m.BuildCreateSequenceSql("public.user2roles_id_seq"), m.DbOwnerName)
    return m.Migrate(ctx, "user2roles", sql)
}

func tokensUp(ctx context.Context, m *Migration) error {
    sql := `%s
    CREATE TABLE IF NOT EXISTS public.token
    (
        id integer NOT NULL DEFAULT nextval('token_id_seq'::regclass),
        value character varying(128) COLLATE pg_catalog."default" NOT NULL,
        valid_to bigint NOT NULL,
        created_at bigint NOT NULL,
        updated_at bigint,
        user_id integer NOT NULL,
        CONSTRAINT pktoken PRIMARY KEY (id),
        CONSTRAINT token_val_unq UNIQUE (value),
        CONSTRAINT token_userid_fk FOREIGN KEY (user_id)
        REFERENCES public.users (id) MATCH SIMPLE
            ON UPDATE CASCADE
            ON DELETE CASCADE
            NOT VALID
    )
    TABLESPACE pg_default;
    ALTER TABLE IF EXISTS public.token
        OWNER to %s;`    
    sql = fmt.Sprintf(sql, m.BuildCreateSequenceSql("public.token_id_seq"), m.DbOwnerName)
    return m.Migrate(ctx, "token", sql)
}

func RunCoreMigration260624(ctx context.Context, pool *pgxpool.Pool, dbOwner string) error {
    mHandlers := []PGMigrationHandler{usersUp, rolesUp, user2rolesUp, tokensUp}
    // try run migration
    return RunMigration(ctx, pool, dbOwner, mHandlers)
}
