package goncho

import webhookspkg "github.com/TrebuchetDynamics/goncho/internal/webhooks"

type WebhookDeliveryStatus = webhookspkg.WebhookDeliveryStatus

const (
	WebhookDeliveryDelivered        WebhookDeliveryStatus = webhookspkg.WebhookDeliveryDelivered
	WebhookDeliveryRetryable        WebhookDeliveryStatus = webhookspkg.WebhookDeliveryRetryable
	WebhookDeliveryFailed           WebhookDeliveryStatus = webhookspkg.WebhookDeliveryFailed
	WebhookDeliveryEndpointDisabled WebhookDeliveryStatus = webhookspkg.WebhookDeliveryEndpointDisabled
	WebhookDeliverySkipped          WebhookDeliveryStatus = webhookspkg.WebhookDeliverySkipped
)

type WebhookDeliveryErrorClass = webhookspkg.WebhookDeliveryErrorClass

const (
	WebhookDeliveryErrorNone       WebhookDeliveryErrorClass = webhookspkg.WebhookDeliveryErrorNone
	WebhookDeliveryErrorHTTPStatus WebhookDeliveryErrorClass = webhookspkg.WebhookDeliveryErrorHTTPStatus
	WebhookDeliveryErrorNetwork    WebhookDeliveryErrorClass = webhookspkg.WebhookDeliveryErrorNetwork
	WebhookDeliveryErrorSigning    WebhookDeliveryErrorClass = webhookspkg.WebhookDeliveryErrorSigning
	WebhookDeliveryErrorStore      WebhookDeliveryErrorClass = webhookspkg.WebhookDeliveryErrorStore
	WebhookDeliveryErrorDisabled   WebhookDeliveryErrorClass = webhookspkg.WebhookDeliveryErrorDisabled
)

type WebhookDeliveryEndpoint = webhookspkg.WebhookDeliveryEndpoint

type WebhookDeliveryStore = webhookspkg.WebhookDeliveryStore

type WebhookHTTPClient = webhookspkg.WebhookHTTPClient

type WebhookClock = webhookspkg.WebhookClock

type WebhookDeliveryWorker = webhookspkg.WebhookDeliveryWorker

type WebhookDeliveryRequest = webhookspkg.WebhookDeliveryRequest

type WebhookHTTPRequest = webhookspkg.WebhookHTTPRequest

type WebhookHTTPResponse = webhookspkg.WebhookHTTPResponse

type WebhookDeliveryAttempt = webhookspkg.WebhookDeliveryAttempt

type WebhookDeliveryResult = webhookspkg.WebhookDeliveryResult

type WebhookDeliveryEvidence = webhookspkg.WebhookDeliveryEvidence
