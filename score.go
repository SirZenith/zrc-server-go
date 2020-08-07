package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/albrow/forms"
)

// ScoreKeys are keys that must appear in a ScoreRecord
var ScoreKeys = []string{
	"song_id", "difficulty", "score",
	"shiny_perfect_count", "perfect_count", "near_count", "miss_count",
	"health", "modifier", "beyond_gauge", "clear_type",
}

func init() {
	HandlerMap[path.Join(APIRoot, APIVer, "score", "token")] = scoreTokenHandler
	HandlerMap[path.Join(APIRoot, APIVer, "score", "song")] = scoreUploadHandler
}

func scoreTokenHandler(w http.ResponseWriter, r *http.Request) {
	token := new(ScoreToken)
	container := Container{true, token, 0}
	fmt.Fprint(w, container.toJSON())
}

func scoreUploadHandler(w http.ResponseWriter, r *http.Request) {
	result := ScoreUploadResult{true, map[string]int{"user_rating": 0}}
	userID, err := strconv.Atoi(r.Header.Get("i"))
	if err != nil {
		log.Println(r.URL.Path, ": Error occur while reading user id from header")
		log.Println(err)
		return
	}
	data, err := forms.Parse(r)
	if err != nil {
		log.Println(r.URL.Path, ": Error occured while parsing forms")
		log.Println(err)
		return
	}
	val := data.Validator()
	for _, key := range ScoreKeys {
		val.Require(key)
	}
	if val.HasErrors() {
		log.Println(r.URL.Path, ": Score record received lacks of necessary filed(s) in form")
		for k, v := range val.ErrorMap() {
			log.Printf(r.URL.Path, ": %s - %s\n", k, v)
		}
		return
	}
	rating, err := scoreToRating(
		data.Get("song_id"), data.GetInt("difficulty"),
		data.GetFloat("score"),
	)
	if err != nil {
		log.Println(err)
		return
	}
	playedDate := time.Now().Unix()
	tx, err := db.Begin()
	if err != nil {
		log.Printf("%s: Can't make transacation object", r.URL.Path)
		log.Println(err)
		return
	}
	_, err = tx.Exec(`insert into score (
		user_id, played_date, song_id, difficulty, score,
		shiny_pure, pure, far, lost, rating,
		health, clear_type
		) values(:1, :2, :3, :4, :5, :6,:7, :8, :9, :10, :11, :12)`,
		userID, playedDate, data.Get("song_id"),
		data.GetInt("difficulty"), data.GetInt("score"),
		data.GetInt("shiny_perfect_count"), data.GetInt("perfect_count"),
		data.GetInt("near_count"), data.GetInt("miss_count"), rating,
		data.GetInt("health"), data.GetInt("clear_type"),
	)
	if err != nil {
		log.Printf("%s: Error occured while inserting to SCORE", r.URL.Path)
		log.Println(err)
		return
	}

	userRating, err := updateRating(
		tx, userID, playedDate, rating,
		data.GetInt("score"), data.Get("song_id"), data.GetInt("clear_type"),
	)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return
	}
	tx.Commit()
	result.Value["user_rating"] = userRating

	res, err := json.Marshal(result)
	if err != nil {
		log.Printf("%s: Error occured while generating output content\n", r.URL.Path)
		log.Println(err)
		return
	}
	fmt.Fprint(w, string(res))
}

func scoreToRating(songID string, difficulty int, score float64) (float64, error) {
	var baseRating float64
	err := db.QueryRow(
		"select rating from chart_info where song_id = :1 and difficulty = :2",
		songID, difficulty,
	).Scan(&baseRating)
	if err != nil {
		log.Printf("Error while querying base rating for `%s`\n", songID)
		return 0, err
	}

	rating := 0.0
	if score >= 10_000_000 {
		rating = baseRating + 2
	} else if score >= 9_800_000 {
		rating = baseRating + 1 + (score-9_800_000)/200_000
	} else if rating = baseRating + (score-9_500_000)/300_000; rating < 0 {
		rating = 0
	}
	return rating, nil
}

