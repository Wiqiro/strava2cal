package main

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var mongoClient *mongo.Client

func initMongo() error {
	client, err := mongo.Connect(options.Client().ApplyURI(MONGO_URI))
	if err != nil {
		return err
	}
	mongoClient = client
	return nil
}

func disconnectMongo() {
	if mongoClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = mongoClient.Disconnect(ctx)
}

func getToken() (*StravaToken, error) {
	if mongoClient == nil {
		return nil, nil
	}
	coll := mongoClient.Database(MONGO_DB).Collection("token")
	var token StravaToken
	err := coll.FindOne(context.Background(), bson.D{{Key: "_id", Value: "token"}}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func saveToken(token *StravaToken) error {
	if mongoClient == nil {
		return nil
	}

	coll := mongoClient.Database(MONGO_DB).Collection("token")
	_, err := coll.UpdateOne(
		context.Background(),
		bson.D{{Key: "_id", Value: "token"}},
		bson.D{{Key: "$set", Value: token}},
		options.UpdateOne().SetUpsert(true),
	)
	return err

}

func upsertActivity(activity *Activity) error {
	if mongoClient == nil {
		return nil
	}
	coll := mongoClient.Database(MONGO_DB).Collection("activities")
	_, err := coll.UpdateOne(
		context.Background(),
		bson.D{{Key: "_id", Value: activity.Id}},
		bson.D{{Key: "$set", Value: activity}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func setActivities(activities []Activity) error {
	if mongoClient == nil {
		return nil
	}
	coll := mongoClient.Database(MONGO_DB).Collection("activities")
	_, err := coll.DeleteMany(context.Background(), bson.D{})
	if err != nil {
		return err
	}

	_, err = coll.InsertMany(context.Background(), activities)
	if err != nil {
		return err
	}

	return nil
}

func getActivities() ([]Activity, error) {
	if mongoClient == nil {
		return nil, nil
	}
	coll := mongoClient.Database(MONGO_DB).Collection("activities")
	cur, err := coll.Find(context.Background(), bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())
	var out []Activity
	for cur.Next(context.Background()) {
		var a Activity
		if err := cur.Decode(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}
