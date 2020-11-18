package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	record, err := makeRecord(r)
	tx, err := db.Begin()
	if err != nil {
		log.Printf("%s: Can't make transacation object: %s", r.URL.Path, err)
		return
	}

	inserter, err := newInserter(tx, userID)
	if err != nil {
		tx.Rollback()
		log.Println(err)
		return
	}

	targets := []func(*sql.Tx, int, *ScoreRecord) (int, error){
		insertScoreRecord,
		updateBestScore,
		inserter.insert,
		updatePlayerRating,
	}

	var rating int
	for _, target := range targets {
		if rating, err = target(tx, userID, record); err != nil {
			tx.Rollback()
			log.Println(err)
			http.Error(w, "Server side error", http.StatusInternalServerError)
			return
		}
	}

	result.Value["user_rating"] = rating

	tx.Commit()

	res, err := json.Marshal(result)
	if err != nil {
		log.Printf("%s: Error occured while generating output content: %s\n", r.URL.Path, err)
		return
	}
	fmt.Fprint(w, string(res))
}

func makeRecord(r *http.Request) (*ScoreRecord, error) {
	data, err := forms.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("error occured while parsing forms: %s", err)
	}
	val := data.Validator()
	for _, key := range ScoreKeys {
		val.Require(key)
	}
	if val.HasErrors() {
		for k, v := range val.ErrorMap() {
			log.Printf(r.URL.Path, ": %s - %s\n", k, v)
		}
		return nil, fmt.Errorf("score record received lacks of necessary filed(s) in form")
	}
	record := scoreRecordFromForm(data)
	record.TimePlayed = time.Now().Unix()
	err = record.scoreToRating()
	if err != nil {
		return nil, err
	}
	return record, nil
}

func insertScoreRecord(tx *sql.Tx, userID int, record *ScoreRecord) (int, error) {
	_, err := tx.Exec(sqlStmtInsertScore,
		userID,
		record.TimePlayed,
		record.SongID,
		record.Difficulty,
		record.Score,
		record.Shiny,
		record.Pure,
		record.Far,
		record.Lost,
		record.Rating,
		record.Health,
		record.ClearType,
	)
	if err != nil {
		return 0, fmt.Errorf("error occured while inserting new score record: %w", err)
	}
	return 0, nil
}

