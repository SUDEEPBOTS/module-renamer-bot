package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDatabase implements Database interface over MongoDB.
type MongoDatabase struct {
	client       *mongo.Client
	usersColl    *mongo.Collection
	bannedColl   *mongo.Collection
	statsColl    *mongo.Collection
}

// NewMongoDatabase connects to a MongoDB server.
func NewMongoDatabase(uri, dbName string) (*MongoDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	return &MongoDatabase{
		client:     client,
		usersColl:  db.Collection("users"),
		bannedColl: db.Collection("banned"),
		statsColl:  db.Collection("stats"),
	}, nil
}

func (m *MongoDatabase) AddUser(id int64, username, firstName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user := User{
		ID:        id,
		Username:  username,
		FirstName: firstName,
		JoinedAt:  time.Now().Unix(),
	}

	opts := options.Update().SetUpsert(true)
	_, err := m.usersColl.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": user}, opts)
	return err
}

func (m *MongoDatabase) GetUsers() ([]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := m.usersColl.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []int64
	for cursor.Next(ctx) {
		var u User
		if err := cursor.Decode(&u); err == nil {
			users = append(users, u.ID)
		}
	}
	return users, nil
}

func (m *MongoDatabase) CountUsers() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.usersColl.CountDocuments(ctx, bson.M{})
}

func (m *MongoDatabase) IsBanned(id int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	count, err := m.bannedColl.CountDocuments(ctx, bson.M{"_id": id})
	return err == nil && count > 0
}

func (m *MongoDatabase) BanUser(id int64, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	banned := BannedUser{
		ID:       id,
		Reason:   reason,
		BannedAt: time.Now().Unix(),
	}
	opts := options.Update().SetUpsert(true)
	_, err := m.bannedColl.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": banned}, opts)
	return err
}

func (m *MongoDatabase) UnbanUser(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.bannedColl.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (m *MongoDatabase) IncrementRenames() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := options.Update().SetUpsert(true)
	_, err := m.statsColl.UpdateOne(ctx, bson.M{"_id": "global"}, bson.M{"$inc": bson.M{"total_renames": 1}}, opts)
	return err
}

func (m *MongoDatabase) GetTotalRenames() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var res struct {
		TotalRenames int64 `bson:"total_renames"`
	}
	err := m.statsColl.FindOne(ctx, bson.M{"_id": "global"}).Decode(&res)
	if err != nil {
		return 0
	}
	return res.TotalRenames
}

func (m *MongoDatabase) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}
