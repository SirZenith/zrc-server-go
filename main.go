package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

var (
	// Port is port number used by server
	Port string
	// HostName is server address
	HostName string
	// APIRoot is leading path of all request URL
	APIRoot = "/glad-you-came"
	// R router used globally
	R = mux.NewRouter()
	// InsideHandler recording internal func inside info getting purpose handler
	InsideHandler = map[string]func(userID int, r *http.Request) (ToJSON, error){}
	db            *sql.DB
)

func init() {
	port := flag.Int("port", 8080, "Port number for server")
	hostFlag := flag.String("host", "127.0.0.1:8080", "Host name for server")
	docuemntRoot := flag.String("root", "", "Root path of server documents.")
	dbFile := flag.String("db", "", "sqlite DB file to use.")
	flag.Parse()

	connectToDB(*dbFile)

	Port = fmt.Sprintf("%d", *port)
	HostName = "http://" + *hostFlag
	fmt.Printf("%s%s\n", HostName, APIRoot)

	if *docuemntRoot != "" {
		fmt.Println("Document Root:", *docuemntRoot)
		os.Chdir(*docuemntRoot)
	}
}

func connectToDB(dbFile string) {
	dbFile, err := filepath.Abs(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("sqlite3", dbFile)
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
