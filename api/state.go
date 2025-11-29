package main

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Collections:
// - token: {_id: "token", access_token, refresh_token, expires_at}
// - subscription: {_id: "subscription", id: number}
// - activities: one document per activity, _id = activity.Id

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
	res := coll.FindOneAndUpdate(
		context.Background(),
		bson.D{{Key: "_id", Value: "token"}},
		bson.D{{Key: "$set", Value: token}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	)
	return res.Err()
}

/* func loadSubscriptionId() (int, error) {
	if mongoClient == nil {
		return 0, nil
	}
	coll := mongoClient.Database(MONGO_DB).Collection("subscription")
	var doc struct {
		Id int `bson:"id"`
	}
	err := coll.FindOne(context.Background(), bson.D{{Key: "_id", Value: "subscription"}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		}
		return 0, err
	}
	return doc.Id, nil
}

func saveSubscriptionID(id int) error {
	if mongoClient == nil {
		return nil
	}
	coll := mongoClient.Database(MONGO_DB).Collection("subscription")
	res := coll.FindOneAndUpdate(
		context.Background(),
		bson.D{{Key: "_id", Value: "subscription"}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "id", Value: id}}}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	)
	return res.Err()
} */

func addActivity(activity *Activity) error {
	if mongoClient == nil {
		return nil
	}
	coll := mongoClient.Database(MONGO_DB).Collection("activities")
	res := coll.FindOneAndUpdate(
		context.Background(),
		bson.D{{Key: "_id", Value: activity.Id}},
		bson.D{{Key: "$set", Value: activity}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	)
	return res.Err()
}

func setActivities(activities []Activity) error {
	if mongoClient == nil {
		return nil
	}
	coll := mongoClient.Database(MONGO_DB).Collection("activities")
	// Upsert each activity; simple loop for now
	for i := range activities {
		res := coll.FindOneAndUpdate(
			context.Background(),
			bson.D{{Key: "_id", Value: activities[i].Id}},
			bson.D{{Key: "$set", Value: activities[i]}},
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		)
		if err := res.Err(); err != nil {
			fmt.Println("Error saving activity:", err)
			return err
		}
	}
	return nil
}

func listActivities() ([]Activity, error) {
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
