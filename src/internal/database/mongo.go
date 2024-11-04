package database

import (
    "context" // Add this import
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB implementation
type MongoDatabase struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewMongoDatabase() *MongoDatabase {
	return &MongoDatabase{}
}

func (m *MongoDatabase) Connect(ctx context.Context, connectionString string) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
	if err != nil {
		return err
	}
	m.client = client
	m.db = client.Database("mydb") // Set your database name
	return nil
}

func (m *MongoDatabase) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

func (m *MongoDatabase) Create(ctx context.Context, collection string, document interface{}) error {
	_, err := m.db.Collection(collection).InsertOne(ctx, document)
	return err
}


func (m *MongoDatabase) FindOne(ctx context.Context, collection string, filter interface{}, result interface{}) error {
	return m.db.Collection(collection).FindOne(ctx, filter).Decode(result)
}

func (m *MongoDatabase) Find(ctx context.Context, collection string, filter interface{}, result interface{}, limit int64, offset int64) error {
    cursor, err := m.db.Collection(collection).Find(ctx, filter, options.Find().SetLimit(limit).SetSkip(offset))
    if err != nil {
        return err
    }
    defer cursor.Close(ctx)

    return cursor.All(ctx, result) // Decode all results into the provided result slice
}

func (m *MongoDatabase) Update(ctx context.Context, collection string, filter interface{}, update interface{}) error {
	_, err := m.db.Collection(collection).UpdateOne(ctx, filter, update)
	return err
}

func (m *MongoDatabase) Delete(ctx context.Context, collection string, filter interface{}) error {
	_, err := m.db.Collection(collection).DeleteOne(ctx, filter)
	return err
}