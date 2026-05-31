package webhooks

import (
	"errors"
	"time"

	deliverycontract "github.com/TrebuchetDynamics/goncho/internal/webhooks/delivery"
	eventcontract "github.com/TrebuchetDynamics/goncho/internal/webhooks/events"
	"github.com/TrebuchetDynamics/goncho/internal/webhooks/urlpolicy"
)

const (
	defaultWebhookDeliveryMaxAttempts = deliverycontract.DefaultMaxAttempts
	defaultWebhookDeliveryBackoff     = deliverycontract.DefaultBackoff
)

type WebhookDeliveryStatus = deliverycontract.Status

const (
	WebhookDeliveryDelivered        WebhookDeliveryStatus = deliverycontract.Delivered
	WebhookDeliveryRetryable        WebhookDeliveryStatus = deliverycontract.Retryable
	WebhookDeliveryFailed           WebhookDeliveryStatus = deliverycontract.Failed
	WebhookDeliveryEndpointDisabled WebhookDeliveryStatus = deliverycontract.EndpointDisabled
	WebhookDeliverySkipped          WebhookDeliveryStatus = deliverycontract.Skipped
)

type WebhookDeliveryErrorClass = deliverycontract.ErrorClass

const (
	WebhookDeliveryErrorNone       WebhookDeliveryErrorClass = deliverycontract.ErrorNone
	WebhookDeliveryErrorHTTPStatus WebhookDeliveryErrorClass = deliverycontract.ErrorHTTPStatus
	WebhookDeliveryErrorNetwork    WebhookDeliveryErrorClass = deliverycontract.ErrorNetwork
	WebhookDeliveryErrorSigning    WebhookDeliveryErrorClass = deliverycontract.ErrorSigning
	WebhookDeliveryErrorStore      WebhookDeliveryErrorClass = deliverycontract.ErrorStore
	WebhookDeliveryErrorDisabled   WebhookDeliveryErrorClass = deliverycontract.ErrorDisabled
)

type WebhookDeliveryEndpoint = deliverycontract.Endpoint

type WebhookDeliveryStore = deliverycontract.Store

type WebhookHTTPClient = deliverycontract.HTTPClient

type WebhookClock = deliverycontract.Clock

type WebhookDeliveryWorker = deliverycontract.Worker

type WebhookDeliveryRequest = deliverycontract.Request

type WebhookHTTPRequest = deliverycontract.HTTPRequest

type WebhookHTTPResponse = deliverycontract.HTTPResponse

type WebhookDeliveryAttempt = deliverycontract.Attempt

type WebhookDeliveryResult = deliverycontract.Result

type WebhookDeliveryEvidence = deliverycontract.Evidence

func retryableWebhookStatus(statusCode int) bool {
	return deliverycontract.RetryableStatus(statusCode)
}

func failureDisableReason(result WebhookDeliveryResult) string {
	return deliverycontract.FailureDisableReason(result)
}

func buildWebhookDeliveryPayload(event WebhookEvent, now time.Time) (string, error) {
	payload, err := deliverycontract.Payload(event, now)
	if errors.Is(err, eventcontract.ErrWorkspaceRequired) {
		return "", ErrWebhookWorkspaceRequired
	}
	return payload, err
}

func redactWebhookEndpointURL(raw string) string {
	return urlpolicy.RedactEndpoint(raw)
}
