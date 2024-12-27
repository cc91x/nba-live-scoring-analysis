package helpers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func FetchOdds(date string) (err error) {
	client, err := loadMongoDbClient(*Config)
	if err != nil {
		return
	}
	defer func() {
		if err4 := closeMongoDBConnection(client, err); err4 != nil {
			err = err4
		}
	}()
	rawOddsCollection := getHistoricalOddscollection(client, Config.Database.Schema)

	var existingData RawOddsResponse
	var oddsResponses []RawOddsResponse
	var rawOdds *RawOddsResponse
	for _, val := range utcHoursForLookup {
		err = rawOddsCollection.FindOne(context.TODO(), rawOddsDbFilter(date, val)).Decode(&existingData)
		if err == mongo.ErrNoDocuments {
			rawOdds, err = fetchOdds(date, val)
			oddsResponses = append(oddsResponses, *rawOdds)
		}
		if err != nil {
			return err
		}
	}
	Logger.Printf("Fetched %d new odds responses from source", len(oddsResponses))
	_, err = upsertRawOddsRows(oddsResponses, rawOddsCollection)
	return err
}

func fetchOdds(date string, utcHour int) (oddsResponse *RawOddsResponse, err error) {
	urlString := buildOddsSourceUrl(date, strconv.Itoa(utcHour))
	response, err1 := http.Get(urlString)
	responseData, err2 := io.ReadAll(response.Body)
	err3 := json.Unmarshal(responseData, &oddsResponse)

	if err1 != nil || err2 != nil || err3 != nil {
		return nil, handleMultipleErrors(err1, err2, err3)
	}
	oddsResponse.Date = date
	oddsResponse.UtcHour = utcHour
	return oddsResponse, nil
}

func upsertRawOddsRows(oddsResponse []RawOddsResponse, dbCollection *mongo.Collection) (*mongo.BulkWriteResult, error) {
	var operations = make([]mongo.WriteModel, 0, len(oddsResponse))
	for _, doc := range oddsResponse {
		operations = append(operations, mongo.NewUpdateOneModel().
			SetFilter(rawOddsDbFilter(doc.Date, doc.UtcHour)).
			SetUpdate(bson.M{"$set": doc}).
			SetUpsert(true))
	}
	return upsertItemsGeneric(operations, dbCollection)
}

// TODO: Hide this from git
func buildOddsSourceUrl(date string, utcHour string) string {
	return Config.OddsApi.BaseUrl + oddsSourceApiPath + "?apiKey=" + Config.OddsApi.Key + "&markets=spreads,totals,h2h&regions=us&date=" + date + "T" + utcHour + ":00:00Z"
}
