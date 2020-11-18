package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"unsafe"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

//#include "./theos_code/main.h"
import "C"

var (
	// Port is port number used by server
	Port string
	// HostName is server address
	HostName string
	// APIRoot is leading path of all request URL
	APIRoot           = "/glad-you-came"
	scoreCardTemplate string
	scorePageTemplate string
	// InsideHandler recording internal func inside info getting purpose handler
	InsideHandler = map[string]func(userID int, r *http.Request) (ToJSON, error){}
	db            *sql.DB
)

const fileServerPrefix = "/static/songs"

var scoreCardTemplatePath = path.Join("static", "score_lookup", "card_template.html")
var scorePageTemplatePath = path.Join("static", "score_lookup", "page_template.html")

func setRouting(apiroot string) *mux.Router {
	router := mux.NewRouter()

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fileServerPath := path.Join(pwd, fileServerPrefix)
	router.PathPrefix(fileServerPrefix).Handler(
		fileServerWithAuth(
			http.StripPrefix(fileServerPrefix, http.FileServer(http.Dir(fileServerPath))),
		),
	)

	router.Path(path.Join("/score", "b30", "{id:[0-9]{9}}")).Methods("GET").Handler(http.HandlerFunc(scoreLookupHandler))

	s := router.PathPrefix(apiroot).Subrouter()

	s.Path("/auth/login").Methods("POST").HandlerFunc(loginHandler)

	s.Path("/compose/aggregate").Methods("GET").Handler(http.HandlerFunc(aggregateHandler))

	s.Path("/user/me/character").Methods("POST").Handler(http.HandlerFunc(changeCharacter))
	s.PathPrefix("/user/me/characters/{partID}/toggle_uncap").Methods("POST").Handler(http.HandlerFunc(toggleUncap))

	s.Path("/game/info").Methods("GET").Handler(http.HandlerFunc(gameInfoHandler))
	InsideHandler["/game/info"] = getGameInfo

	s.Path("/purchase/bundle/pack").Methods("GET").Handler(http.HandlerFunc(packInfoHandler))
	InsideHandler["/purchase/bundle/pack"] = getPackInfo

	s.Path("/present/me").Methods("GET").Handler(http.HandlerFunc(presentMeHandler))
	InsideHandler["/present/me"] = presentMe

	s.Path("/user/me/save").Methods("GET").Handler(http.HandlerFunc(returnBackup))
	s.Path("/user/me/save").Methods("POST").Handler(http.HandlerFunc(receiveBackup))

	s.Path("/score/token").Methods("GET").Handler(http.HandlerFunc(scoreTokenHandler))
	s.Path("/score/song").Methods("POST").Handler(http.HandlerFunc(scoreUploadHandler))

	s.Path("/user/me").Methods("GET").Handler(http.HandlerFunc(userInfoHandler))

	s.PathPrefix("/user/me/setting").Methods("POST").Handler(http.HandlerFunc(userSettingHandler))
	InsideHandler["/user/me"] = getUserInfo

	s.Path("/world/map/me").Methods("GET").Handler(http.HandlerFunc(myMapInfoHandler))
	InsideHandler["/world/map/me"] = getMyMapInfo

	s.Path("/serve/download/me/song").Methods("GET").Handler(http.HandlerFunc(songDownloadHandler))
	InsideHandler["/serve/download/me/song"] = getDownloadList

	return router
}

func startUp(args []string) {
	commandLine := flag.NewFlagSet(args[0], flag.ExitOnError)

	port := commandLine.Int("port", 8080, "Port number for server.")
	hostFlag := commandLine.String("host", "127.0.0.1", "Host name for server.")
	docuemntRoot := commandLine.String("root", "", "Root path of server documents.")
	dbFile := commandLine.String("db", "ZrcaeaDB.db", "sqlite DB file to use.")

	commandLine.Parse(args[1:])

	connectToDB(*dbFile)

	Port = fmt.Sprintf("%d", *port)
	HostName = fmt.Sprintf("http://%s:%s", *hostFlag, Port)
	fmt.Printf("Root URL: %s%s\n", HostName, APIRoot)

	if *docuemntRoot != "" {
		fmt.Println("Documents Root:", *docuemntRoot)
		os.Chdir(*docuemntRoot)
	}

	readTemplate()
	if scoreCardTemplate == "" || scorePageTemplate == "" {
		log.Fatal("Can't read webpage templates.")
	}
}

func connectToDB(dbFile string) {
	var err error
	dbFile, err = filepath.Abs(dbFile)
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

func readTemplate() {
	content, err := ioutil.ReadFile(scoreCardTemplatePath)
	if err != nil {
		log.Fatal(err)
	}
	scoreCardTemplate = string(content)

	content, err = ioutil.ReadFile(scorePageTemplatePath)
	if err != nil {
		log.Fatal(err)
	}
	scorePageTemplate = string(content)
}

func goStrings(argc C.int, argv **C.char) []string {
	length := int(argc)
	tmpSlice := (*[1 << 9]*C.char)(unsafe.Pointer(argv))[:length:length]
	goStrings := make([]string, length)
	for i, s := range tmpSlice {
		goStrings[i] = C.GoString(s)
	}
	return goStrings
}

//ExportMainObjectiveC main function bridge for objective c
//export ExportMainObjectiveC
func ExportMainObjectiveC(argc C.int, argv, envp **C.char) C.int {
	//convert args from iOS args to golang's os.Args
	args := goStrings(argc, argv)
	startUp(args)
	defer db.Close()

	fmt.Println("Starting a server at port", Port)
	router := setRouting(APIRoot)
	if err := http.ListenAndServe(":"+Port, router); err != nil {
		log.Fatal(err)
	}
	return 0
}

func main() {}
