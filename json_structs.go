package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/albrow/forms"
	"github.com/dgrijalva/jwt-go"
)

// ToJSON interface for json info purpose structs
type ToJSON interface {
	toJSON() string
}

// EmptyList type define
type EmptyList []struct{}

func (c *EmptyList) toJSON() string {
	res, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// EmptyMap type define
type EmptyMap []struct{}

func (c *EmptyMap) toJSON() string {
	res, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// Container universal container
type Container struct {
	Success   bool   `json:"success"`
	Value     ToJSON `json:"value,omitempty"`
	ErrorCode int    `json:"error_code,omitempty"`
}

func (c *Container) toJSON() string {
	res, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// Section: Login
// ============================================================================

type userClaims struct {
	UserID int `json:"name"`
	jwt.StandardClaims
}

// LoginToken contain token for login
type LoginToken struct {
	Token     string `json:"access_token"`
	Type      string `json:"token_type"`
	Success   bool   `json:"success"`
	ErrorCode int    `json:"error_code,omitempty"`
}

// AggCall represent a call pass to /compose/aggregate
type AggCall struct {
	ID       int8   `json:"id"`
	EndPoint string `json:"endpoint"`
}

// AggResult is result of one call to /compose/aggregate
type AggResult struct {
	ID    int8   `json:"id"`
	Value ToJSON `json:"value,omitempty"`
}

// AggContainer container for AggResult
type AggContainer []AggResult

func (c *AggContainer) toJSON() string {
	res, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// Section: User Info
// ============================================================================

// UserInfo return from server when request by cilent
type UserInfo struct {
	IsAprilFools          bool             `json:"is_aprilfools"`
	CurrAvailableMaps     []string         `json:"curr_available_maps"`
	CharacterStats        []CharacterStats `json:"character_stats"`
	Friends               []string         `json:"friends"`
	Settings              Setting          `json:"settings"`
	UserID                int              `json:"user_id"`
	Name                  string           `json:"name"`
	DisplaName            string           `json:"display_name"`
	UserCode              string           `json:"user_code"`
	Ticket                int              `json:"ticket"`
	PartID                int8             `json:"character"`
	IsLockedNameDuplicate bool             `json:"is_locked_name_duplicated"`
	IsSkillSealed         bool             `json:"is_skill_sealed"`
	CurrentMap            string           `json:"current_map"`
	ProgBoost             int8             `json:"prog_boost"`
	NextFragstamTs        int64            `json:"next_fragstam_ts"`
	MaxStaminaTs          int64            `json:"max_stamina_ts"`
	Stamina               int8             `json:"stamina"`
	WorldUnlocks          []string         `json:"world_unlocks"`
	WorldSongs            []string         `json:"world_songs"`
	Singles               []string         `json:"singles"`
	Packs                 []string         `json:"packs"`
	Characters            []int8           `json:"characters"`
	Cores                 []CoreInfo       `json:"cores"`
	RecentScore           []ScoreRecord    `json:"recent_score"`
	MaxFriend             int8             `json:"max_friend"`
	Rating                int              `json:"rating"`
	JoinDate              int64            `json:"join_date"`
}

func (info *UserInfo) toJSON() string {
	res, err := json.Marshal(info)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// CharacterStats store status of a partner
type CharacterStats struct {
	Voice              []int    `json:"voice,omitempty"`
	IsUncappedOverride bool     `json:"is_uncapped_override"`
	IsUncapped         bool     `json:"is_uncapped"`
	UncapCores         []string `json:"uncap_cores"`
	CharType           int8     `json:"char_type"`
	SkillIDUncap       string   `json:"skill_id_uncap"`
	SkillRequiresUncap bool     `json:"skill_requires_uncap"`
	SkillUnlockLevel   int8     `json:"skill_unlock_level"`
	SkillID            string   `json:"skill_id"`
	Overdrive          float64  `json:"overdrive"`
	Prog               float64  `json:"prog"`
	Frag               float64  `json:"frag"`
	LevelExp           int      `json:"level_exp"`
	Exp                float64  `json:"exp"`
	Level              int8     `json:"level"`
	PartName           string   `json:"name"`
	PartID             int8     `json:"character_id"`
	ProgTempest        float64  `json:"prog_tempest,omitempty"`
}

// ToggleResult is result return when request passed to /user/me/toggle/character
type ToggleResult struct {
	UserID    int              `json:"user_id"`
	Character []CharacterStats `json:"character"`
}

func (r *ToggleResult) toJSON() string {
	res, err := json.Marshal(r)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// CoreInfo recording how many core does a player have
type CoreInfo struct {
	CoreType string `json:"core_type"`
	Amount   int8   `json:"amount"`
	ID       string `json:"_id"`
}

// Setting store player settings
type Setting struct {
	StaminaNotification bool `json:"max_stamina_notification_enabled"`
	HideRating          bool `json:"is_hide_rating"`
	FavoriteCharacter   int8 `json:"favorite_character"`
}

// Seciton: Pack Info
// ============================================================================

// PackInfoContainer type define
type PackInfoContainer []PackInfo

func (c *PackInfoContainer) toJSON() string {
	res, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// PackInfo represent a pack
type PackInfo struct {
	Name         string     `json:"name"`
	Items        []PackItem `json:"items"`
	Price        int        `json:"price"`
	OrigPrice    int        `josn:"price"`
	DiscountFrom int64      `json:"discount_from"`
	DiscountTo   int64      `json:"discount_to"`
}

// PackItem represent a pack item in pack
type PackItem struct {
	ID          string `json:"id"`
	ItemType    string `json:"type"`
	IsAvailable bool   `json:"is_available"`
}

// Seciton: Checksum
// ============================================================================

// CheckSumContainer container for CheckSum
type CheckSumContainer map[string]*Checksum

func (c *CheckSumContainer) toJSON() string {
	res, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// Checksum record checksum of a song and its
type Checksum struct {
	Audio map[string]string            `json:"audio,omitempty"`
	Chart map[string]map[string]string `json:"chart,omitempty"`
}

// Seciton: Game Info
// ============================================================================

// GameInfo recording current game info from server
type GameInfo struct {
	MaxStam             int8        `json:"max_stamina"`
	StaminaRecoverTick  int         `json:"stamina_recover_tick"`
	CoreExp             int         `json:"core_exp"`
	Now                 int64       `json:"curr_ts"`
	LevelSteps          []levelStep `json:"level_steps"`
	WorldRankingEnabled bool        `json:"world_ranking_enabled"`
	BydUnlocked         bool        `json:"is_byd_chapter_unlocked"`
}

type levelStep struct {
	Lv  int `json:"level"`
	Exp int `json:"level_exp"`
}

func (i *GameInfo) toJSON() string {
	res, err := json.Marshal(i)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// Seciton: Map Info
// ============================================================================

// MapInfoContainer is simple type wrapper
type MapInfoContainer struct {
	UserID  int       `json:"user_id"`
	CurrMap string    `json:"current_map"`
	Maps    []MapInfo `json:"maps"`
}

func (c *MapInfoContainer) toJSON() string {
	res, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// MapInfo contain info of map
type MapInfo struct {
	AffMultiplier []float64 `json:"affinity_multiplier"`
	AvailableFrom int64     `json:"available_from"`
	AvailableTo   int64     `json:"available_to"`
	BeyondHealth  int8      `json:"beyond_health"`
	PartAffinity  []int8    `json:"character_affinity"`
	Chapter       int       `json:"chapter"`
	Coordinate    string    `json:"coordinate"`
	CurrCapture   int       `json:"curr_capture"`
	CurrPosition  int       `json:"curr_position"`
	CustomBG      string    `json:"custom_bg"`
	IsBeyond      bool      `json:"is_beyond"`
	IsLegacy      bool      `json:"is_legacy"`
	IsLocked      bool      `json:"is_locked"`
	IsRepeatable  bool      `json:"is_repeatable"`
	MapID         string    `json:"map_id"`
	RequireID     string    `json:"require_id"`
	RequireType   string    `json:"require_type"`
	RequireValue  int       `json:"require_value"`
	StamCost      int       `json:"stamina_cost"`
	StepCount     int       `json:"step_count"`
	Rewards       []Reward  `json:"rewards"`
}

// Reward is reward in world map
type Reward struct {
	Items    []RewardItem `json:"items,omitepmty"`
	Position int          `json:"position"`
}

// RewardItem is item inside Reward struct
type RewardItem struct {
	ItemType string `json:"type"`
	ItemID   string `json:"id,omitempty"`
	Amount   int32  `json:"amount,omitempty"`
}

// Seciton: Score
// ============================================================================

// ScoreToken is token used for upload score
type ScoreToken struct {
	Token string `json:"token"`
}

func (t *ScoreToken) toJSON() string {
	res, err := json.Marshal(t)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(res)
}

// ScoreRecord represent score of a paly result
type ScoreRecord struct {
	SongID        string  `json:"song_id"`
	Difficulty    int8    `json:"difficulty"`
	Rating        float64 `json:"rating,omitempty"`
	Score         int     `json:"score"`
	Shiny         int     `json:"shiny_perfect_count"`
	Pure          int     `json:"perfect_count"`
	Far           int     `json:"near_count"`
	Lost          int     `json:"miss_count"`
	Health        int8    `json:"health"`
	TimePlayed    int64   `json:"time_played"`
	Modifier      int     `json:"modifier"`
	BeyondGague   int8    `json:"beyond_gague,omitempty"`
	ClearType     int8    `json:"clear_type"`
	BestClearType int8    `json:"best_clear_type,omitempty"`
}

func scoreRecordFromForm(data *forms.Data) *ScoreRecord {
	return &ScoreRecord{
		SongID:     data.Get("song_id"),
		Difficulty: int8(data.GetInt("difficulty")),
		Score:      data.GetInt("score"),
		Shiny:      data.GetInt("shiny_perfect_count"),
		Pure:       data.GetInt("perfect_count"),
		Far:        data.GetInt("near_count"),
		Lost:       data.GetInt("miss_count"),
		Health:     int8(data.GetInt("health")),
		ClearType:  int8(data.GetInt("clear_type")),
	}
}

func (r *ScoreRecord) scoreToRating() error {
	r.Rating = 0.0
	score := float64(r.Score)
	var baseRating float64
	err := db.QueryRow(sqlStmtBaseRating, r.SongID, r.Difficulty).Scan(&baseRating)
	if err != nil {
		return fmt.Errorf("error while querying base rating for `%s`: %w", r.SongID, err)
	} else if baseRating == 0 {
		return errorZeroRating
	}

	if r.Score >= 10_000_000 {
		r.Rating = baseRating + 2
	} else if r.Score >= 9_800_000 {
		r.Rating = baseRating + 1 + (score-9_800_000)/200_000
	} else if r.Rating = baseRating + (score-9_500_000)/300_000; r.Rating < 0 {
		r.Rating = 0
	}
	return nil
}

// ScoreUploadResult is resut return from server
type ScoreUploadResult struct {
	Success bool           `json:"success"`
	Value   map[string]int `json:"value,omitempty"`
}
