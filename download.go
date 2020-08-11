package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/albrow/forms"
)

// AuthFileServer is a wrapper for FilerServer handler, add athentication support
type AuthFileServer struct {
	authenticator func(http.ResponseWriter, *http.Request) bool
	fileServer    http.Handler
}

func (f *AuthFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if f.authenticator(w, r) {
		f.fileServer.ServeHTTP(w, r)
	} else {
		http.Error(w, "Authentication Failed", http.StatusForbidden)
	}
}

// DlExpiresTime is duration before a download request expires.
var DlExpiresTime float64

// LastDlListCheck record time of last expired request cleaning
var LastDlListCheck time.Time

func init() {
	duration, _ := time.ParseDuration("15m")
	DlExpiresTime = duration.Seconds()
	LastDlListCheck = time.Now()

	R.Handle(
		path.Join(APIRoot, APIVer, "serve/download/me/song"),
		http.HandlerFunc(songDownloadHandler),
	)
	InsideHandler[path.Join(APIRoot, APIVer, "serve/download/me/song")] = getDownloadList
	R.PathPrefix("/static/songs").Handler(
		fileServerWithAuth(
			http.StripPrefix("/static/songs", http.FileServer(http.Dir("./static/songs"))),
		),
	)
}

func songDownloadHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
	}
	tojson, err := getDownloadList(userID, r)
	container := Container{false, nil, 0}
	if err != nil {
		log.Println(err)
	} else {
		container.Success = true
		container.Value = tojson
		fmt.Fprint(w, container.toJSON())
	}
}

func getDownloadList(userID int, r *http.Request) (ToJSON, error) {
	data, err := forms.Parse(r)
	if err != nil {
		log.Println("Error occured while parsing form(s) fot getting download list")
		return nil, err
	}
	getURL := data.GetBool("url")
	songs := []string{}
	if data.KeyExists("sid") {
		songs = data.Values["sid"]
	}
	container := map[string]*Checksum{}
	err = getPurchaseDL(userID, container, getURL, songs)
	if err != nil {
		return nil, err
	}
	return (*CheckSumContainer)(&container), nil
}

func getPurchaseDL(userID int, container map[string]*Checksum, getURL bool, songs []string) error {
	if len(songs) > 10 {
		log.Println("To much sid request for getting download list")
		return errors.New("Too musch request at a time")
	} else if len(songs) > 0 {
		for i := range songs {
			songs[i] = "'" + songs[i] + "'"
		}
	}
	err := getPurchasedDL(
		userID, "pack_purchase_info pur", "pur.pack_name = song.pack_name",
		container, getURL, songs,
	)
	if err != nil {
		return err
	}
	err = getPurchasedDL(
		userID, "single_purchase_info pur", "pur.song_id = song.song_id",
		container, getURL, songs,
	)
	if err != nil {
		return err
	}
	return nil
}

