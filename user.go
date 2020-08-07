package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/albrow/forms"
)

// SettingMap mapping request URL into column in PLAYER table
var SettingMap map[string]string

func init() {
	HandlerMap[path.Join(APIRoot, APIVer, "user/me")] = userInfoHandler
	HandlerMap[path.Join(APIRoot, APIVer, "user/me/setting")+"/"] = userSettingHandler
	InsideHandler[path.Join(APIRoot, APIVer, "user/me")] = getUserInfo

	SettingMap = map[string]string{}
	SettingMap["is_hide_rating"] = "is_hide_rating"
	SettingMap["max_stamina_notification_enabled"] = "max_stamina_notification"
}

func userInfoHandler(w http.ResponseWriter, r *http.Request) {
	userID := 1
	tojson, err := getUserInfo(userID)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Fprint(w, tojson.toJSON())
	}
}

func getUserInfo(userID int) (ToJSON, error) {
	var (
		userName              string
		displayName           string
		userCode              int64
		ticket                int
		partID                int8
		isLockedNameDuplicate string
		isSkillSealed         string
		currMap               string
		progBoost             int8
		stamina               int8
		nextFragstamTs        int64
		maxStaminaTs          int64
		staminaNotification   string
		hideRating            string
		favoriteCharacter     int8
		recentScoreDate       sql.NullInt64
		maxFriend             int8
		rating                int
		joinDate              int64
	)
	err := db.QueryRow(`select
			user_name, user_code, display_name, ticket, NVL(partner, 0),
			is_locked_name_duplicated, is_skill_sealed,
			curr_map, prog_boost, stamina,
			next_fragstam_ts, max_stamina_ts,
			max_stamina_notification, is_hide_rating, NVL(favorite_partner, 0),
			recent_score_date, max_friend, rating, join_date
		from
			player
		where
			user_id = :1`, userID).Scan(
		&userName, &userCode, &displayName, &ticket, &partID,
		&isLockedNameDuplicate, &isSkillSealed,
		&currMap, &progBoost, &stamina,
		&nextFragstamTs, &maxStaminaTs,
		&staminaNotification, &hideRating, &favoriteCharacter,
		&recentScoreDate, &maxFriend, &rating, &joinDate,
	)
	if err != nil {
		log.Println("Error occured while querying table PLAYER with USER_ID:", userID)
		return nil, err
	}
	charStatuses, err := getCharacterStats(userID)
	if err != nil {
		return nil, err
	}
	characters := []int8{}
	for _, status := range charStatuses {
		characters = append(characters, status.PartID)
	}
	worldUnlocks, err := getItemList(userID, "world_unlock", "item_name")
	if err != nil {
		return nil, err
	}
	worldSongUnlocks, err := getItemList(userID, "world_song_unlock", "item_name")
	if err != nil {
		return nil, err
	}
	packs, err := getItemList(userID, "pack_purchase_info", "pack_name")
	if err != nil {
		return nil, err
	}
	singles, err := getItemList(userID, "single_purchase_info", "song_id")
	if err != nil {
		return nil, err
	}
	coreInfoes, err := getCoreInfo(userID)
	if err != nil {
		return nil, err
	}
	recentScore, err := getRecentScore(userID)
	if err != nil {
		return nil, err
	}
	settings := Setting{
		staminaNotification == "t",
		hideRating == "t",
		favoriteCharacter,
	}
	if displayName == "" {
		displayName = userName
	}
	info := UserInfo{
		true,
		[]string{},
		charStatuses,
		[]string{},
		settings,
		userID,
		userName,
		displayName,
		fmt.Sprintf("%09d", userCode),
		ticket,
		partID,
		isLockedNameDuplicate == "y",
		isSkillSealed == "t",
		currMap,
		progBoost,
		nextFragstamTs,
		maxStaminaTs,
		stamina,
		worldUnlocks,
		worldSongUnlocks,
		singles,
		packs,
		characters,
		coreInfoes,
		[]ScoreRecord{recentScore},
		maxFriend,
		rating,
		joinDate,
	}

	return &info, nil
}

