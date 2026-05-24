package goncho

import (
	"context"

	webhookspkg "github.com/TrebuchetDynamics/goncho/internal/webhooks"
)

const (
	DefaultWebhookWorkspaceLimit = webhookspkg.DefaultWebhookWorkspaceLimit
	MaxWebhookURLLength          = webhookspkg.MaxWebhookURLLength
)

var (
	ErrWebhookWorkspaceRequired = webhookspkg.ErrWebhookWorkspaceRequired
	ErrWebhookInvalidURL        = webhookspkg.ErrWebhookInvalidURL
	ErrWebhookLimitReached      = webhookspkg.ErrWebhookLimitReached
	ErrWebhookNotFound          = webhookspkg.ErrWebhookNotFound
	ErrWebhookSecretMissing     = webhookspkg.ErrWebhookSecretMissing
)

type WebhookEndpointCreateParams = webhookspkg.WebhookEndpointCreateParams

type WebhookEndpointCreateResult = webhookspkg.WebhookEndpointCreateResult

type WebhookEndpoint = webhookspkg.WebhookEndpoint

type WebhookEventType = webhookspkg.WebhookEventType

const (
	WebhookEventQueueEmpty WebhookEventType = webhookspkg.WebhookEventQueueEmpty
	WebhookEventTest       WebhookEventType = webhookspkg.WebhookEventTest
)

type WebhookEvent = webhookspkg.WebhookEvent

type QueueEmptyWebhookEventParams = webhookspkg.QueueEmptyWebhookEventParams

func (s *Service) GetOrCreateWebhookEndpoint(ctx context.Context, params WebhookEndpointCreateParams) (WebhookEndpointCreateResult, error) {
	return webhookspkg.GetOrCreateEndpoint(ctx, s.db, s.workspaceID, params)
}

func (s *Service) ListWebhookEndpoints(ctx context.Context, workspaceID string) ([]WebhookEndpoint, error) {
	return webhookspkg.ListEndpoints(ctx, s.db, s.workspaceID, workspaceID)
}

func (s *Service) DeleteWebhookEndpoint(ctx context.Context, workspaceID, endpointID string) error {
	return webhookspkg.DeleteEndpoint(ctx, s.db, s.workspaceID, workspaceID, endpointID)
}

func NewTestWebhookEvent(workspaceID string) (WebhookEvent, error) {
	return webhookspkg.NewTestWebhookEvent(workspaceID)
}

func NewQueueEmptyWebhookEvent(params QueueEmptyWebhookEventParams) (WebhookEvent, error) {
	return webhookspkg.NewQueueEmptyWebhookEvent(params)
}

func SignWebhookPayload(payload, secret string) (string, error) {
	return webhookspkg.SignWebhookPayload(payload, secret)
}
