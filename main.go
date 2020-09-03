package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/kardianos/osext"
	_ "github.com/mattn/go-sqlite3"
)

// HostName is server address
var HostName = "http://192.168.124.2:8080"

// APIRoot is leading path of all request URL
var APIRoot = "/coffee"

// APIVer is version number appear in all request URL
var APIVer = "12"

// R router used globally
var R = mux.NewRouter()

// InsideHandler recording internal func inside info getting purpose handler
var InsideHandler = map[string]func(userID int, r *http.Request) (ToJSON, error){}
var db *sql.DB

func init() {
	var err error
	exePath, _ := osext.ExecutableFolder()
	db, err = sql.Open("sqlite3", path.Join(exePath, "db", "ArcaeaDB.db"))
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
	fmt.Println("Starting a server at port 8080")
	if err := http.ListenAndServe(":8080", R); err != nil {
		log.Fatal(err)
	}
}
