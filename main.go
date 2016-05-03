package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cloudfoundry-community/buddy-broker/buddy"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("buddy-broker")
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	buddyAPI := buddy.New(logger)
	http.Handle("/", buddyAPI)
	logger.Fatal("http-listen", http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil))
}
