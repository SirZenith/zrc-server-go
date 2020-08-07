package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
)

func init() {
	HandlerMap[path.Join(APIRoot, APIVer, "game/info")] = gameInfoHandler
	InsideHandler[path.Join(APIRoot, APIVer, "game/info")] = getGameInfo
}

func gameInfoHandler(w http.ResponseWriter, r *http.Request) {
	tojson, err := getGameInfo(0)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Fprint(w, tojson.toJSON())
	}
}

func getGameInfo(_ int) (ToJSON, error) {
	var (
		now                  int64
		maxStam              int8
		stamRecoverTick      int
		coreExp              int
		worldRankingEnabled  string
		isBydChapterUnlocked string
	)
	err := db.QueryRow(`select
			FLOOR(SYSDATE - to_date('19700101', 'YYYYMMDD')) * 24 * 60 * 60,
			max_stamina, stamina_recover_tick, core_exp,
			world_ranking_enabled, is_byd_chapter_unlocked
		from game_info`).Scan(&now, &maxStam, &stamRecoverTick, &coreExp, &worldRankingEnabled, &isBydChapterUnlocked)
	if err != nil {
		log.Println("Error occured while querying GAME_INFO")
		return nil, err
	}
	levelsteps := []map[string]int{}
	rows, err := db.Query("select lv, exp_val from level_exp")
	if err != nil {
		log.Println("Error occured while querying table LEVEL_EXP")
		return nil, err
	}
	defer rows.Close()

	var (
		lv  int
		exp int
	)
	for rows.Next() {
		rows.Scan(&lv, &exp)
		levelsteps = append(levelsteps, map[string]int{
			"level": lv, "level_exp": exp,
		})
	}
	if err := rows.Err(); err != nil {
		log.Println("Error occured while reading rows from LEVEL_EXP")
		return nil, err
	}

	info := GameInfo{
		maxStam, stamRecoverTick, coreExp, now, levelsteps,
		worldRankingEnabled == "t", isBydChapterUnlocked == "t",
	}

	return &info, nil
}
