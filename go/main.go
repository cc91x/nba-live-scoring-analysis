package main

import (
	"nba/helpers"
)

func main() {
	processName, date, logFile := helpers.Setup()
	defer logFile.Close()

	helpers.Logger.Printf("Running process: %s for date: %s", processName, date)

	processType, err := helpers.ValueOf(processName)
	switch processType {
	case helpers.CleanAllGames:
		err = helpers.CleanGames(date)
	case helpers.FetchRawOdds:
		err = helpers.FetchOdds(date)
	case helpers.CleanRawOdds:
		err = helpers.CleanOdds(date)
	case helpers.CombineGameWithOdds:
		err = helpers.CombineGamesAndOddsToCsv(date)
	default:
		helpers.Logger.Println("Incorrect process type parameter")
	}

	if err != nil {
		helpers.ErrorWithFailure(err)
	}
	helpers.Logger.Println("No errors detected. Exiting with success")
}
