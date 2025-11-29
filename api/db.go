package main

import (
	"context"
	"time"

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
