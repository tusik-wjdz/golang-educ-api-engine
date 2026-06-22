package main

import (
	"fmt"
	"os"	
	"test-api/internal"	
)

func main() {
	// testMe()
	// return
	fmt.Println("Educ. Golang API v0.0.3 (260622)\nAuthor: W.Dzieciol (educ. purposes)")
	_, err := internal.InitApi()
	if (err != nil) {
		println(err)
		return
	}
	if ! internal.Listen() {
		fmt.Println("Something went wrong. Check logs and configuration.")
		os.Exit(1)		
	}
	fmt.Println("Graceful shutdown completed.")
	os.Exit(0)
}



