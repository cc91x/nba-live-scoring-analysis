package helpers

import "go.mongodb.org/mongo-driver/bson"

/* Config file */
type NbaConfig struct {
	Database struct {
		Schema string `yaml:"schema"`
		Host   string `yaml:"host"`
		Port   string `yaml:"port"`
	} `yaml:"database"`
	OddsApi struct {
		BaseUrl string `yaml:"baseUrl"`
		Key     string `yaml:"key"`
	} `yaml:"oddsApi"`
}

/* Raw game in DB */
type RawNbaGame struct {
	Resource       string     `bson:"resource"`
	Parameters     Parameters `bson:"parameters"`
	PlayByPlayRows bson.A     `bson:"rawPlayByPlay"`
	Date           string     `bson:"date"`
	Matchup        string     `bson:"matchup"`
	SeasonId       string     `bson:"seasonId"`
}

type Parameters struct {
	GameId     string `bson:"GameID"`
	StarPeriod int32  `bson:"StartPeriod"`
	EndPeriod  int32  `bson:"EndPeriod"`
}

type RawPlay struct {
	EstTime       string
	GameClockTime string
	Quarter       int32
	Score         string
}

/* Cleaned game, after processing */
type CleanedGame struct {
	GameId     string       `bson:"gameId"`
	Date       string       `bson:"date"`
	StartTime  string       `bson:"startTime"`
	AwayTeamId string       `bson:"awayTeamId"`
	HomeTeamId string       `bson:"homeTeamId"`
	PlayByPlay []PlayByPlay `bson:"playByPlay"`
	SeasonId   string       `bson:"seasonId"`
}

type PlayByPlay struct {
	SecondsElapsed int32
	AwayScore      int
	HomeScore      int
}

/* Team metadata in DB */
type TeamMetadata struct {
	TeamId          int    `bson:"teamId"`
	TeamName        string `bson:"teamName"`
	TeamAbbreviaton string `bson:"teamAbbreviation"`
}

/* Response from odds source API */
type RawOddsResponse struct {
	Timestamp         string     `json:"timestamp" bson:"timestamp"`
	PreviousTimestamp string     `json:"previous_timestamp" bson:"previous_timestamp"`
	NextTimestamp     string     `json:"next_timestamp" bson:"next_timestamp"`
	Data              []OddsData `json:"data" bson:"data"`
	Date              string     `json:"date" bson:"date"`
	UtcHour           int        `json:"utcHour" bson:"utcHour"`
}

type OddsData struct {
	Id           string      `json:"id" bson:"id"`
	SportKey     string      `json:"sport_key" bson:"sport_key"`
	SportTitle   string      `json:"sport_title" bson:"sport_title"`
	CommenceTime string      `json:"commence_time" bson:"commence_time"`
	HomeTeam     string      `json:"home_team" bson:"home_team"`
	AwayTeam     string      `json:"away_team" bson:"away_team"`
	Bookmakers   []Bookmaker `json:"bookmakers" bson:"bookmakers"`
}

type Bookmaker struct {
	Key        string   `json:"key" bson:"key"`
	Title      string   `json:"title" bson:"title"`
	LastUpdate string   `json:"last_update" bson:"last_update"`
	Markets    []Market `json:"markets" bson:"markets"`
}

type Market struct {
	Key        string    `json:"key" bson:"key" `
	LastUpdate string    `json:"last_update" bson:"last_update" `
	Outcome    []Outcome `json:"outcomes" bson:"outcomes" `
}

type Outcome struct {
	Name  string  `json:"name" bson:"name"`
	Price float64 `json:"price" bson:"price"`
	Point float64 `json:"point" bson:"point"`
}

/* Cleaned odds data, after processing */
type CleanedOdds struct {
	GameId      string      `bson:"gameId"`
	Bookmaker   string      `bson:"bookmaker"`
	MoneyLine   MoneyLine   `bson:"moneyLine"`
	PointSpread PointSpread `bson:"pointSpread"`
	Total       Total       `bson:"total"`
}

type Total struct {
	Total      float32 `bson:"total"`
	OverPrice  float32 `bson:"overPrice"`
	UnderPrice float32 `bson:"underPrice"`
}

type MoneyLine struct {
	AwayPrice float32 `bson:"awayPrice"`
	HomePrice float32 `bson:"homePrice"`
}

type PointSpread struct {
	AwaySpread float32 `bson:"awaySpread"`
	HomeSpread float32 `bson:"homeSpread"`
	AwayPrice  float32 `bson:"awayPrice"`
	HomePrice  float32 `bson:"homePrice"`
}

/* CSV columns */
type GameCsv struct {
	GameId               string
	SeasonId             string
	Date                 string
	StartTime            string
	AwayTeamAbbreviation string
	AwayTeamId           string
	HomeTeamAbbreviation string
	HomeTeamId           string
	AwayMl               string
	HomeMl               string
	AwaySpread           string
	HomeSpread           string
	AwayFinalScore       string
	HomeFinalScore       string
}

type PlayByPlayCsv struct {
	GameId         string
	SecondsElapsed string
	AwayScore      string
	HomeScore      string
	UnderdogScore  string
	FavoriteScore  string
}
