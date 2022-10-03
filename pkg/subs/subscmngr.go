package subs

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Svc struct {
	mongo *mongo.Client
}

func New(mongo *mongo.Client) *Svc {
	return &Svc{
		mongo: mongo,
	}
}

type Subscription struct {
	// Channel is the communication medium to send notifications to the subscriber.
	// It can be one of the following:
	// - sms
	Channel string `bson:"channel"`

	// Endpoint is the address to send notifications to.
	// For SMS, it is the phone number.
	Endpoint string `bson:"endpoint"`

	// Status is the status of the subscription.
	// It can be one of the following:
	// - active
	// - inactive
	Status string `bson:"status"`

	// // LastNotifiedAt is the time when the subscription is last notified.
	// LastNotifiedAt int64 `bson:"last_notified_at"`

	// // LastNotifiedStatus is the status of the last notification.
	// // It can be one of the following:
	// // - success
	// // - failure
	// LastNotifiedStatus string `bson:"last_notified_status"`

	// // LastNotifiedError is the error of the last notification.
	// LastNotifiedError string `bson:"last_notified_error"`
}

func (s *Svc) Subscribe(channel string, endpoint string) error {
	coll := s.mongo.Database("connchk").Collection("subscriptions")
	// insert or update a subscription
	sub := Subscription{
		Channel:   channel,
		Endpoint:  endpoint,
		Status:    "active",
	}

	u, err := coll.UpdateOne(
		context.Background(),
		bson.M{
			"channel":  channel,
			"endpoint": endpoint,
		},
		bson.M{
			"$set": sub,
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	log.Printf("updated %d documents", u.ModifiedCount)

	return nil
}

func (s *Svc) Unsubscribe(channel string, endpoint string) error {
	coll := s.mongo.Database("connchk").Collection("subscriptions")
	_, err := coll.UpdateOne(
		context.Background(),
		bson.M{
			"channel":  channel,
			"endpoint": endpoint,
		},
		bson.M{
			"$set": bson.M{
				"status": "inactive",
			},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *Svc) ListActiveSubs(channel string) ([]Subscription, error) {
	coll := s.mongo.Database("connchk").Collection("subscriptions")
	// list all active subscriptions
	cur, err := coll.Find(
		context.Background(),
		bson.M{
			"channel": channel,
			"status":  "active",
		},
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	var subs []Subscription
	for cur.Next(context.Background()) {
		var sub Subscription
		err := cur.Decode(&sub)
		if err != nil {
			return nil, err
		}

		subs = append(subs, sub)
	}

	return subs, nil
}