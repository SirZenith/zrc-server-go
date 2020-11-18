package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
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
}

func songDownloadHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
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
	needURL := data.GetBool("url")
	songs := []string{}
	if data.KeyExists("sid") {
		songs = data.Values["sid"]
	}
	checksums, err := getPurchaseDL(userID, songs, needURL)
	if err != nil {
		return nil, err
	}
	return (*CheckSumContainer)(&checksums), nil
}

func getPurchaseDL(userID int, songs []string, needURL bool) (map[string]*Checksum, error) {
	songIDCondition := ""
	checksums := map[string]*Checksum{}
	if len(songs) > 0 {
		for i := range songs {
			songs[i] = "'" + songs[i] + "'"
		}
		songIDCondition = fmt.Sprintf("and song.song_id in (%s)", strings.Join(songs, ", "))
	}
	if err := getPurchaseFromTable(
		userID, "pack_purchase_info pur", "pur.pack_name = song.pack_name",
		checksums, songIDCondition, needURL,
	); err != nil {
		return nil, err
	}
	if err := getPurchaseFromTable(
		userID, "single_purchase_info pur", "pur.song_id = song.song_id",
		checksums, songIDCondition, needURL,
	); err != nil {
		return nil, err
	}

	return checksums, nil
}

type dlInfo struct {
	songID        string
	audioChecksum string
	songDL        string
	difficulty    string
	chartChecksum string
	chartDL       string
}

func getPurchaseFromTable(userID int, tableName string, condition string, checksums map[string]*Checksum, songIDCondition string, needURL bool) error {
	stmt := fmt.Sprintf(sqlStmtQueryDLInfo, tableName, condition, songIDCondition)
	rows, err := db.Query(stmt, userID)
	if err != nil {
		return fmt.Errorf(
			"Error occured while querying table %v for download list: %w",
			tableName, err,
		)
	}
	defer rows.Close()

	info := new(dlInfo)
	for rows.Next() {
		rows.Scan(
			&info.songID, &info.audioChecksum, &info.songDL,
			&info.difficulty, &info.chartChecksum, &info.chartDL,
		)
		if info.songDL == "t" {
			var item *Checksum = nil
			if item = checksums[info.songID]; item == nil {
				item = new(Checksum)
			}
			item.Audio = map[string]string{"checksum": info.audioChecksum}
			if needURL {
				item.Audio["url"] = path.Join(HostName, fileServerPrefix, info.songID, "base.ogg")
			}
			checksums[info.songID] = item
		}
		if info.chartDL == "t" {
			var item *Checksum = nil
			if item = checksums[info.songID]; item == nil {
				item = new(Checksum)
			}
			item.Chart = map[string]map[string]string{
				info.difficulty: {"checksum": info.chartChecksum},
			}
			if needURL {
				filename := info.difficulty + ".aff"
				item.Chart[info.difficulty]["url"] = path.Join(HostName, fileServerPrefix, info.songID, filename)
			}
			checksums[info.songID] = item
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf(
			"Error occured while reading quiried dl info rows from %v: %w",
			tableName, err,
		)
	}
	return nil
}

func fileServerWithAuth(fileServer http.Handler) http.Handler {
	return &AuthFileServer{authenAndClean, fileServer}
}

func authenAndClean(w http.ResponseWriter, r *http.Request) bool {
	// var now time.Time
	// if now = time.Now(); now.Sub(LastDlListCheck).Seconds() > DlExpiresTime {
	// 	_, err := db.Exec(
	// 		`delete from dl_request where request_time < ?`,
	// 		now.Unix()-int64(DlExpiresTime),
	// 	)
	// 	if err != nil {
	// 		log.Println("Error occured while cleaning download request")
	// 		log.Println(err)
	// 		return false
	// 	}
	// 	LastDlListCheck = now
	// }
	// data, err := forms.Parse(r)
	// if err != nil {
	// 	log.Println(err)
	// 	return false
	// }
	// val := data.Validator()
	// val.Require("user_id")
	// val.Require("song_id")
	// val.Require("time")
	// if val.HasErrors() {
	// 	log.Println("form passed lacks of necessary key")
	// 	for k, v := range val.ErrorMap() {
	// 		fmt.Printf("%s: %s\n", k, v)
	// 	}
	// 	return false
	// }
	// userID := data.GetInt("user_id")
	// songID := data.Get("song_id")
	// requestTime, err := strconv.ParseInt(data.Get("time"), 10, 64)
	// if err != nil {
	// 	log.Println("Invalid time in request form")
	// 	return false
	// } else if now.Unix()-requestTime > int64(DlExpiresTime) {
	// 	return false
	// }
	// err = db.QueryRow(`select
	// 		user_id
	// 	from
	// 		dl_request
	// 	where
	// 		user_id = ?1 and song_id = ?2 and request_time = ?3`,
	// 	userID, songID, requestTime,
	// ).Scan(&userID)
	// if err != nil {
	// 	log.Println("Erorr occured while looking up request in table DL_REQUEST")
	// 	log.Println(err)
	// 	return false
	// }

	// _, err = db.Exec(`delete from dl_request
	// 	where
	// 		user_id = ?1 and song_id = ?2 and request_time = ?3`,
	// 	userID, songID, requestTime,
	// )
	// if err != nil {
	// 	log.Println("Erorr occured while deleting request in table DL_REQUEST")
	// 	log.Println(err)
	// 	return false
	// }
	return true
}
