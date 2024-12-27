package helpers

import (
	"context"
	"encoding/csv"
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func CombineGamesAndOddsToCsv(date string) (err error) {
	client, err0 := loadMongoDbClient(*Config)
	if err0 != nil {
		return err0
	}
	defer func() {
		if err6 := closeMongoDBConnection(client, err); err6 != nil {
			err = err6
		}
	}()

	teamMetadataCollection := getTeamMetadataCollection(client, Config.Database.Schema)
	cleanedGamesCollection := getCleanedGamesCollection(client, Config.Database.Schema)
	cleanedOddsCollection := getCleanedOddsCollection(client, Config.Database.Schema)

	teamIdToAbbrev, err3 := fetchTeamIdsToAbbreviation(teamMetadataCollection)
	games, err1 := findCleanedGame(date, cleanedGamesCollection)
	gameToOdds, err2 := findOddsForGames(games, cleanedOddsCollection)
	if err1 != nil || err2 != nil || err3 != nil {
		return handleMultipleErrors(err1, err2)
	}

	gameCsvRows := make(map[string][]string)
	playsCsvRows := make(map[string][]string)
	for _, game := range games {
		gameRow := createGameCsv(game, gameToOdds[game.GameId], teamIdToAbbrev)
		gameCsvRows[gameCsvKeyFunc(gameRow)] = gameRow

		playsRow := createPlaysCsv(game, gameToOdds[game.GameId])
		for _, row := range playsRow {
			playsCsvRows[playsCsvKeyFunc(row)] = row
		}
	}

	err4 := upsertCsv(gamesCsvName, gameCsvRows, gameCsvKeyFunc)
	err5 := upsertCsv(playsCsvName, playsCsvRows, playsCsvKeyFunc)
	if err4 != nil || err5 != nil {
		return handleMultipleErrors(err4, err5)
	}
	return nil
}

func fetchTeamIdsToAbbreviation(dbCollection *mongo.Collection) (teamsToAbbrev map[string]string, err error) {
	teamMetaData, err := findTeamMetadata(dbCollection)
	if err != nil {
		return nil, err
	}
	teamsToAbbrev = make(map[string]string)
	for _, result := range teamMetaData {
		teamsToAbbrev[strconv.Itoa(result.TeamId)] = result.TeamAbbreviaton
	}
	return teamsToAbbrev, nil
}

func findOddsForGames(games []CleanedGame, dbCollection *mongo.Collection) (oddsByGame map[string]CleanedOdds, err error) {
	var odds []CleanedOdds
	cursor, err1 := dbCollection.Find(context.TODO(), cleanedGamesQueryFilter(games))
	err2 := cursor.All(context.TODO(), &odds)
	if err1 != nil || err2 != nil {
		return nil, handleMultipleErrors(err1, err2)
	}

	oddsByGame = make(map[string]CleanedOdds)
	for _, odd := range odds {
		oddsByGame[odd.GameId] = odd
	}
	return oddsByGame, nil
}

func createGameCsv(game CleanedGame, odds CleanedOdds, teamIdToAbbrev map[string]string) []string {
	awayScore, homeScore := extractFinalScore(game)
	return []string{
		game.GameId,
		game.SeasonId,
		game.Date,
		game.StartTime,
		teamIdToAbbrev[game.AwayTeamId],
		game.AwayTeamId,
		teamIdToAbbrev[game.HomeTeamId],
		game.HomeTeamId,
		strconv.FormatFloat(float64(odds.MoneyLine.AwayPrice), 'f', -2, 32),
		strconv.FormatFloat(float64(odds.MoneyLine.HomePrice), 'f', -2, 32),
		strconv.FormatFloat(float64(odds.PointSpread.AwaySpread), 'f', -1, 32),
		strconv.FormatFloat(float64(odds.PointSpread.HomeSpread), 'f', -1, 32),
		strconv.FormatFloat(float64(odds.Total.Total), 'f', -1, 32),
		strconv.Itoa(awayScore),
		strconv.Itoa(homeScore),
	}
}

func extractFinalScore(game CleanedGame) (awayScore int, homeScore int) {
	lastPlay := game.PlayByPlay[len(game.PlayByPlay)-1]
	return lastPlay.AwayScore, lastPlay.HomeScore
}

func createPlaysCsv(game CleanedGame, odds CleanedOdds) (csvRows [][]string) {
	awayIsFavored := odds.PointSpread.AwaySpread <= 0
	for _, play := range game.PlayByPlay {
		underdogScore := ternaryOperator(awayIsFavored, play.HomeScore, play.AwayScore)
		favoriteScore := ternaryOperator(!awayIsFavored, play.HomeScore, play.AwayScore)
		csvRows = append(csvRows, []string{
			game.GameId,
			strconv.Itoa(int(play.SecondsElapsed)),
			strconv.Itoa(play.AwayScore),
			strconv.Itoa(play.HomeScore),
			strconv.Itoa(underdogScore),
			strconv.Itoa(favoriteScore),
			strconv.Itoa(favoriteScore - underdogScore),
		})
	}
	return csvRows
}

func upsertCsv(csvName string, rowsToInsert map[string][]string, getKey func([]string) string) error {
	var numUpdatedRows int
	var numNewRows int

	fullCsvPath := csvDirectory + "/" + csvName
	readFile, err1 := os.Open(fullCsvPath)
	if err1 != nil {
		return err1
	}
	defer readFile.Close()

	reader := csv.NewReader(readFile)
	rows, err2 := reader.ReadAll()
	if err2 != nil {
		return err2
	}

	var newCsv [][]string
	for _, row := range rows {
		if val, exists := rowsToInsert[getKey(row)]; !exists {
			newCsv = append(newCsv, row)
		} else {
			newCsv = append(newCsv, val)
			numUpdatedRows += 1
			delete(rowsToInsert, getKey(row))
		}
	}
	for _, row := range rowsToInsert {
		newCsv = append(newCsv, row)
		numNewRows += 1
	}

	writeFile, err3 := os.Create(fullCsvPath)
	if err3 != nil {
		return err3
	}

	writer := csv.NewWriter(writeFile)
	defer writer.Flush()
	for _, row := range newCsv {
		if err4 := writer.Write(row); err4 != nil {
			return err4
		}
	}
	Logger.Printf("For csv %s inserted %d new rows and updated %d rows", csvName, numNewRows, numUpdatedRows)
	return nil
}

func cleanedGamesQueryFilter(games []CleanedGame) bson.M {
	var gameIds = make([]string, 0, len(games))
	for _, game := range games {
		gameIds = append(gameIds, game.GameId)
	}
	return bson.M{
		"gameId": bson.M{
			"$in": gameIds,
		}}
}

func gameCsvKeyFunc(row []string) string {
	return row[0]
}

func playsCsvKeyFunc(row []string) string {
	return row[0] + row[1]
}
