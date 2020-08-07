package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
)

func init() {
	HandlerMap[path.Join(APIRoot, APIVer, "serve/download/me/song")] = songDownloadHandler
	InsideHandler[path.Join(APIRoot, APIVer, "serve/download/me/song")] = getDownloadList
}

func songDownloadHandler(w http.ResponseWriter, r *http.Request) {
	userID := 1
	tojson, err := getDownloadList(userID)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Fprint(w, tojson.toJSON())
	}
}

func getDownloadList(userID int) (ToJSON, error) {
	container := map[string]*Checksum{}
	err := getPurchaseDL(userID, "pack_purchase_info", "pack_name", container)
	if err != nil {
		return nil, err
	}
	return (*CheckSumContainer)(&container), nil
}

func getPurchaseDL(userID int, table string, target string, container map[string]*Checksum) error {
	rows, err := db.Query(fmt.Sprintf(`select
			song.song_id, song.checksum as "Audio Checksum",
			to_char(difficulty), chart_info.checksum as "Chart Checksum"
		from
			%[1]s, song, chart_info
		where
			user_id = :1
			and song.%[2]s = %[1]s.%[2]s
			and song.song_id = chart_info.song_id`,
		table, target),
		userID)
	if err != nil {
		log.Println(
			"Error occured while querying", table, "for download list.")
		return err
	}
	defer rows.Close()

	var (
		songID        string
		difficulty    string
		audioChecksum string
		chartChecksum string
	)
	for rows.Next() {
		rows.Scan(&songID, &audioChecksum, &difficulty, &chartChecksum)
		checksum, ok := container[songID]
		if !ok {
			checksum = &Checksum{
				AudioSum: map[string]string{"checksum": audioChecksum},
				ChartSum: map[string]map[string]string{},
			}
			container[songID] = checksum
		}
		checksum.ChartSum[difficulty] = map[string]string{
			"checksum": chartChecksum}
	}

	if err = rows.Err(); err != nil {
		log.Println("Error occured while reading quiried rows from WORLD_SONG_UNLOCK for download list.")
		return err
	}
	return nil
}

func getBydDL(userID int, container map[string]*Checksum) error {
	rows, err := db.Query("select song_id, checksum from chart_info where difficulty = 3")
	if err != nil {
		log.Println("Error occured while looking for BYD charts")
		return err
	}
	defer rows.Close()

	var (
		songID   string
		checksum string
	)
	for rows.Next() {
		rows.Scan(&songID, &checksum)
		_, ok := container[songID]
		if ok {
			continue
		}
		sum := &Checksum{
			AudioSum: map[string]string{},
			ChartSum: map[string]map[string]string{
				"3": {"checksum": checksum},
			},
		}
		container[songID] = sum
	}
	if err := rows.Err(); err != nil {
		log.Println("Error occured while reading rows queried for BYD charts.")
		return err
	}

	return nil
}
