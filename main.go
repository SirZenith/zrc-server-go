package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/godror/godror"
)

// APIRoot is leading path of all request URL
var APIRoot = "/coffee"

// APIVer is version number appear in all request URL
var APIVer = "12"

// HandlerMap recording request URLs and handler corresponding
var HandlerMap = map[string]func(w http.ResponseWriter, r *http.Request){}

// InsideHandler recording internal func inside info getting purpose handler
var InsideHandler = map[string]func(userID int) (ToJSON, error){}
var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("godror", "ARCAEA/ARCAEA@localhost:1521/xe")
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Println("Error while connecting to database.")
		log.Fatal(err)
	}
}

func main() {
	defer db.Close()
	for url, handler := range HandlerMap {
		http.HandleFunc(url, handler)
	}
	fmt.Println("Starting a server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}

	// tojson, err := getMyMapInfo(1)
	// if err != nil {
	// 	log.Print(err)
	// 	return
	// }
	// fmt.Println(tojson.toJSON())
}