func getCoreInfo(userID int) ([]CoreInfo, error) {
	rows, err := db.Query(`select
			c.internal_id, c.core_name, amount
		from
			core_possess_info p, core c
		where
			user_id = :1
		and 
			c.core_id = p.core_id`, userID)
	if err != nil {
		log.Println("Error occured while querying table CORE_POSSESS_INFO")
		return nil, err
	}
	defer rows.Close()

	coreInfoes := []CoreInfo{}
	var (
		coreName   string
		internalID string
		amount     int8
	)
	for rows.Next() {
		rows.Scan(&coreName, &internalID, &amount)
		coreInfoes = append(coreInfoes, CoreInfo{coreName, amount, internalID})
	}
	if err = rows.Err(); err != nil {
		log.Println("Error occured while reading rows queried from CORE_POESS_INFO")
		return nil, err
	}
	return coreInfoes, nil
}

func getRecentScore(userID int) (ScoreRecord, error) {
	record := ScoreRecord{}
	err := db.QueryRow(`select
			s.song_id, s.difficulty, s.score,
			s.shiny_pure, s.pure, s.far, s.lost,
			s.health, s.modifier,
			s.clear_type, s2.clear_type "best clear type"
		from
			score s, best_score b, score s2
		where
			s.played_date = (select max(played_date) from score)
		and
			s.song_id = b.song_id
		and
			s2.song_id = b.song_id`).Scan(
		&record.SongID, &record.Difficulty, &record.Score,
		&record.Shiny, &record.Pure, &record.Far, &record.Lost,
		&record.Health, &record.Modifier,
		&record.ClearType, &record.BestClearType,
	)
	if err != nil {
		log.Println("Error occured while querying most recent score from SCORE")
		return record, err
	}

	return record, nil
}

func getItemList(userID int, tableName string, targetName string) ([]string, error) {
	rows, err := db.Query(
		fmt.Sprintf(
			"select %s from %s where user_id = :1", targetName, tableName),
		userID,
	)
	if err != nil {
		log.Println("Error occured while querying table ", tableName)
		return nil, err
	}
	defer rows.Close()

	results := []string{}
	var itemName string
	for rows.Next() {
		rows.Scan(&itemName)
		results = append(results, itemName)
	}

	if err = rows.Err(); err != nil {
		log.Println("Error occured while querying table ", tableName)
		return nil, err
	}
	return results, nil
}

func userSettingHandler(w http.ResponseWriter, r *http.Request) {
	target := path.Base(r.URL.Path)
	data, err := forms.Parse(r)
	if err != nil {
		log.Println(err)
	}

	userID, err := strconv.Atoi(r.Header.Get("i"))
	if err != nil {
		log.Println("Wrong userID: ", userID, " to do setting for.")
		return
	}

	val := data.Validator()
	val.Require("value")
	if val.HasErrors() {
		log.Println("Immproper setting request with no field in form")
		return
	}
	value, err := strconv.ParseBool(data.Get("value"))
	if err != nil {
		log.Println("Illegal value for making user setting: ", data.Get("value"))
		return
	}
	target, ok := SettingMap[target]
	if !ok {
		log.Printf(
			"Unknow setting option: `%s`\n been passed to /user/me/setting",
			r.URL.Path,
		)
		return
	}
	err = changeSetting(userID, target, value)
	if err != nil {
		log.Println(err)
	}
	tojson, err := getUserInfo(userID)
	if err != nil {
		log.Println(err)
	} else {
		container := Container{true, tojson, 0}
		fmt.Fprintln(w, container.toJSON())
	}
}

func changeSetting(userID int, target string, isOn bool) error {
	var value string
	if isOn {
		value = "t"
	} else {
		value = ""
	}
	_, err := db.Exec(
		fmt.Sprintf("update player set %s = '%s' where user_id = %d",
			target, value, userID,
		))
	if err != nil {
		log.Printf(
			"Error occured while modifying PLAYER for setting `%s` to `%v` with userID = %d",
			target, value, userID,
		)
		return err
	}

	return nil
}