// RecentScoreItem is used for picking score record when updating rating
type RecentScoreItem struct {
	playedDate  int64
	repeatTimes int
	rating      float64
}

func updateRating(tx *sql.Tx, userID int, newPlayedDate int64, newRating float64, score int, songID string, clearType int) (int, error) {
	var rating int = 0
	err := tx.QueryRow(
		"select rating from player where user_id = :1",
		userID,
	).Scan(&rating)
	if err != nil {
		log.Println(
			"Error occured when querying user rating during updating rating with userID =",
			userID,
		)
		return rating, err
	}
	err = updateRatingRecent(tx, userID, newPlayedDate, score, newRating, clearType)
	if err != nil {
		return rating, err
	} else if err = updateBestScore(tx, userID, newPlayedDate, songID, newRating); err != nil {
		return rating, err
	} else if err = updateRatingBest(tx, userID, newPlayedDate, newRating); err != nil {
		return rating, err
	}
	err = tx.QueryRow(`
	select round((b30 + r10) / 40 * 100) from (
		select
			sum(rating) b30
		from 
			best_30 b, score s
		where
			b.user_id = :1
		and 
			b.user_id = s.user_id
		and 
			b.played_date = s.played_date
	), (
		select
			sum(rating) r10
		from 
			recent_10 r, score s
		where
			r.user_id = :1
		and 
			r.user_id = s.user_id
		and 
			r.played_date = s.played_date
	)`, userID).Scan(&rating)
	if err != nil {
		log.Println("Error occured while compute user rating")
		return rating, err
	}

	_, err = tx.Exec(
		"update player set rating = :1 where user_id = :2",
		rating, userID,
	)
	if err != nil {
		log.Println("Error occured while modifying rating of user:", userID)
		return rating, err
	}
	return rating, nil
}

func updateRatingRecent(tx *sql.Tx, userID int, newPlayedDate int64, score int, newRating float64, clearType int) error {
	rows, err := tx.Query(`
	with
		r30 as (select s.played_date, (s.song_id || s.difficulty) as ident, s.rating
			from
				recent_played r, score s
			where
				r.user_id = :1
				and r.user_id = s.user_id
				and r.played_date = s.played_date
		),
		repeat_table as (select ident, count(*) as repeat_times from r30 group by ident)
	select
		played_date, repeat_times, rating, (select count(*) as diff_count from repeat_table)
	from
		r30, repeat_table
	where
		r30.ident = repeat_table.ident
	order by
		rating desc`, userID)
	if err == sql.ErrNoRows {
		_, err := tx.Exec(
			"insert into recent_played(user_id, played_date) values(:1, :2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while insterting into RECENT_PLAYED")
		}
		_, err = tx.Exec(
			"insert into recent_10(user_id, played_date) values(:1, :2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while insterting into RECENT_10")
		}
	} else if err != nil {
		log.Println("Error occured while querying table RECENT_PLAYED")
		return err
	}
	defer rows.Close()

	var (
		playedDate  int64
		repeatTimes int
		rating      float64
		diffCount   int
	)
	results := []RecentScoreItem{}
	for rows.Next() {
		rows.Scan(&playedDate, &repeatTimes, &rating, &diffCount)
		results = append(results, RecentScoreItem{playedDate, repeatTimes, rating})
	}

	if err = rows.Err(); err != nil {
		log.Println("Error occured while reading rows queried from RECENT_PLAYED")
		return err
	}

	if len(results) < 10 {
		_, err := tx.Exec(
			"insert into recent_played(user_id, played_date) values(:1, :2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while insterting into RECENT_PLAYED")
		}
		_, err = tx.Exec(
			"insert into recent_10(user_id, played_date) values(:1, :2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while insterting into RECENT_10")
		}
	} else if len(results) < 30 {
		_, err := tx.Exec(
			"insert into recent_played(user_id, played_date) values(:1, :2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while insterting into RECENT_PLAYED")
		}
		if newRating > results[9].rating {
			_, err = tx.Exec(
				`update recent_10 set played_date = :1
				where user_id = :2 and played_date = :3`,
				newPlayedDate, userID, results[9].playedDate,
			)
			if err != nil {
				log.Println("Error occured while modifying RECENT_10")
				return err
			}
		}
	} else {
		isEx := score >= 9_800_000
		noMoreThan10 := diffCount < 10
		isHardClear := clearType == 5
		targetInd := 0
		for i, result := range results {
			if (isEx || isHardClear) && i < 10 {
				continue
			} else if noMoreThan10 && result.repeatTimes < 2 {
				continue
			} else if result.playedDate < results[targetInd].playedDate {
				targetInd = i
			}
		}

		if targetInd < 10 {
			var replacement int64 = 0
			if newRating > results[10].rating {
				replacement = newPlayedDate
			} else {
				replacement = results[10].playedDate
			}
			_, err = tx.Exec(
				`update recent_10 set played_date = :1
				where user_id = :2 and played_date = :3`,
				replacement, userID, results[targetInd].playedDate,
			)
			if err != nil {
				log.Println("Error occured while insterting into RECENT_10")
			}
		}
		_, err = tx.Exec(
			`update recent_played set played_date = :1
			where user_id = :2 and played_date = :3`,
			newPlayedDate, userID, results[targetInd].playedDate,
		)
		if err != nil {
			log.Println("Error occured while modifying RECENT_PLAYED")
			return err
		}
	}

	return nil
}

