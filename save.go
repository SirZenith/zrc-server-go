package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/albrow/forms"
)

// DataAndChecksumKeys are keys must be included in uploaded backup date
var DataAndChecksumKeys []string

func init() {

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
	err = db.QueryRow(sqlStmtReadBackupData, userID).Scan(&data)
	if err != nil {
		log.Printf("%s: Error occured while querying table DATA_BACKUP for downloading data: %s\n", r.URL.Path, err)
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
		log.Printf("%s: Error occured while parsing form: %s\n", r.URL.Path, err)
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
	}
	results := []string{}
	for _, key := range DataAndChecksumKeys {
		content := data.Get(key + "_data")
		checksum := data.Get(key + "_checksum")
		sum := fmt.Sprintf("%x", md5.Sum([]byte(content)))
		if string(sum) != checksum {
			log.Printf("%s: Checksum check failed for key `%s`", r.URL.Path, key)
			return
		}
		results = append(results, fmt.Sprintf(`"%s":%s`, key, content))
	}
	result := strings.Join(results, ",")
	_, err = db.Exec(sqlStmtWriteBackupDate, result, userID)
	if err != nil {
		log.Printf("%s: Error occured while writing backup data: %s\n", r.URL.Path, err)
	}
	fmt.Fprintf(w, `{"success":true,"value":{"user_id":%d}}`, userID)
}
