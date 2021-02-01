package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/albrow/forms"
)

// SettingMap mapping request URL into column in PLAYER table
var SettingMap = map[string]string{}

func init() {
	SettingMap["is_hide_rating"] = "is_hide_rating"
	SettingMap["max_stamina_notification_enabled"] = "max_stamina_notification"
	SettingMap["favorite_character"] = "favorite_partner"
}

func presentMeHandler(w http.ResponseWriter, r *http.Request) {
	var (
		userID int
		err    error
	)
	if NeedAuth {
		userID, err = verifyBearerAuth(r.Header.Get("Authorization"))
		if err != nil {
			c := Container{false, nil, 203}
			http.Error(w, c.toJSON(), http.StatusUnauthorized)
			return
		}
	} else {
		userID = staticUserID
	}
	tojson, err := presentMe(userID, r)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Fprint(w, tojson.toJSON())
	}
}

func presentMe(_ int, _ *http.Request) (ToJSON, error) {
	return &EmptyList{}, nil
}

func userInfoHandler(w http.ResponseWriter, r *http.Request) {
	var (
		userID int
		err    error
	)
	if NeedAuth {
		userID, err = verifyBearerAuth(r.Header.Get("Authorization"))
		if err != nil {
			c := Container{false, nil, 203}
			http.Error(w, c.toJSON(), http.StatusUnauthorized)
			return
		}
	} else {
		userID = staticUserID
	}
	tojson, err := getUserInfo(userID, r)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Fprint(w, tojson.toJSON())
	}
}

func getUserInfo(userID int, _ *http.Request) (ToJSON, error) {
	var (
		userCode              int64
		isLockedNameDuplicate string
		isSkillSealed         string
		staminaNotification   string
		hideRating            string
		recentScoreDate       sql.NullInt64
	)
	info := new(UserInfo)
	err := db.QueryRow(sqlStmtUserInfo, userID).Scan(
		&info.Name,
		&userCode,
		&info.DisplaName,
		&info.Ticket,
		&info.PartID,
		&isLockedNameDuplicate,
		&isSkillSealed,
		&info.CurrentMap,
		&info.ProgBoost,
		&info.Stamina,
		&info.NextFragstamTs,
		&info.MaxStaminaTs,
		&staminaNotification,
		&hideRating,
		&info.Settings.FavoriteCharacter,
		&recentScoreDate,
		&info.MaxFriend,
		&info.Rating,
		&info.JoinDate,
	)
	if err != nil {
		log.Println("Error occured while querying table PLAYER with USER_ID:", userID)
		return nil, err
	}

	info.CurrAvailableMaps = []string{}
	info.Friends = []string{}
	info.Settings.StaminaNotification = staminaNotification == "t"
	info.Settings.HideRating = hideRating == "t"
	info.UserCode = fmt.Sprintf("%09d", userCode)
	info.IsLockedNameDuplicate = isLockedNameDuplicate == "t"
	info.IsSkillSealed = isSkillSealed == "t"

	var charStatses []CharacterStats
	if charStatses, err = getCharacterStats(userID, -1); err != nil {
		return nil, err
	}
	info.CharacterStats = charStatses

	characters := []int8{}
	for _, status := range charStatses {
		characters = append(characters, status.PartID)
	}
	info.Characters = characters

	var worldUnlocks []string
	if worldUnlocks, err = getItemList(userID, "world_unlock", "item_name"); err != nil {
		return nil, err
	}
	info.WorldUnlocks = worldUnlocks

	var worldSongUnlocks []string
	if worldSongUnlocks, err = getItemList(userID, "world_song_unlock", "item_name"); err != nil {
		return nil, err
	}
	info.WorldSongs = worldSongUnlocks

	var packs []string
	if packs, err = getItemList(userID, "pack_purchase_info", "pack_name"); err != nil {
		return nil, err
	}
	info.Packs = packs

	var singles []string
	if singles, err = getItemList(userID, "single_purchase_info", "song_id"); err != nil {
		return nil, err
	}
	info.Singles = singles

	var coreInfoes []CoreInfo
	if coreInfoes, err = getCoreInfo(userID); err != nil {
		return nil, err
	}
	info.Cores = coreInfoes

	var recentScore ScoreRecord
	if recentScore, err = getMostRecentScore(userID); err != nil {
		return nil, err
	}
	info.RecentScore = []ScoreRecord{recentScore}

	var isAprilFools string
	if err := db.QueryRow(sqlStmtAprilfools).Scan(&isAprilFools); err != nil {
		log.Println("Error occured while reading April Fools info.")
		return nil, err
	}
	info.IsAprilFools = isAprilFools == "t"

	return info, nil
}

func getCoreInfo(userID int) ([]CoreInfo, error) {
	rows, err := db.Query(sqlStmtCoreInfo, userID)
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
		return nil, fmt.Errorf("error occured while querying core info: %w", err)
	}
	return coreInfoes, nil
}

func getMostRecentScore(userID int) (ScoreRecord, error) {
	record := ScoreRecord{}
	var modifier sql.NullInt32
	err := db.QueryRow(sqlStmtMostRecentScore, userID).Scan(
		&record.SongID, &record.Difficulty, &record.Score,
		&record.Shiny, &record.Pure, &record.Far, &record.Lost,
		&record.Health, &modifier,
		&record.ClearType, &record.BestClearType,
	)
	if err != nil {
		return record, fmt.Errorf("error occured while querying most recent score: %w", err)
	}

	if modifier.Valid {
		record.Modifier = int(modifier.Int32)
	} else {
		record.Modifier = 0
	}

	return record, nil
}

func getItemList(userID int, tableName string, targetName string) ([]string, error) {
	rows, err := db.Query(
		fmt.Sprintf(
			"select %s from %s where user_id = ?", targetName, tableName),
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
	targetPath := path.Base(r.URL.Path)
	data, err := forms.Parse(r)
	if err != nil {
		log.Println(err)
	}

	var userID int
	if NeedAuth {
		userID, err = verifyBearerAuth(r.Header.Get("Authorization"))
		if err != nil {
			c := Container{false, nil, 203}
			http.Error(w, c.toJSON(), http.StatusUnauthorized)
			return
		}
	} else {
		userID = staticUserID
	}

	val := data.Validator()
	val.Require("value")
	if val.HasErrors() {
		log.Println("Immproper setting request with no field in form")
		for k, v := range val.ErrorMap() {
			log.Println(k, v)
		}
		return
	}
	target, ok := SettingMap[targetPath]
	if !ok {
		log.Printf(
			"Unknow setting option: `%s`\n been passed to /user/me/setting",
			r.URL.Path,
		)
		return
	}
	if target == "favorite_partner" {
		partID := data.GetInt("value")
		err = changeFavouritePartner(userID, partID)
	} else {
		value := data.GetBool("value")
		err = changeSetting(userID, target, value)
	}
	if err != nil {
		log.Println(err)
	}
	tojson, err := getUserInfo(userID, r)
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
	if _, err := db.Exec(
		fmt.Sprintf(sqlStmtUserSetting, target, value, userID),
	); err != nil {
		return fmt.Errorf(
			"Error occured while modifying PLAYER for setting `%s` to `%v` with userID = %d: %w",
			target, value, userID, err,
		)
	}

	return nil
}

func changeFavouritePartner(userID int, partID int) error {
	_, err := db.Exec(
		fmt.Sprintf(sqlStmtFavouritePartner, partID, userID),
	)
	if err != nil {
		return fmt.Errorf(
			"Error occured while modifying PLAYER for setting `favorite_partner` to `%v` with userID = %d: %w",
			partID, userID, err,
		)
	}

	return nil
}
