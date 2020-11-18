package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/albrow/forms"
)

var clearTypes = []string{
	"track-lost", "normal-clear", "full-recall",
	"pure-memory", "easy-clear", "hard-clear",
}

var diffs = []string{"PST", "PRS", "FTR", "BYD"}

func scoreLookupHandler(w http.ResponseWriter, r *http.Request) {
	userCode := path.Base(r.URL.Path)
	rows, err := db.Query(sqlStmtScoreLookup, userCode)
	if err == sql.ErrNoRows {
		log.Printf("%s: Currently no record in database for user `%s`\n", r.URL.Path, userCode)
		return
	} else if err != nil {
		log.Printf("%s: Error occured while looking up score record: %s\n", r.URL.Path, err)
		return
	}
	defer rows.Close()

	data, err := forms.Parse(r)
	if err != nil {
		log.Printf("%s: Error occured while parsing request form: %s\n", r.URL.Path, err)
	}
	isGetJSON := data.GetBool("json")

	var (
		songTitle  string
		baseRating float64
	)
	record := new(ScoreRecord)
	results := []string{}
	count := 1
	for rows.Next() {
		rows.Scan(
			&record.TimePlayed, &record.SongID, &songTitle, &record.Difficulty, &baseRating,
			&record.Score, &record.Shiny, &record.Pure, &record.Far, &record.Lost,
			&record.Rating, &record.Health, &record.ClearType,
		)
		result := genScoreText(record, songTitle, baseRating, isGetJSON, count)
		results = append(results, result)
		count++
	}
	if err := rows.Err(); err != nil {
		log.Printf("%s: Error occured while reading rows queried for score looking up", r.URL.Path)
		return
	}
	if isGetJSON {
		fmt.Fprintf(w, "[%s]", strings.Join(results, ","))
	} else {
		fmt.Fprint(w, genScoreHTML(userCode, results))
	}
}

func genScoreText(
	record *ScoreRecord, songTitle string, baseRating float64, isGetJSON bool, count int,
) string {
	ratingTitle := fmt.Sprintf("%d", int(baseRating))
	if (baseRating > 9.6 && baseRating < 10.0) || (baseRating > 10.6 && baseRating < 11.0) {
		ratingTitle += "+"
	}
	clearTypeTitle := clearTypes[record.ClearType]
	var result string
	if isGetJSON {
		if res, err := json.Marshal(record); err != nil {
			log.Println("Error occured while generating score JSON")
			log.Println(err)
		} else {
			result = string(res)
		}
	} else {
		result = fmt.Sprintf(
			scoreCardTemplate, count, songTitle, diffs[record.Difficulty], ratingTitle, baseRating,
			record.Score, record.Rating, record.Pure, record.Shiny, record.Far, record.Lost, clearTypeTitle,
			strings.Title(strings.Replace(clearTypeTitle, "-", " ", 1)),
			time.Unix(record.TimePlayed, 0),
		)
	}
	return result
}

func genScoreHTML(userCode string, results []string) string {
	var (
		b30    float64
		r10    float64
		output string
	)
	err := db.QueryRow(sqlStmtGetScoreLookupRating, userCode).Scan(&b30, &r10)
	if err != nil {
		log.Printf("Error occured while getting r10 and b30 data: %s\n", err)
	}
	output = strings.ReplaceAll(
		scorePageTemplate, "{{CONTENT}}", strings.Join(results, "\n"))
	output = strings.ReplaceAll(output, "{{R10}}", fmt.Sprintf("%.6f", r10))
	output = strings.ReplaceAll(output, "{{B30}}", fmt.Sprintf("%.6f", b30))
	return output
}