func updateBestScore(tx *sql.Tx, userID int, record *ScoreRecord) (int, error) {
	var (
		score      int
		playedDate int64
	)
	err := tx.QueryRow(
		sqlStmtLookupBestScore, userID, record.SongID, record.Difficulty,
	).Scan(&score, &playedDate)
	if err == sql.ErrNoRows {
		_, err = tx.Exec(sqlStmtInsertBestScore, userID, record.TimePlayed)
		if err != nil {
			return 0, fmt.Errorf("error occured while insert new best score: %w", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("error occured while looking up best score: %w", err)
	} else if record.Score > score {
		_, err = tx.Exec(sqlStmtReplaceBestScore, record.TimePlayed, playedDate)
		if err != nil {
			return 0, fmt.Errorf("error occured while replacing best score: %w", err)
		}
	}
	return 0, nil
}

type recentScoreItem struct {
	playedDate int64
	rating     float64
}

type recentScoreInserter struct {
	r10         map[string]*recentScoreItem
	normalItems []*recentScoreItem
}

func newInserter(tx *sql.Tx, userID int) (*recentScoreInserter, error) {
	rows, err := tx.Query(sqlStmtLookupRecentScore, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inserter := &recentScoreInserter{
		r10:         map[string]*recentScoreItem{},
		normalItems: []*recentScoreItem{},
	}

	iden := ""
	isR10 := ""
	for rows.Next() {
		item := new(recentScoreItem)
		rows.Scan(&item.rating, &item.playedDate, &iden, &isR10)
		if isR10 == "t" {
			inserter.r10[iden] = item
		} else {
			inserter.normalItems = append(inserter.normalItems, item)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inserter, nil
}

func (inserter *recentScoreInserter) insert(tx *sql.Tx, userID int, record *ScoreRecord) (int, error) {
	// (newItem recentScoreItem, newIdentifier string, score int, clearType int8)
	target := &recentScoreItem{
		rating:     record.Rating,
		playedDate: record.TimePlayed,
	}
	newIdentifier := fmt.Sprintf("%s%d", record.SongID, record.Difficulty)
	if target, replacement, isR10, needNewR10, err := inserter.insertIntoR10(tx, userID, newIdentifier, target, record.Score, record.ClearType); err != nil {
		return 0, err
	} else if err = inserter.insertIntoNormalItem(tx, userID, target, replacement, isR10, needNewR10); err != nil {
		return 0, err
	}
	return 0, nil
}

func (inserter *recentScoreInserter) insertIntoR10(tx *sql.Tx, userID int, identifier string, target *recentScoreItem, score int, clearType int8) (*recentScoreItem, *recentScoreItem, bool, bool, error) {
	// target may change during trying to insert it into r10, ret_target is
	// the final target in this process, and the starting target for next
	// process (insert into normat item).
	var retTarget *recentScoreItem = nil
	// candidate record that current target will possiblely replace.
	var replacement *recentScoreItem = nil
	// wheather current target record should be marked as an r10.
	isR10 := false
	// need_new_r10, if true, record with highest rating among normal item
	// will become a new r10 item.
	needNewR10 := false
	if record, ok := inserter.r10[identifier]; ok {
		if record.rating <= target.rating {
			if _, err := tx.Exec(sqlStmtReplaceRecnetScore, target.playedDate, "t", userID, record.playedDate); err != nil {
				return retTarget, replacement, isR10, needNewR10, err
			}
			retTarget = record
		} else {
			retTarget = target
		}
	} else {
		if len(inserter.r10) < 10 {
			if len(inserter.r10)+len(inserter.normalItems) < 30 {
				if _, err := tx.Exec(sqlStmtInsertRecentScore, userID, target.playedDate, "t"); err != nil {
					return retTarget, replacement, isR10, needNewR10, err
				}
				// no need for further process, no target any more
				retTarget = nil
			} else {
				needNewR10 = true
			}
		} else if len(inserter.r10)+len(inserter.normalItems) < 30 {
			// just inserte record into normal item list, do nothing with r10
		} else {
			isEx := score >= 9_800_000
			isHardClear := clearType == 5
			for _, item := range inserter.r10 {
				if (isEx || isHardClear) && target.rating < item.rating {
					continue
				}
				if item.rating <= target.rating {
					isR10 = true
				}
				if replacement == nil {
					replacement = item
				} else if item.playedDate < replacement.playedDate {
					replacement = item
				}
			}
			if isR10 {
				if _, err := tx.Exec(sqlStmtReplaceRecnetScore, target.playedDate, "t", userID, replacement.playedDate); err != nil {
					return retTarget, replacement, isR10, needNewR10, err
				}
				retTarget = replacement
				isR10 = false
				replacement = nil
			} else {
				needNewR10 = true
			}
		}
	}

	return retTarget, replacement, isR10, needNewR10, nil
	// Possible return values:
	// None, None, false, false. When both r10 and r30 is not full.
	// Some, None,  true, false. When r10 is not full but r30 is full.
	// Some, Some, false,  true. When new record can't be insert into r10.
	// Some, None, false, false. When new record's identifier collides, or
	//                           new record insert into r10 and take a old
	//                           record out of r10 group
}

func (inserter *recentScoreInserter) insertIntoNormalItem(tx *sql.Tx, userID int, target *recentScoreItem, replacement *recentScoreItem, isR10 bool, needNewR10 bool) error {
	if target == nil {
		return nil
	}
	if isR10 {
		// is_r10 will be true only when r10 is not full but r30 is.
		if _, err := tx.Exec(sqlStmtReplaceRecnetScore, target.playedDate, "t", userID, inserter.normalItems[0].playedDate); err != nil {
			return err
		}
		target = inserter.normalItems[0]
	}
	if len(inserter.r10)+len(inserter.normalItems) < 30 {
		if _, err := tx.Exec(sqlStmtInsertRecentScore, userID, target.playedDate, ""); err != nil {
			return err
		}
		return nil
	}
	for _, item := range inserter.normalItems {
		if replacement == nil {
			replacement = item
		} else if item.playedDate < replacement.playedDate {
			replacement = item
			needNewR10 = false
		}
	}
	if replacement.playedDate != target.playedDate {
		if _, err := tx.Exec(sqlStmtReplaceRecnetScore, target.playedDate, "", userID, replacement.playedDate); err != nil {
			return err
		}
	}
	if needNewR10 {
		// if need_new_r10 is true, record being replaced is in r10, so it's
		// safe to directly take highest rating record from normal item group
		// as new a r10 record.
		newR10 := inserter.normalItems[0].playedDate
		if _, err := tx.Exec(sqlStmtInsertRecentScore, newR10, "t", userID, newR10); err != nil {
			return err
		}
	}
	return nil
}

func updatePlayerRating(tx *sql.Tx, userID int, _ *ScoreRecord) (int, error) {
	var rating int
	err := tx.QueryRow(sqlStmtComputeRating, userID).Scan(&rating)
	if err != nil {
		return rating, fmt.Errorf("error occured while compute user rating: %w", err)
	}

	if _, err := tx.Exec(sqlStmtUpdateRating, rating, userID); err != nil {
		return rating, fmt.Errorf("error occured while modifying rating of user: %d: %w", userID, err)
	}
	return rating, nil
}
