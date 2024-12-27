package helpers

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/* MongoDb connection related */
func loadMongoDbClient(config NbaConfig) (client *mongo.Client, err error) {
	clientUrl := "mongodb://" + config.Database.Host + ":" + config.Database.Port

	clientOptions := options.Client().ApplyURI(clientUrl)
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, errors.New("error connecting to mongoDb client")
	}
	return client, nil
}

func closeMongoDBConnection(client *mongo.Client, err error) error {
	if err != nil {
		return err
	}
	if err = client.Disconnect(context.TODO()); err != nil {
		return err
	}
	return nil
}

/* DB operations related */
func upsertItemsGeneric(operations []mongo.WriteModel, dbCollection *mongo.Collection) (writeResult *mongo.BulkWriteResult, err error) {
	if len(operations) == 0 {
		Logger.Println("Found 0 rows to upsert")
	} else {
		result, err := dbCollection.BulkWrite(context.TODO(), operations)
		if err != nil {
			return nil, err
		}
		Logger.Printf("Upserted %v documents", result.UpsertedCount)
	}
	return writeResult, nil
}

func findTeamMetadata(teamInfoCollection *mongo.Collection) (teamMetadata []TeamMetadata, err error) {
	cursor, err1 := teamInfoCollection.Find(context.TODO(), bson.M{})
	err2 := cursor.All(context.TODO(), &teamMetadata)

	if err1 != nil || err2 != nil {
		return nil, errors.New("error doing db lookup for team metadata")
	}
	return teamMetadata, nil
}

func findCleanedGame(date string, dbCollection *mongo.Collection) (games []CleanedGame, err error) {
	cursor, err1 := dbCollection.Find(context.TODO(), dateFieldStringFilter(date))
	err2 := cursor.All(context.TODO(), &games)
	if err1 != nil || err2 != nil {
		return nil, handleMultipleErrors(err1, err2)
	}
	Logger.Printf("Found %d processed games in DB", len(games))
	return games, nil
}

/* Error management related funcs */
func ErrorWithFailure(err error) {
	Logger.Fatalf("Error: %v", err)
}

func handleMultipleErrors(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

/* Mongo query filters */
func rawOddsDbFilter(date string, utcHour int) bson.M {
	return bson.M{
		"date":    date,
		"utcHour": utcHour,
	}
}

func dateFieldStringFilter(date string) bson.M {
	return bson.M{"date": date}
}

/* Generic util */
func ternaryOperator[T any](condition bool, val1 T, val2 T) T {
	if condition {
		return val1
	} else {
		return val2
	}
}
