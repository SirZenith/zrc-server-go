package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/kardianos/osext"
	_ "github.com/mattn/go-sqlite3"
)

// Port is port number used by server
var Port string

// HostName is server address
var HostName string

// APIRoot is leading path of all request URL
var APIRoot string

// APIVer is version number appear in all request URL
var APIVer string

// R router used globally
var R = mux.NewRouter()

func init() {
	port := flag.Int("p", 8080, "Port number for server")
	hostFlag := flag.String("h", "127.0.0.1", "Host name for server")
	apiRoot := flag.String("r", "coffee", "Root path for API")
	apiVer := flag.Int("v", 1, "API version for current server")
	flag.Parse()

	Port = fmt.Sprintf("%d", *port)
	HostName = "http://" + *hostFlag
	APIRoot = *apiRoot
	APIVer = fmt.Sprintf("%d", *apiVer)
}

// InsideHandler recording internal func inside info getting purpose handler
var InsideHandler = map[string]func(userID int, r *http.Request) (ToJSON, error){}
var db *sql.DB

func init() {
	var err error
	exePath, _ := osext.ExecutableFolder()
	db, err = sql.Open("sqlite3", path.Join(exePath, "ArcaeaDB.db"))
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
	fmt.Println("Starting a server at port", Port)
	if err := http.ListenAndServe(":"+Port, R); err != nil {
		log.Fatal(err)
	}
}
