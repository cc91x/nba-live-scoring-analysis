package helpers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var timezoneEstName string = "America/New_York"
var timezoneUtcName string = "UTC"
var amSuffix string = ":00 AM"

var moneylineKey string = "h2h"
var spreadKey string = "spreads"
var totalKey string = "totals"
var overOutcome string = "Over"

func CleanOdds(date string) (err error) {
	client, err := loadMongoDbClient(*Config)
	if err != nil {
		return err
	}
	defer func() {
		if err6 := closeMongoDBConnection(client, err); err6 != nil {
			err = err6
		}
	}()

	rawOddsCollection := getHistoricalOddscollection(client, Config.Database.Schema)
	cleanedOddsCollection := getCleanedOddsCollection(client, Config.Database.Schema)
	cleanedGamesCollection := getCleanedGamesCollection(client, Config.Database.Schema)
	teamMetadataCollection := getTeamMetadataCollection(client, Config.Database.Schema)

	teamIdsToNamesMap, err1 := fetchTeamNameToIds(teamMetadataCollection)
	gamesOnDate, err2 := findCleanedGame(date, cleanedGamesCollection)
	if err1 != nil || err2 != nil {
		return handleMultipleErrors(err1, err2)
	}

	var cleanedOdds = make([]CleanedOdds, 0, len(gamesOnDate))
	for _, game := range gamesOnDate {
		utcHour, err3 := determineLatestHourBeforeGame(game)
		rawOdds, err4 := findRawOdds(utcHour, game, teamIdsToNamesMap, rawOddsCollection)
		cleanedOdd, err5 := cleanOddsEntry(rawOdds, game)

		if err3 != nil || err4 != nil || err5 != nil {
			return handleMultipleErrors(err3, err4, err5)
		}
		cleanedOdds = append(cleanedOdds, *cleanedOdd)
	}

	return upsertGameOdds(cleanedOdds, cleanedOddsCollection)
}

func fetchTeamNameToIds(dbCollection *mongo.Collection) (teamNameToIds map[string]string, err error) {
	teamMetadata, err := findTeamMetadata(dbCollection)
	if err != nil {
		return nil, err
	}
	teamNameToIds = make(map[string]string)
	for _, result := range teamMetadata {
		teamNameToIds[strconv.Itoa(result.TeamId)] = result.TeamName
	}
	return teamNameToIds, nil
}

func determineLatestHourBeforeGame(game CleanedGame) (latestHour int, err error) {
	gameStartTime, err1 := convertDateTimeToStandard(game.StartTime, game.Date, timezoneEstName)
	for _, utcHour := range utcHoursForLookup {
		clockTime := strconv.Itoa(utcHour) + amSuffix
		utcStartTime, err2 := convertDateTimeToStandard(clockTime, game.Date, timezoneUtcName)

		if err1 != nil || err2 != nil {
			return 0, handleMultipleErrors(err1, err2)
		}
		if gameStartTime.Sub(*utcStartTime).Minutes() > 0 {
			latestHour = utcHour
		}
	}
	return latestHour, nil
}

func extractTimeUnits(clockTime string) (minute int, hour int, err error) {
	if len(clockTime) == 7 {
		clockTime = "0" + clockTime
	}
	if !isValidClockTimeString(clockTime) {
		return 0, 0, errors.New("invalid clocktime string")
	}

	minute, err1 := strconv.Atoi(clockTime[3:5])
	hour, err2 := strconv.Atoi(clockTime[:2])
	if err1 != nil || err2 != nil {
		return 0, 0, handleMultipleErrors(err1, err2)
	}
	if clockTime[6:] == "PM" {
		hour += 12
	}
	return minute, hour, nil
}

func extractDateUnits(date string) (year int, month int, day int, err error) {
	if !isValidDateString(date) {
		return 0, 0, 0, errors.New("invalid date string")
	}

	vals := strings.Split(date, "-")
	year, err1 := strconv.Atoi(vals[0])
	month, err2 := strconv.Atoi(vals[1])
	day, err3 := strconv.Atoi(vals[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, handleMultipleErrors(err1, err2, err3)
	}
	return year, month, day, nil
}

func convertDateTimeToStandard(clockTime string, date string, timezone string) (*time.Time, error) {
	minute, hour, err1 := extractTimeUnits(clockTime)
	year, month, day, err2 := extractDateUnits(date)
	loc, err3 := time.LoadLocation(timezone)

	if err1 != nil || err2 != nil || err3 != nil {
		return nil, handleMultipleErrors(err1, err2, err3)
	}
	standardTime := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)
	return &standardTime, nil
}

