package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/albrow/forms"
	"github.com/kardianos/osext"
)

var clearTypes = []string{
	"track-lost", "normal-clear", "full-recall",
	"pure-memory", "easy-clear", "hard-clear",
}

var diffs = []string{"PST", "PRS", "FTR", "BYD"}

var scoreCardTemplate string
var scorePageTemplate string

func init() {
	filename, err := osext.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exeDir := path.Dir(filename)
	content, err := ioutil.ReadFile(path.Join(exeDir, "static", "score_lookup", "card_template.html"))
	if err != nil {
		log.Fatal(err)
	}
	scoreCardTemplate = string(content)

	content, err = ioutil.ReadFile(path.Join(exeDir, "static", "score_lookup", "page_template.html"))
	if err != nil {
		log.Fatal(err)
	}
	scorePageTemplate = string(content)
}

func init() {
	R.Path(path.Join("/score", "b30", "{id:[0-9]{9}}")).Handler(
		http.HandlerFunc(scoreLookupHandler),
	)
}

func scoreLookupHandler(w http.ResponseWriter, r *http.Request) {
	userCode := path.Base(r.URL.Path)
	rows, err := db.Query(`select
			sc.played_date, so.song_id, so.title_local_en, sc.difficulty,
			c.rating as base_rating,
			sc.score, sc.shiny_pure, sc.pure, sc.far, sc.lost,
			sc.rating, sc.health, sc.clear_type
		from
			player p, best_score b, score sc, song so, chart_info c
		where
			p.user_code = ?
			and p.user_id = b.user_id
			and p.user_id = sc.user_id
			and sc.played_date = b.played_date
			and so.song_id = sc.song_id
			and so.song_id = c.song_id
			and c.difficulty = sc.difficulty
		order by sc.rating desc`, userCode)
	if err == sql.ErrNoRows {
		log.Printf("Currently no record in database for user `%s`\n", userCode)
		return
	} else if err != nil {
		log.Println("Error occured while looking up score record")
		log.Println(err)
		return
	}
	defer rows.Close()

	data, err := forms.Parse(r)
	if err != nil {
		log.Println("Error occured while parsing request form")
		log.Println(err)
	}
	isGetJSON := data.GetBool("json")

	var (
		playedDate int64
		songID     string
		songTitle  string
		difficulty int8
		baseRating float64
		score      int
		shiny      int
		pure       int
		far        int
		lost       int
		rating     float64
		health     int8
		clearType  int8
	)
	results := []string{}
	count := 1
	for rows.Next() {
		rows.Scan(
			&playedDate, &songID, &songTitle, &difficulty, &baseRating,
			&score, &shiny, &pure, &far, &lost,
			&rating, &health, &clearType,
		)
		result := genScoreText(
			playedDate, songID, songTitle, difficulty, baseRating,
			score, shiny, pure, far, lost,
			rating, health, clearType,
			isGetJSON, count,
		)
		results = append(results, result)
		count++
	}
	if err := rows.Err(); err != nil {
		log.Println("Error occured while reading rows queried for score looking up")
		return
	}
	if isGetJSON {
		fmt.Fprintf(w, "[%s]", strings.Join(results, ","))
	} else {
		fmt.Fprint(w, genScoreHTML(userCode, results))
	}
}

func genScoreText(
	playedDate int64, songID string, songTitle string, difficulty int8, baseRating float64,
	score int, shiny int, pure int, far int, lost int,
	rating float64, health int8, clearType int8,
	isGetJSON bool, count int,
) string {
	ratingTitle := fmt.Sprintf("%.1f", baseRating)
	if (baseRating > 9.6 && baseRating < 10.0) || (baseRating > 10.6 && baseRating < 11.0) {
		ratingTitle += "+"
	}
	clearTypeTitle := clearTypes[clearType]
	var result string
	if isGetJSON {
		record := ScoreRecord{
			songID, difficulty, rating,
			score, shiny, pure, far, lost,
			health, playedDate, 0, 0, clearType, 0,
		}
		if res, err := json.Marshal(record); err != nil {
			log.Println("Error occured while generating score JSON")
			log.Println(err)
		} else {
			result = string(res)
		}
	} else {
		result = fmt.Sprintf(
			scoreCardTemplate, count, songTitle, diffs[difficulty], ratingTitle, baseRating,
			score, rating, pure, shiny, far, lost, clearTypeTitle,
			strings.Title(strings.Replace(clearTypeTitle, "-", " ", 1)),
			time.Unix(playedDate, 0),
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
	err := db.QueryRow(`
	with
    best as (
		select ROW_NUMBER () OVER ( 
			order by rating desc
		) row_num,
		  rating
		from  best_score b, score s
		where b.user_id = ?1
			and b.user_id = s.user_id
			and b.played_date = s.played_date
	),
	recent as (
		select rating
		from  recent_score r, score s
		where r.user_id = ?1
			and r.is_recent_10 = 't'
			and r.user_id = s.user_id
			and r.played_date = s.played_date
	)
	select b30, r10
	from (
		select sum(rating) / count(rating) b30
		from best
		where row_num <= 30
	), (
		select sum(rating) / count(rating) r10
		from recent
	)`, userCode).Scan(&b30, &r10)
	if err != nil {
		log.Println("Error occured while getting r10 and b30 data")
		log.Println(err)
	}
	output = strings.ReplaceAll(
		scorePageTemplate, "{{CONTENT}}", strings.Join(results, "\n"))
	output = strings.ReplaceAll(output, "{{R10}}", fmt.Sprintf("%.6f", r10))
	output = strings.ReplaceAll(output, "{{B30}}", fmt.Sprintf("%.6f", b30))
	return output
}