func updateBestScore(tx *sql.Tx, userID int, newPlayedDate int64, songID string, newRating float64) error {
	var (
		rating     float64
		playedDate int64
	)
	err := tx.QueryRow(`select
			s.rating, s.played_date
		from
			best_score, score s
		where
			best_score.user_id = :1
		and
			best_score.user_id = s.user_id
		and
			best_score.played_date = s.played_date
		and
			s.song_id = :2`, userID, songID).Scan(&rating, &playedDate)
	if err == sql.ErrNoRows {
		_, err = tx.Exec(
			"insert into best_score(user_id, played_date) values(:1, :2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while insert into BEST_SCORE")
			return err
		}
	} else if err != nil {
		log.Println("Error occured while querying table BEST_SCORE")
		return err
	} else if newRating > rating {
		_, err = tx.Exec(
			"update best_score set played_date = :1 where played_date = :2",
			newPlayedDate, playedDate,
		)
		if err != nil {
			log.Println("Error occured while modifying table BEST_SCORE")
			return err
		}
	}
	return nil
}

func updateRatingBest(tx *sql.Tx, userID int, newPlayedDate int64, newRating float64) error {
	var (
		rating     float64
		playedDate int64
		b30Count   int
	)
	err := tx.QueryRow(`
	with
		b30 as (
			select
					s.rating, s.played_date
			from
				best_30, score s
			where
				best_30.user_id = :1
			and
				best_30.user_id = s.user_id
			and
				best_30.played_date = s.played_date
		)
	select
		rating, played_date, b30_count
	from
		b30, (select count(*) b30_count from b30)
	where
		rating = (select min(rating) from b30)
	and
		rownum = 1`, userID).Scan(&rating, &playedDate, &b30Count)
	if err == sql.ErrNoRows {
		_, err = tx.Exec(
			"insert into best_30(user_id, played_date) values(:1, :2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while inserting new record to EMPTY table BEST_30")
			return err
		}
	} else if err != nil {
		log.Println("Error occured while querying table BEST_30")
		return err
	} else if b30Count < 30 {
		_, err = tx.Exec(
			"insert into best_30(user_id, played_date) values(:1, :2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while inserting new record to table BEST_30")
			return err
		}
	} else if newRating > rating {
		_, err = tx.Exec(
			"update best_30 set played_date = :1 where played_date = :2",
			newPlayedDate, playedDate,
		)
		if err != nil {
			log.Println("Error occured while modifying record table BEST_30")
			return err
		}
	}
	return nil
}
