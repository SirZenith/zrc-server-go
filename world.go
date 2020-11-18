package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

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
		isBeyond     string
		isLegacy     string
		isLocked     string
		isRepeatable string
		mapID        string
		rewards      []Reward
	)

	rows, err := db.Query(sqlStmtMapInfo, userID)
	if err != nil {
		log.Println("Error occured while querying table WORLD_MAP.")
		return nil, err
	}
	defer rows.Close()

	info := new(MapInfo)
	infoes := []MapInfo{}
	for rows.Next() {
		rows.Scan(
			&info.AvailableFrom, &info.AvailableTo,
			&info.BeyondHealth,
			&info.Chapter,
			&info.Coordinate,
			&info.CustomBG,
			&isBeyond,
			&isLegacy,
			&isRepeatable,
			&info.MapID,
			&info.RequireID,
			&info.RequireType,
			&info.RequireValue,
			&info.StamCost,
			&info.StepCount,
			&info.CurrCapture,
			&info.CurrPosition,
			&isLocked,
		)
		info.IsBeyond = isBeyond == "t"
		info.IsLegacy = isLegacy == "t"
		info.IsLocked = isLocked == "t"
		info.IsRepeatable = isRepeatable == "t"

		info.PartAffinity, info.AffMultiplier, err = getMapAffinity(mapID)
		if err != nil {
			return nil, err
		}

		rewards, err = getRewards(mapID)
		if err != nil {
			return nil, err
		}
		info.Rewards = rewards

		infoes = append(infoes, *info)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error occured while reading map info: %w", err)
	}

	var currMap string
	err = db.QueryRow(sqlStmtCurrentMap, userID).Scan(&currMap)
	if err != nil {
		return nil, fmt.Errorf("error occur while querying current map for user = %d: %w", userID, err)
	}

	return &MapInfoContainer{userID, currMap, infoes}, nil
}

func getMapAffinity(mapID string) ([]int8, []float64, error) {
	partners, multipliers := []int8{}, []float64{}
	rows, err := db.Query(sqlStmtMapAffinity, mapID)
	if err != nil {
		return nil, nil, fmt.Errorf("error occured while querying map affinity for map = %s: %w", mapID, err)
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
		return nil, nil, fmt.Errorf("Error occured while reading rows queried from MAP_AFFINITY: %w", err)
	}

	return partners, multipliers, nil
}

func getRewards(mapID string) ([]Reward, error) {
	rows, err := db.Query(sqlStmtRewards, mapID)
	if err != nil {
		return nil, fmt.Errorf("Error occured while querying table MAP_REWARD with MAP_ID = %s: %w", mapID, err)
	}
	defer rows.Close()

	rewards := []Reward{}
	var (
		position int
		amount   sql.NullInt32
	)
	item := new(RewardItem)
	for rows.Next() {
		rows.Scan(&item.ItemID, &item.ItemType, &amount, &position)
		item.Amount = amount.Int32
		rewards = append(rewards, Reward{
			Items:    []RewardItem{*item},
			Position: position,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"Error occured while reading rows queried from tabletable MAP_REWARD with MAP_ID = %s: %w", mapID, err)
	}

	return rewards, nil
}

// func getRewards(mapID string) ([]Reward, error) {
// 	rows, err := db.Query(
// 		"select reward_id, item_type, amount, position from map_reward where map_id = ?",
// 		mapID,
// 	)
// 	if err != nil {
// 		log.Println("Error occured while querying table MAP_REWARD with MAP_ID = ", mapID)
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	rewards := []Reward{}
// 	var (
// 		rewardID   string
// 		rewardType string
// 		amount     sql.NullInt32
// 		position   int
// 	)
// 	for rows.Next() {
// 		rows.Scan(&rewardID, &rewardType, &amount, &position)
// 		rewards = append(rewards, Reward{
// 			Items: []RewardItem{
// 				{rewardType, rewardID, amount.Int32},
// 			},
// 			Position: position,
// 		})
// 	}

// 	if err = rows.Err(); err != nil {
// 		log.Println(
// 			"Error occured while reading rows queried from tabletable MAP_REWARD with MAP_ID = ", mapID)
// 		return nil, err
// 	}

// 	return rewards, nil
// }
