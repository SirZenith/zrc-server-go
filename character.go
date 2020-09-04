package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/albrow/forms"
)

func init() {
	R.Path(path.Join(APIRoot, "user/me/character")).Handler(
		http.HandlerFunc(changeCharacter),
	)
	R.PathPrefix(path.Join(APIRoot, "user/me/characters/{partID}/toggle_uncap")).Methods("POST").Handler(
		http.HandlerFunc(toggleUncap),
	)
}

func getCharacterStats(userID int) ([]CharacterStats, error) {
	var (
		isUncappedOverride string
		isUncapped         string
		exp                float64
		overDrive          float64
		prog               float64
		frag               float64
		level              int8
		levelExp           int
		partID             int8
		progTempest        float64
		skillID            string
		skillIDUncap       string
		charType           int8
		skillRequiresUncap string
		skillUnlockLevel   int8
		partName           string
		frag1              float64
		frag20             float64
		prog1              float64
		prog20             float64
		overdrive1         float64
		overdrive20        float64
	)
	rows, err := db.Query(`select
			part_id,
			ifnull(is_uncapped_override, ''), ifnull(is_uncapped, ''),
			overdrive, prog, frag, prog_tempest,
			part_stats.lv,
			part_stats.exp_val,
			level_exp.exp_val as "Level Exp"
		from
			part_stats, level_exp
		where
			part_stats.user_id = ?
			and part_stats.lv = level_exp.lv`,
		userID,
	)
	if err != nil {
		log.Println("Error occured while querying table PART_STATS for all chatacters' stats")
		return nil, err
	}
	defer rows.Close()

	stmtStr := `select 
			ifnull(skill_id, ''), ifnull(skill_id_uncap, ''), char_type, 
			ifnull(skill_requires_uncap, ''), skill_unlock_level, part_name,
			frag_1, frag_20, prog_1, prog_20, overdrive_1, overdrive_20
		from partner
		where part_id = ?`
	stmt, err := db.Prepare(stmtStr)
	if err != nil {
		log.Println("Error occured while preparing statement ", stmtStr)
		return nil, err
	}

	statses := []CharacterStats{}
	for rows.Next() {
		rows.Scan(
			&partID,
			&isUncappedOverride, &isUncapped,
			&overDrive, &prog, &frag, &progTempest,
			&level, &exp, &levelExp,
		)
		err := stmt.QueryRow(partID).Scan(
			&skillID, &skillIDUncap, &charType,
			&skillRequiresUncap, &skillUnlockLevel, &partName,
			&frag1, &frag20, &prog1, &prog20, &overdrive1, &overdrive20,
		)
		if err != nil {
			log.Println("Error occured while using prepared statement ", stmtStr)
			return nil, err
		}

		stats := CharacterStats{
			isUncappedOverride == "t",
			isUncapped == "t",
			[]string{},
			charType,
			skillIDUncap,
			skillRequiresUncap == "t",
			skillUnlockLevel,
			skillID,
			overdrive20,
			prog20,
			frag20,
			levelExp,
			exp,
			level,
			partName,
			partID,
			progTempest,
		}
		statses = append(statses, stats)
	}
	return statses, nil
}

func getSingleCharacterStats(userID int, partID int8) (*CharacterStats, error) {
	var (
		isUncappedOverride string
		isUncapped         string
		exp                float64
		overDrive          float64
		prog               float64
		frag               float64
		level              int8
		levelExp           int
		progTempest        float64
		skillID            string
		skillIDUncap       string
		charType           int8
		skillRequiresUncap string
		skillUnlockLevel   int8
		partName           string
		frag1              float64
		frag20             float64
		prog1              float64
		prog20             float64
		overdrive1         float64
		overdrive20        float64
	)
	err := db.QueryRow(`select
			ifnull(is_uncapped_override, ''), ifnull(is_uncapped, ''),
			overdrive, prog, frag, prog_tempest,
			part_stats.lv, part_stats.exp_val,
			level_exp.exp_val as "Level Exp",
			partner.part_id, partner.skill_id,
			ifnull(partner.skill_id_uncap, ''), partner.char_type,
			ifnull(partner.skill_requires_uncap, ''),
			partner.skill_unlock_level,
			partner.part_name,
			partner.frag_1, partner.frag_20,
			partner.prog_1, partner.prog_20,
			partner.overdrive_1, partner.overdrive_20
		from
			part_stats, level_exp, partner
		where
			part_stats.user_id = ?1
			and part_stats.part_id = ?2
			and partner.part_id = ?2
			and part_stats.lv = level_exp.lv`,
		userID, partID,
	).Scan(
		&isUncappedOverride, &isUncapped,
		&overDrive, &prog, &frag, &progTempest,
		&level, &exp, &levelExp,
		&partID, &skillID, &skillIDUncap, &charType,
		&skillRequiresUncap, &skillUnlockLevel, &partName,
		&frag1, &frag20, &prog1, &prog20, &overdrive1, &overdrive20,
	)
	if err != nil {
		log.Println("Error occured while querying table PART_STATS for single character")
		log.Printf("userID = %d, partID = %d", userID, partID)
		return nil, err
	}

	stats := CharacterStats{
		isUncappedOverride == "t",
		isUncapped == "t",
		[]string{},
		charType,
		skillIDUncap,
		skillRequiresUncap == "t",
		skillUnlockLevel,
		skillID,
		overDrive,
		prog,
		frag,
		levelExp,
		exp,
		level,
		partName,
		partID,
		progTempest,
	}

	return &stats, nil
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
	fmt.Printf("changing character: %s, skill: %s\n", character, skillSealed)

	if val, err := strconv.ParseBool(skillSealed); err != nil {
		skillSealed = ""
	} else if val {
		skillSealed = "t"
	} else {
		skillSealed = ""
	}

	_, err = db.Exec(`update player
		set partner = ?1, is_skill_sealed = ?2
		where user_id = ?3`, character, skillSealed, userID)
	if err != nil {
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
	partStr := path.Base(path.Dir(r.URL.Path))
	partID, err := strconv.Atoi(partStr)
	if err != nil {
		log.Println("Error character ID for toggle uncap partID = ", partID)
		log.Println(err)
		container.Success = false
	} else if _, err = db.Exec(`update part_stats set is_uncapped_override =
	case when is_uncapped_override = 't' then 'f'
		  else 't'
	end
	where part_id = ?`, partID); err != nil {
		log.Println("Error occured while modifying uncap toggle state in table.")
		log.Println(err)
		container.Success = false
	} else if stats, err := getSingleCharacterStats(userID, int8(partID)); err != nil {
		log.Println(err)
		container.Success = false
	} else {
		container.Value = &ToggleResult{
			userID, []*CharacterStats{stats},
		}
	}
	fmt.Fprint(w, container.toJSON())
}
