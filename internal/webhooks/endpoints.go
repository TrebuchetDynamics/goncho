package webhooks

import (
	"context"
	"database/sql"
	"errors"

	endpointcontract "github.com/TrebuchetDynamics/goncho/internal/webhooks/endpoints"
	eventcontract "github.com/TrebuchetDynamics/goncho/internal/webhooks/events"
	"github.com/TrebuchetDynamics/goncho/internal/webhooks/signing"
)

const (
	DefaultWebhookWorkspaceLimit = endpointcontract.DefaultWorkspaceLimit
	MaxWebhookURLLength          = endpointcontract.MaxURLLength
)

var (
	ErrWebhookWorkspaceRequired = endpointcontract.ErrWorkspaceRequired
	ErrWebhookInvalidURL        = endpointcontract.ErrInvalidURL
	ErrWebhookLimitReached      = endpointcontract.ErrLimitReached
	ErrWebhookNotFound          = endpointcontract.ErrNotFound
	ErrWebhookSecretMissing     = signing.ErrSecretMissing
)

type WebhookEndpointCreateParams = endpointcontract.CreateParams

type WebhookEndpointCreateResult = endpointcontract.CreateResult

type WebhookEndpoint = endpointcontract.Endpoint

type WebhookEventType = eventcontract.Type

const (
	WebhookEventQueueEmpty = eventcontract.QueueEmpty
	WebhookEventTest       = eventcontract.Test
)

type WebhookEvent = eventcontract.Event

type QueueEmptyWebhookEventParams = eventcontract.QueueEmptyParams

func GetOrCreateEndpoint(ctx context.Context, db *sql.DB, defaultWorkspaceID string, params WebhookEndpointCreateParams) (WebhookEndpointCreateResult, error) {
	return endpointcontract.GetOrCreate(ctx, db, defaultWorkspaceID, params)
}

func ListEndpoints(ctx context.Context, db *sql.DB, defaultWorkspaceID, workspaceID string) ([]WebhookEndpoint, error) {
	return endpointcontract.List(ctx, db, defaultWorkspaceID, workspaceID)
}

func DeleteEndpoint(ctx context.Context, db *sql.DB, defaultWorkspaceID, workspaceID, endpointID string) error {
	return endpointcontract.Delete(ctx, db, defaultWorkspaceID, workspaceID, endpointID)
}

func NewTestWebhookEvent(workspaceID string) (WebhookEvent, error) {
	event, err := eventcontract.NewTest(workspaceID)
	if errors.Is(err, eventcontract.ErrWorkspaceRequired) {
		return WebhookEvent{}, ErrWebhookWorkspaceRequired
	}
	return event, err
}

func NewQueueEmptyWebhookEvent(params QueueEmptyWebhookEventParams) (WebhookEvent, error) {
	event, err := eventcontract.NewQueueEmpty(params)
	if errors.Is(err, eventcontract.ErrWorkspaceRequired) {
		return WebhookEvent{}, ErrWebhookWorkspaceRequired
	}
	return event, err
}

func SignWebhookPayload(payload, secret string) (string, error) {
	signature, err := signing.SignPayload(payload, secret)
	if errors.Is(err, signing.ErrSecretMissing) {
		return "", ErrWebhookSecretMissing
	}
	return signature, err
}
