package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/albrow/forms"
	"github.com/gorilla/mux"
)

var voiceList = []int{0, 1, 2, 3, 100, 1000, 1001}

func getCharacterStats(userID int, partID int8) ([]CharacterStats, error) {
	cond := ""
	if partID >= 0 {
		cond = fmt.Sprintf(sqlStmtSingleCharCond, partID)
	}
	statses := []CharacterStats{}
	stmt, err := db.Prepare(sqlStmtCharStaticStats)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := db.Query(sqlStmtOwnedChar+cond, userID)
	if err != nil {
		return statses, err
	}
	defer rows.Close()

	var (
		hasVoice           int
		isUncappedOverride string
		isUncapped         string
		skillRequiresUncap string
	)
	for rows.Next() {
		stats := new(CharacterStats)
		rows.Scan(
			&stats.PartID,
			&isUncappedOverride,
			&isUncapped,
			&stats.Overdrive,
			&stats.Prog,
			&stats.Frag,
			&stats.ProgTempest,
			&stats.Level,
			&stats.Exp,
			&stats.LevelExp,
		)

		if err := stmt.QueryRow(stats.PartID).Scan(
			&hasVoice,
			&stats.SkillID,
			&stats.SkillIDUncap,
			&skillRequiresUncap,
			&stats.SkillUnlockLevel,
			&stats.PartName,
			&stats.CharType,
		); err != nil {
			return nil, err
		}

		if hasVoice != -1 {
			stats.Voice = voiceList
		}
		stats.IsUncappedOverride = isUncappedOverride == "t"
		stats.IsUncapped = isUncapped == "t"
		stats.SkillRequiresUncap = skillRequiresUncap == "t"
		stats.UncapCores = []string{}

		statses = append(statses, *stats)
	}
	if err := rows.Err(); err != nil {
		return statses, fmt.Errorf("Error occured while querying table PART_STATS for single character, userID = %d, partID = %d: %w", userID, partID, err)
	}

	return statses, nil
}

func changeCharacter(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
	}
	data, err := forms.Parse(r)
	if err != nil {
		log.Println(err)
	}

	val := data.Validator()
	val.Require("character")
	val.Require("skill_sealed")

	character := data.Get("character")
	skillSealed := data.Get("skill_sealed")

	if val, err := strconv.ParseBool(skillSealed); err == nil && val {
		skillSealed = "t"
	} else {
		skillSealed = ""
	}

	if _, err := db.Exec(sqlStmtChangeChar, character, skillSealed, userID); err != nil {
		log.Println(err)
	}
	fmt.Fprintf(
		w,
		`{"success": true,"value": {"user_id": %d, "character": %s}}`,
		userID, character,
	)
}

func toggleUncap(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
	}

	container := Container{true, nil, 0}
	partID, err := strconv.Atoi(mux.Vars(r)["partID"])
	if err != nil {
		log.Printf("Error character ID for toggle uncap partID = %d: %s\n", partID, err)
		container.Success = false
	} else if _, err = db.Exec(sqlStmtToggleUncap, partID); err != nil {
		log.Printf("Error occured while modifying uncap toggle state in table: %s\n", err)
		container.Success = false
	} else if stats, err := getCharacterStats(userID, int8(partID)); err != nil {
		log.Println(err)
		container.Success = false
	} else {
		container.Value = &ToggleResult{userID, stats}
	}
	fmt.Fprint(w, container.toJSON())
}
