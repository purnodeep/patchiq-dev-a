package events

import (
	"database/sql"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubscriberFactory creates a new Watermill subscriber with a unique consumer group.
// Each call returns a fresh subscriber so that multiple handlers on the same topic
// each receive every message (fan-out) instead of competing (fan-in).
type SubscriberFactory func(consumerGroup string) (message.Subscriber, error)

// NewPublisherAndSubscriberFactory creates a Watermill SQL publisher and a factory
// that creates subscribers with unique consumer groups. This replaces the old
// NewPublisherAndSubscriber which shared a single subscriber across all handlers.
func NewPublisherAndSubscriberFactory(db *sql.DB, logger watermill.LoggerAdapter) (*watermillSQL.Publisher, SubscriberFactory, error) {
	pub, err := watermillSQL.NewPublisher(db, watermillSQL.PublisherConfig{
		SchemaAdapter:        watermillSQL.DefaultPostgreSQLSchema{},
		AutoInitializeSchema: true,
	}, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("create watermill publisher: %w", err)
	}

	factory := func(consumerGroup string) (message.Subscriber, error) {
		sub, err := watermillSQL.NewSubscriber(db, watermillSQL.SubscriberConfig{
			ConsumerGroup:    consumerGroup,
			SchemaAdapter:    watermillSQL.DefaultPostgreSQLSchema{},
			OffsetsAdapter:   watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
			InitializeSchema: true,
		}, logger)
		if err != nil {
			return nil, fmt.Errorf("create watermill subscriber (group=%s): %w", consumerGroup, err)
		}
		return sub, nil
	}

	return pub, factory, nil
}

// NewRouter creates a Watermill message router.
func NewRouter(logger watermill.LoggerAdapter) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, fmt.Errorf("create watermill router: %w", err)
	}
	return router, nil
}
