
package main

import (
	"log"
	"net/http"

	_ "github.com/psyark/yre-rental/apihandler"
)

// StartServer はサーバーを開始します
func StartServer() {
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
