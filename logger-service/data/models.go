package data

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func New(mongo *mongo.Client) Models {
	client = mongo

	return Models{
		LogEntry: LogEntry{},
	}
}

type Models struct {
	LogEntry LogEntry
}

type LogEntry struct {
	ID        string    `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string    `bson:"name" json:"name"`
	Data      string    `bson:"data" json:"data"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

func (l *LogEntry) collection() *mongo.Collection {
	return client.Database("logs").Collection("logs")
}

func (l *LogEntry) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

func (l *LogEntry) Insert() error {
	collection := l.collection()
	ctx, _ := l.context()
	_, err := collection.InsertOne(ctx, LogEntry{
		Name:      l.Name,
		Data:      l.Data,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		log.Printf("Error inserting log entry: %v\n", err)
		return err
	}
	return nil
}

func (l *LogEntry) All() ([]*LogEntry, error) {
	collection := l.collection()
	ctx, cancel := l.context()
	defer cancel()

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		log.Printf("Error getting log entries: %v\n", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []*LogEntry
	for cursor.Next(ctx) {
		var item LogEntry
		if err := cursor.Decode(&item); err != nil {
			log.Printf("Error decoding log entry: %v\n", err)
			return nil, err
		}
		logs = append(logs, &item)
	}
	return logs, nil
}

func (l *LogEntry) GetOne(id string) (*LogEntry, error) {
	collection := l.collection()
	ctx, cancel := l.context()
	defer cancel()

	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("Error converting string to mongo ID: %v\n", err)
		return nil, err
	}

	var entry LogEntry
	err = collection.FindOne(ctx, bson.M{"_id": docID}).Decode(&entry)
	if err != nil {
		log.Printf("Error getting log entry: %v\n", err)
		return nil, err
	}
	return &entry, nil
}

func (l *LogEntry) DropCollection() error {
	collection := l.collection()
	ctx, cancel := l.context()
	defer cancel()

	err := collection.Drop(ctx)
	if err != nil {
		log.Printf("Error dropping collection: %v\n", err)
		return err
	}
	return nil
}

func (l *LogEntry) Update() (*mongo.UpdateResult, error) {
	ctx, cancel := l.context()
	collection := l.collection()
	defer cancel()

	docID, err := primitive.ObjectIDFromHex(l.ID)
	if err != nil {
		log.Printf("Error converting string to mongo ID: %v\n", err)
		return nil, err
	}
	result, err := collection.UpdateOne(ctx, bson.M{"_id": docID},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "name", Value: l.Name},
				{Key: "data", Value: l.Data},
				{Key: "updated_at", Value: time.Now().UTC()},
			}},
		})
	if err != nil {
		log.Printf("Error updating log entry: %v\n", err)
		return nil, err
	}
	return result, nil
}