func findRawOdds(utcHour int, game CleanedGame, teamIdsToNamesMap map[string]string, dbCollection *mongo.Collection) (oddsData OddsData, err error) {
	awayTeamName, ok1 := teamIdsToNamesMap[game.AwayTeamId]
	homeTeamName, ok2 := teamIdsToNamesMap[game.HomeTeamId]

	var rawOdds RawOddsResponse
	err1 := dbCollection.FindOne(context.TODO(), rawOddsDbFilter(game.Date, utcHour)).Decode(&rawOdds)
	if err1 != nil || !ok1 || !ok2 {
		return OddsData{}, errors.New("could not find any games for this date and time")
	}

	for _, rawOddsGame := range rawOdds.Data {
		if rawOddsGame.AwayTeam == awayTeamName && rawOddsGame.HomeTeam == homeTeamName {
			return rawOddsGame, nil
		}
	}
	return OddsData{}, errors.New("could not find odds for this game")
}

func cleanOddsEntry(odds OddsData, game CleanedGame) (cleanedOdds *CleanedOdds, err error) {
	validBooks := filterAndOrderBookmakers(odds.Bookmakers, bookmakersPriority)
	for _, bookmaker := range validBooks {
		if len(bookmaker.Markets) == 3 {
			ml, spread, total := extractOdds(bookmaker, odds.AwayTeam)
			return &CleanedOdds{
				GameId:      game.GameId,
				Bookmaker:   bookmaker.Key,
				MoneyLine:   ml,
				PointSpread: spread,
				Total:       total,
			}, nil
		}
	}
	return nil, errors.New("could not find valid bookmaker")
}

func filterAndOrderBookmakers(allBooks []Bookmaker, bookmakersPriority map[string]int) []Bookmaker {
	var validBooks = make([]Bookmaker, 0, len(bookmakersPriority))
	for _, book := range allBooks {
		if _, ok := bookmakersPriority[book.Key]; ok {
			validBooks = append(validBooks, book)
		}
	}
	sort.Slice(validBooks, func(i, j int) bool {
		return bookmakersPriority[validBooks[i].Key] < bookmakersPriority[validBooks[j].Key]
	})
	return validBooks
}

func extractOdds(bookmaker Bookmaker, awayTeam string) (ml MoneyLine, spread PointSpread, total Total) {
	for _, market := range bookmaker.Markets {
		switch market.Key {
		case moneylineKey:
			ml = createMoneyLine(market, awayTeam)
		case spreadKey:
			spread = createSpread(market, awayTeam)
		case totalKey:
			total = createTotal(market)
		default:
			fmt.Printf("Unknown market found: %s. Skipping", market.Key)
		}
	}
	return ml, spread, total
}

func createMoneyLine(market Market, awayTeam string) (ml MoneyLine) {
	for _, outcome := range market.Outcome {
		if outcome.Name == awayTeam {
			ml.AwayPrice = float32(outcome.Price)
		} else {
			ml.HomePrice = float32(outcome.Price)
		}
	}
	return ml
}

func createSpread(market Market, awayTeam string) (spread PointSpread) {
	for _, outcome := range market.Outcome {
		if outcome.Name == awayTeam {
			spread.AwayPrice = float32(outcome.Price)
			spread.AwaySpread = float32(outcome.Point)
		} else {
			spread.HomePrice = float32(outcome.Price)
			spread.HomeSpread = float32(outcome.Point)
		}
	}
	return spread
}

func createTotal(market Market) (total Total) {
	for _, outcome := range market.Outcome {
		if outcome.Name == overOutcome {
			total.OverPrice = float32(outcome.Price)
		} else {
			total.UnderPrice = float32(outcome.Price)
		}
		total.Total = float32(outcome.Point)
	}
	return total
}

func upsertGameOdds(odds []CleanedOdds, dbCollection *mongo.Collection) (err error) {
	var operations = make([]mongo.WriteModel, 0, len(odds))
	for _, game := range odds {
		operations = append(operations, mongo.NewUpdateOneModel().
			SetFilter(cleanedOddsGameFilter(game)).
			SetUpdate(bson.M{"$set": game}).
			SetUpsert(true))
	}
	_, err = upsertItemsGeneric(operations, dbCollection)
	return err
}

func isValidClockTimeString(clocktime string) bool {
	return len(clocktime) == 8 && clocktime[2] == ':' && (clocktime[5:] == " AM" || clocktime[5:] == " PM")
}

func isValidDateString(date string) bool {
	return len(date) == 10 && date[4] == '-' && date[7] == '-'
}

func cleanedOddsGameFilter(odds CleanedOdds) bson.M {
	return bson.M{"game": odds.GameId}
}
