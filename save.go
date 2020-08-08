package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/albrow/forms"
)

// DataAndChecksumKeys are keys must be included in uploaded backup date
var DataAndChecksumKeys []string

func init() {
	HandlerMap[path.Join(APIRoot, APIVer, "user/me/save")] = backupHandler
	DataAndChecksumKeys = []string{
		"scores", "clearlamps", "clearedsongs", "unlocklist",
		"installid", "devicemodelname", "story", "version",
	}
}

func backupHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	if r.Method == "GET" {
		err = returnBackup(w, r)
	} else if r.Method == "POST" {
		err = receiveBackup(w, r)
	}
	if err != nil {
		log.Println(err)
	}
}

func returnBackup(w http.ResponseWriter, r *http.Request) error {
	userID, err := strconv.Atoi(r.Header.Get("i"))
	if err != nil {
		log.Println("Failed to read user id from header when downloading backup")
		return err
	}
	var data string
	err = db.QueryRow(
		"select backup_data from data_backup where user_id = :1", userID,
	).Scan(&data)
	if err != nil {
		log.Println("Error occured while querying table DATA_BACKUP for downloading data")
		return err
	} else if data == "" {
		http.Error(w, `{"success":false,"error_code":402}`, http.StatusNotFound)
	} else {
		fmt.Fprintf(w, `{"success":true,"value":{"user_id":%d,%s}}`, userID, data)
	}
	return nil
}

func receiveBackup(w http.ResponseWriter, r *http.Request) error {
	userID, err := strconv.Atoi(r.Header.Get("i"))
	if err != nil {
		log.Println("Failed to read user id from header when uploading backup")
		return err
	}
	data, err := forms.Parse(r)
	if err != nil {
		log.Printf("%s: Error occured while parsing form\n", r.URL.Path)
		return err
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
		return err
	}
	results := []string{}
	for _, key := range DataAndChecksumKeys {
		content := data.Get(key + "_data")
		checksum := data.Get(key + "_checksum")
		sum := fmt.Sprintf("%x", md5.Sum([]byte(content)))
		if string(sum) != checksum {
			return fmt.Errorf(
				"Checksum check failed for key `%s` with checksum: %s",
				key, string(sum),
			)
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
		return err
	}
	fmt.Fprintf(w, `{"success":true,"value":{"user_id":%d}}`, userID)
	return nil
}
