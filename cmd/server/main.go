package main

import (
	"log"
	"never-price-match-server/internal/app"
)

func main() {
	if err := app.RunFull(); err != nil {
		log.Fatal(err)
	}
}
