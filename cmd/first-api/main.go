package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"test-api/internal"
	db "test-api/internal/db"
)

func main() {    
    args := os.Args[1:]    
    fmt.Println("Educ. Golang API v0.0.35 (260708)\nAuthor: W.Dzieciol (educ. purposes)")
    // checks args
    // todo: add dedicated CLI commands parser
    for _, arg := range args {
        if arg == "--run-custom-test" {
            initApiLite()
            if err := doCustomTest(); err != nil {
                log.Panicln(err)
            }
            os.Exit(0)
        }
        if arg == "--db:migrate" {
            initApiLite()
            // core migrations
            dbUser, exists := internal.GetEnv().GetConfig().Db["db_user"]
            if !exists || dbUser == "" {
                log.Println("Field `db_user` not exists or empty in config file (section: `DB`).")
                os.Exit(1)
            }
            ctx     := context.Background()
            pool    := internal.GetEnv().GetConnPool()
            // run all defined (and existing) migrations
            if err := db.RunMigrations(ctx, pool, dbUser); err != nil {
                log.Println(err); os.Exit(1)
            }
            // stop execution
            os.Exit(0)
        }
    }

    // init API (standard mode)
    _, err := internal.InitApi(false)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    if !internal.Listen() {
        fmt.Println("Something went wrong. Check logs and configuration.")
        os.Exit(1)
    }
    fmt.Println("Graceful shutdown completed.")
    os.Exit(0)
}

// small helper for init API in lite version for configure env. / maintenance tasks
func initApiLite() *internal.ApiEnvironment {
    apiEnv, err := internal.InitApi(true)
    if err != nil {
        // init failed - abnormal termination
        fmt.Println(err); os.Exit(1)
    }
    return apiEnv
}