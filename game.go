package main

import (
	"fmt"
	"log"
	"net/http"
)

func gameInfoHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
	}
	tojson, err := getGameInfo(userID, r)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Fprint(w, tojson.toJSON())
	}
}

func getGameInfo(_ int, _ *http.Request) (ToJSON, error) {
	var (
		worldRankingEnabled  string
		isBydChapterUnlocked string
	)
	info := new(GameInfo)
	err := db.QueryRow(slqStmtGameInfo).Scan(
		&info.Now,
		&info.MaxStam,
		&info.StaminaRecoverTick,
		&info.CoreExp,
		&worldRankingEnabled,
		&isBydChapterUnlocked,
	)
	if err != nil {
		log.Println("Error occured while querying GAME_INFO")
		return nil, err
	}
	levelsteps := []levelStep{}
	rows, err := db.Query(sqlStmtLevelStep)
	if err != nil {
		return nil, fmt.Errorf("Error occured while querying table LEVEL_EXP: %w", err)
	}
	defer rows.Close()

	step := new(levelStep)
	for rows.Next() {
		rows.Scan(&step.Lv, &step.Exp)
		levelsteps = append(levelsteps, *step)
	}
	if err := rows.Err(); err != nil {
		log.Println("Error occured while reading rows from LEVEL_EXP")
		return nil, err
	}

	info.WorldRankingEnabled = worldRankingEnabled == "t"
	info.BydUnlocked = isBydChapterUnlocked == "t"
	info.LevelSteps = levelsteps

	return info, nil
}
