
package main

import (
	"log"
	"net/http"

	_ "github.com/psyark/yre-rental/apihandler"
)

func main() {
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
