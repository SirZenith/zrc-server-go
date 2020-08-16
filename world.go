package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path"
)

func init() {
	R.Handle(
		path.Join(APIRoot, APIVer, "world/map/me"),
		http.HandlerFunc(myMapInfoHandler),
	)
	InsideHandler[path.Join(APIRoot, APIVer, "world/map/me")] = getMyMapInfo
}

func myMapInfoHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
	}
	tojson, err := getMyMapInfo(userID, r)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Fprint(w, tojson.toJSON())
	}
}

func getMyMapInfo(userID int, _ *http.Request) (ToJSON, error) {
	var (
		affMultiplier []float64
		availableFrom int64
		availableTo   int64
		beyondHealth  int8
		partAffinity  []int8
		chapter       int
		coordinate    string
		currCapture   int
		currPosition  int
		customBG      string
		isBeyond      string
		isLegacy      string
		isLocked      string
		isRepeatable  string
		mapID         string
		requireID     string
		requireType   string
		requireValue  int
		stamCost      int
		stepCount     int
		rewards       []Reward
	)

	rows, err := db.Query(`select
			available_from, available_to, beyond_health, chapter, coordinate,
			custom_bg, is_beyond, is_legacy, is_repeatable,
			world_map.map_id,
			require_id, require_type, require_value, stamina_cost, step_count,
			curr_capture, curr_position, is_locked
		from world_map, player_map_prog
		where player_map_prog.map_id = world_map.map_id
		  and player_map_prog.user_id = :1`, userID)
	if err != nil {
		log.Println("Error occured while querying table WORLD_MAP.")
		return nil, err
	}
	defer rows.Close()

	infoes := []MapInfo{}
	for rows.Next() {
		rows.Scan(
			&availableFrom, &availableTo,
			&beyondHealth,
			&chapter,
			&coordinate,
			&customBG,
			&isBeyond, &isLegacy, &isRepeatable,
			&mapID,
			&requireID, &requireType, &requireValue,
			&stamCost, &stepCount,
			&currCapture, &currPosition, &isLocked,
		)
		partAffinity, affMultiplier, err = getMapAffinity(mapID)
		if err != nil {
			return nil, err
		}
		rewards, err = getRewards(mapID)
		if err != nil {
			return nil, err
		}

		infoes = append(infoes, MapInfo{
			affMultiplier,
			availableFrom, availableTo,
			beyondHealth,
			partAffinity,
			chapter,
			coordinate,
			currCapture, currPosition,
			customBG,
			isBeyond == "t", isLegacy == "t",
			isLocked == "t", isRepeatable == "t",
			mapID,
			requireID, requireType, requireValue,
			stamCost, stepCount,
			rewards,
		})
	}

	if err = rows.Err(); err != nil {
		log.Println("Error occured while reading queried rows from WORLD_MAP.")
		return nil, err
	}

	var currMap string
	err = db.QueryRow(
		"select curr_map from player where user_id = :1",
		userID,
	).Scan(&currMap)
	if err != nil {
		log.Println("Error occur while querying CURR_MAP from PLAYER with USER_ID = ", userID)
		return nil, err
	}

	return &MapInfoContainer{userID, currMap, infoes}, nil
}

func getMapAffinity(mapID string) ([]int8, []float64, error) {
	partners, multipliers := []int8{}, []float64{}
	rows, err := db.Query(
		"select part_id, multiplier from map_affinity where map_id = :1",
		mapID,
	)
	if err != nil {
		log.Println("Error occured while querying table MAP_AFFINITY with MAP_ID = ", mapID)
		return nil, nil, err
	}
	defer rows.Close()

	var (
		partID int8
		mul    float64
	)
	for rows.Next() {
		rows.Scan(&partID, &mul)
		partners = append(partners, partID)
		multipliers = append(multipliers, mul)
	}

	if err = rows.Err(); err != nil {
		log.Println("Error occured while reading rows queried from MAP_AFFINITY.")
		return nil, nil, err
	}

	return partners, multipliers, nil
}

func getRewards(mapID string) ([]Reward, error) {
	rows, err := db.Query(
		"select reward_id, item_type, amount, position from map_reward where map_id = :1",
		mapID,
	)
	if err != nil {
		log.Println("Error occured while querying table MAP_REWARD with MAP_ID = ", mapID)
		return nil, err
	}
	defer rows.Close()

	rewards := []Reward{}
	var (
		rewardID   string
		rewardType string
		amount     sql.NullInt32
		position   int
	)
	for rows.Next() {
		rows.Scan(&rewardID, &rewardType, &amount, &position)
		rewards = append(rewards, Reward{
			Items: []RewardItem{
				{rewardType, rewardID, amount.Int32},
			},
			Position: position,
		})
	}

	if err = rows.Err(); err != nil {
		log.Println(
			"Error occured while reading rows queried from tabletable MAP_REWARD with MAP_ID = ", mapID)
		return nil, err
	}

	return rewards, nil
}
