package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/albrow/forms"
)

// DataAndChecksumKeys are keys must be included in uploaded backup date
var DataAndChecksumKeys []string

func init() {
	R.Path(path.Join(APIRoot, APIVer, "user/me/save")).Methods("GET").Handler(
		http.HandlerFunc(returnBackup),
	)
	R.Path(path.Join(APIRoot, APIVer, "user/me/save")).Methods("POST").Handler(
		http.HandlerFunc(receiveBackup),
	)
	DataAndChecksumKeys = []string{
		"scores", "clearlamps", "clearedsongs", "unlocklist",
		"installid", "devicemodelname", "story", "version",
	}
}

func returnBackup(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
	}
	var data string
	err = db.QueryRow(
		"select backup_data from data_backup where user_id = :1", userID,
	).Scan(&data)
	if err != nil {
		log.Println("Error occured while querying table DATA_BACKUP for downloading data")
		log.Println(err)
	} else if data == "" {
		http.Error(w, `{"success":false,"error_code":402}`, http.StatusNotFound)
	} else {
		fmt.Fprintf(w, `{"success":true,"value":{"user_id":%d,%s}}`, userID, data)
	}
}

func receiveBackup(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
	}
	data, err := forms.Parse(r)
	if err != nil {
		log.Printf("%s: Error occured while parsing form\n", r.URL.Path)
		log.Println(err)
	}
	val := data.Validator()
	for _, key := range DataAndChecksumKeys {
		val.Require(key + "_data")
		val.Require(key + "_checksum")
	}
	if val.HasErrors() {
		log.Printf("%s: Data uploaded lacks of necessary key.\n", r.URL.Path)
		for k, v := range val.ErrorMap() {
			log.Printf("%s: %s", k, v)
		}
		log.Println(err)
	}
	results := []string{}
	for _, key := range DataAndChecksumKeys {
		content := data.Get(key + "_data")
		checksum := data.Get(key + "_checksum")
		sum := fmt.Sprintf("%x", md5.Sum([]byte(content)))
		if string(sum) != checksum {
			log.Printf(
				"Checksum check failed for key `%s` with checksum: %s",
				key, string(sum),
			)
			return
		}
		results = append(results, fmt.Sprintf(`"%s":%s`, key, content))
	}
	result := strings.Join(results, ",")
	_, err = db.Exec(
		"update data_backup set backup_data = :1 where user_id = :2",
		result, userID,
	)
	if err != nil {
		log.Println("Error occured while updating table DATA_BACKUP")
		log.Println(err)
	}
	fmt.Fprintf(w, `{"success":true,"value":{"user_id":%d}}`, userID)
}