func getPurchasedDL(userID int, purchaseTable string, condition string, container map[string]*Checksum, getURL bool, songs []string) error {
	stmt := fmt.Sprintf(`select
		song.song_id,
		song.checksum as "Audio Checksum",
		song.remote_dl as "Song DL",
		to_char(difficulty),
		chart_info.checksum as "Chart Checksum",
		chart_info.remote_dl as "Chart DL"
	from
		%s, song, chart_info
	where
		pur.user_id = :1
		and %s
		and song.song_id = chart_info.song_id
		and (song.remote_dl = 't' or chart_info.remote_dl = 't')`, purchaseTable, condition)
	if len(songs) > 0 {
		stmt += fmt.Sprintf("\nand song.song_id in (%s)", strings.Join(songs, ", "))
	}
	rows, err := db.Query(stmt, userID)
	if err != nil {
		log.Printf(
			"Error occured while querying table `%s` for download list.\n",
			purchaseTable,
		)
		return err
	}
	defer rows.Close()

	var (
		songID        string
		audioChecksum string
		isSongDL      string
		difficulty    string
		chartChecksum string
		isChartDL     string
	)
	for rows.Next() {
		rows.Scan(
			&songID, &audioChecksum, &isSongDL,
			&difficulty, &chartChecksum, &isChartDL,
		)
		if isChartDL != "t" {
			continue
		}
		checksum, ok := container[songID]
		if !ok {
			tempMap := map[string]string{}
			if isSongDL == "t" {
				tempMap["checksum"] = audioChecksum
				if getURL {
					requestTime := time.Now().Unix()
					query := fmt.Sprintf(
						"base.ogg?user_id=%d&song_id=%s&time=%d",
						userID, songID, requestTime,
					)
					tempMap["url"] = path.Join(HostName, "/static/songs", songID, query)
					_, err := db.Exec(
						`insert into dl_request(user_id, song_id, request_time) values(:1, :2, :3)`,
						userID, songID, requestTime,
					)
					if err != nil {
						log.Println("Error occured while inserting into table DL_REQUEST")
						return err
					}
				}
			}
			checksum = &Checksum{
				Audio: tempMap,
				Chart: map[string]map[string]string{},
			}
			container[songID] = checksum
		}
		if getURL {
			requestTime := time.Now().Unix()
			query := fmt.Sprintf(
				"%s.aff?user_id=%d&song_id=%s&time=%d",
				difficulty, userID, songID, requestTime,
			)
			checksum.Chart[difficulty] = map[string]string{
				"checksum": chartChecksum,
				"url":      path.Join(HostName, "/static/songs", songID, query),
			}
			_, err := db.Exec(
				`insert into dl_request(user_id, song_id, request_time)  values(:1, :2, :3)`,
				userID, songID, requestTime,
			)
			if err != nil {
				log.Println("Error occured while inserting into table DL_REQUEST")
				return err
			}
		} else {
			checksum.Chart[difficulty] = map[string]string{
				"checksum": chartChecksum,
			}
		}
	}

	if err = rows.Err(); err != nil {
		log.Println("Error occured while reading quiried rows from WORLD_SONG_UNLOCK for download list.")
		return err
	}
	return nil
}

func fileServerWithAuth(fileServer http.Handler) http.Handler {
	return &AuthFileServer{authenAndClean, fileServer}
}

func authenAndClean(w http.ResponseWriter, r *http.Request) bool {
	var now time.Time
	if now = time.Now(); now.Sub(LastDlListCheck).Seconds() > DlExpiresTime {
		_, err := db.Exec(
			`delete from dl_request where requet_time < :1`,
			now.Unix()-int64(DlExpiresTime),
		)
		if err != nil {
			log.Println("Error occured while cleaning download request")
			log.Println(err)
			return false
		}
		LastDlListCheck = now
	}
	data, err := forms.Parse(r)
	if err != nil {
		log.Println(err)
		return false
	}
	val := data.Validator()
	val.Require("user_id")
	val.Require("song_id")
	val.Require("time")
	if val.HasErrors() {
		log.Println("form passed lacks of necessary key")
		for k, v := range val.ErrorMap() {
			fmt.Printf("%s: %s\n", k, v)
		}
		return false
	}
	userID := data.GetInt("user_id")
	songID := data.Get("song_id")
	requestTime, err := strconv.ParseInt(data.Get("time"), 10, 64)
	if err != nil {
		log.Println("Invalid time in request form")
		return false
	} else if now.Unix()-requestTime > int64(DlExpiresTime) {
		return false
	}
	err = db.QueryRow(`select
			user_id
		from
			dl_request
		where
			user_id = :1 and song_id = :2 and request_time = :3`,
		userID, songID, requestTime,
	).Scan(&userID)
	if err != nil {
		log.Println("Erorr occured while looking up request in table DL_REQUEST")
		log.Println(err)
		return false
	}

	_, err = db.Exec(`delete from dl_request
		where
			user_id = :1 and song_id = :2 and request_time = :3`,
		userID, songID, requestTime,
	)
	if err != nil {
		log.Println("Erorr occured while deleting request in table DL_REQUEST")
		log.Println(err)
		return false
	}
	return true
}
