package main

import (
    "fmt"
    "os"	
    "test-api/internal"	
)

func main() {
    args := os.Args[1:]	
    fmt.Println("Educ. Golang API v0.0.32 (260622)\nAuthor: W.Dzieciol (educ. purposes)")
    // init API
    _, err := internal.InitApi()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    // checks args
    for _, arg := range args {
        if arg == "--run-custom-test" {
            if err := doCustomTest(); err != nil {
                os.Exit(1)
            }
            os.Exit(0)
        }
    }
    if !internal.Listen() {
        fmt.Println("Something went wrong. Check logs and configuration.")
        os.Exit(1)
    }
    fmt.Println("Graceful shutdown completed.")
    os.Exit(0)
}



