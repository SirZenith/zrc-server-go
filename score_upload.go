package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/albrow/forms"
)

// ScoreKeys are keys that must appear in a ScoreRecord
var ScoreKeys = []string{
	"song_id", "difficulty", "score",
	"shiny_perfect_count", "perfect_count", "near_count", "miss_count",
	"health", "modifier", "beyond_gauge", "clear_type",
}

var errorZeroRating = errors.New("Rating for this chart is 0")

func init() {
	R.Path(path.Join(APIRoot, "score", "token")).Handler(
		http.HandlerFunc(scoreTokenHandler),
	)
	R.Path(path.Join(APIRoot, "score", "song")).Handler(
		http.HandlerFunc(scoreUploadHandler),
	)
}

func scoreTokenHandler(w http.ResponseWriter, r *http.Request) {
	_, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
	}
	token := new(ScoreToken)
	container := Container{true, token, 0}
	fmt.Fprint(w, container.toJSON())
}

func scoreUploadHandler(w http.ResponseWriter, r *http.Request) {
	result := ScoreUploadResult{true, map[string]int{"user_rating": 0}}
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
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
		) values(?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)`,
		userID, playedDate, data.Get("song_id"),
		data.GetInt("difficulty"), data.GetInt("score"),
		data.GetInt("shiny_perfect_count"), data.GetInt("perfect_count"),
		data.GetInt("near_count"), data.GetInt("miss_count"), rating,
		data.GetInt("health"), data.GetInt("clear_type"),
	)
	if err != nil {
		tx.Rollback()
		log.Printf("%s: Error occured while inserting to SCORE", r.URL.Path)
		log.Println(err)
		return
	}

	userRating, err := updateRating(
		tx, userID, playedDate, rating,
		data.GetInt("score"), data.Get("song_id"), data.GetInt("clear_type"), data.GetInt("difficulty"),
	)
	if err != nil {
		tx.Rollback()
		log.Println(err)
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
		"select rating from chart_info where song_id = ?1 and difficulty = ?2",
		songID, difficulty,
	).Scan(&baseRating)
	if err != nil {
		log.Printf("Error while querying base rating for `%s`\n", songID)
		return 0, err
	} else if baseRating == 0 {
		log.Printf("Zero Rating for `%s %d`\n", songID, difficulty)
		return 0, errorZeroRating
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

func updateRating(tx *sql.Tx, userID int, newPlayedDate int64, newRating float64, score int, songID string, clearType int, difficulty int) (int, error) {
	var rating int = 0
	err := tx.QueryRow(
		"select rating from player where user_id = ?",
		userID,
	).Scan(&rating)
	if err != nil {
		log.Println(
			"Error occured when querying user rating during updating rating with userID =",
			userID,
		)
		return rating, err
	}
	songIden := fmt.Sprintf("%s%d", songID, difficulty)
	err = updateRatingRecent(tx, userID, newPlayedDate, score, newRating, clearType, songIden)
	if err != nil {
		return rating, err
	} else if err := updateBestScore(tx, userID, newPlayedDate, songID, score, difficulty); err != nil {
		return rating, err
	}
	err = tx.QueryRow(`
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
	select round((b30 + r10) / (b30_count + r10_count) * 100)
	from (
		select sum(rating) b30, count(rating) b30_count
		from best
		where row_num <= 30
	), (
		select sum(rating) r10, count(rating) r10_count
		from recent
	)`, userID).Scan(&rating)
	if err != nil {
		log.Println("Error occured while compute user rating")
		return rating, err
	}

	_, err = tx.Exec(
		"update player set rating = ?1 where user_id = ?2",
		rating, userID,
	)
	if err != nil {
		log.Println("Error occured while modifying rating of user:", userID)
		return rating, err
	}
	return rating, nil
}

// RecentScoreItem is used for picking score record when updating rating
type RecentScoreItem struct {
	playedDate int64
	rating     float64
}

type recentScoreInserter struct {
	r10         map[string]*RecentScoreItem
	normalItems []*RecentScoreItem
}

func (inserter *recentScoreInserter) insert(tx *sql.Tx, userID int, score int, clearType int, targetIden string, target *RecentScoreItem) error {
	item, ok := inserter.r10[targetIden]
	var replacement *RecentScoreItem = nil
	isR10 := "t"
	if !ok {
		isEx := score >= 9_800_000
		isHardClear := clearType == 5
		for _, item := range inserter.r10 {
			if (isEx || isHardClear) && target.rating < item.rating {
				continue
			} else if replacement == nil {
				replacement = item
			} else if item.playedDate < replacement.playedDate {
				replacement = item
			}
		}
	} else if item.rating <= target.rating {
		if _, err := tx.Exec(
			`update recent_score set played_date = ?1 where user_id = ?2 and played_date = ?3`,
			target.playedDate, userID, item.playedDate,
		); err != nil {
			log.Println("Error occured while replacing record in RECENT_PLAYED for R10")
			return err
		}
		target = item
		isR10 = ""
	}

	if replacement != nil {
	} else if len(inserter.normalItems) != 0 {
		replacement = inserter.normalItems[0]
		for _, item := range inserter.normalItems {
			if item.playedDate < replacement.playedDate {
				replacement = item
				isR10 = ""
			}
		}
	} else {
		if _, err := tx.Exec(
			`insert into recent_score(user_id, played_date, is_recent_10)
			values(?1, ?2, ?3)`,
			userID, target.playedDate, isR10,
		); err != nil {
			log.Println("Error occured while insterting into empty R30")
			return err
		}

		return nil
	}

	if _, err := tx.Exec(
		`update recent_score set played_date = ?1, is_recent_10 = ?2
			where user_id = ?3 and played_date = ?4`,
		target.playedDate, isR10, userID, replacement.playedDate,
	); err != nil {
		log.Println("Error occured while replacing record in RECENT_PLAYED R30")
		return err
	}

	return nil
}

func (inserter *recentScoreInserter) insertNormalItem(target *RecentScoreItem, replacement *RecentScoreItem) error {
	return nil
}

func updateRatingRecent(tx *sql.Tx, userID int, newPlayedDate int64, score int, newRating float64, clearType int, songIden string) error {
	rows, err := tx.Query(`
	with
		r30 as (select s.played_date, (s.song_id || s.difficulty) iden, s.rating, r.is_recent_10
			from
				recent_score r, score s
			where
				r.user_id = ?1
				and r.user_id = s.user_id
				and r.played_date = s.played_date
		)
	select
		played_date, rating, iden, is_recent_10
	from
		r30
	order by
		rating desc`, userID)
	if err == sql.ErrNoRows {
		_, err := tx.Exec(
			`insert into recent_score(user_id, played_date, is_recent_10)
			values(?1, ?2, 't')`,
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while insterting into empty RECENT_SCORE")
		}
	} else if err != nil {
		log.Println("Error occured while querying table RECENT_SCORE")
		return err
	}
	defer rows.Close()

	target := &RecentScoreItem{newPlayedDate, newRating}
	inserter := recentScoreInserter{map[string]*RecentScoreItem{}, []*RecentScoreItem{}}
	var (
		playedDate int64
		rating     float64
		iden       string
		isR10      string
	)
	for rows.Next() {
		rows.Scan(&playedDate, &rating, &iden, &isR10)
		item := &RecentScoreItem{playedDate, rating}
		if isR10 == "t" {
			inserter.r10[iden] = item
		} else {
			inserter.normalItems = append(inserter.normalItems, item)
		}
	}

	if err = rows.Err(); err != nil {
		log.Println("Error occured while reading rows queried from RECENT_SCORE")
		return err
	}

	return inserter.insert(tx, userID, score, clearType, songIden, target)
}

func updateBestScore(tx *sql.Tx, userID int, newPlayedDate int64, songID string, newScore int, difficulty int) error {
	var (
		score      int
		playedDate int64
	)
	err := tx.QueryRow(`select
			s.score, s.played_date
		from
			best_score b, score s
		where
			b.user_id = ?1
			and b.user_id = s.user_id
			and b.played_date = s.played_date
			and s.song_id = ?2
			and s.difficulty = ?3`,
		userID, songID, difficulty).Scan(&score, &playedDate)
	if err == sql.ErrNoRows {
		_, err = tx.Exec(
			"insert into best_score(user_id, played_date) values(?1, ?2)",
			userID, newPlayedDate,
		)
		if err != nil {
			log.Println("Error occured while insert into BEST_SCORE")
			return err
		}
	} else if err != nil {
		log.Println("Error occured while querying table BEST_SCORE")
		return err
	} else if newScore > score {
		_, err = tx.Exec(
			`update best_score set played_date = ?1 where played_date = ?2`,
			newPlayedDate, playedDate,
		)
		if err != nil {
			log.Println("Error occured while modifying table BEST_SCORE")
			return err
		}
	}
	return nil
}
