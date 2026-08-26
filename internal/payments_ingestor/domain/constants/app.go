package constants

type ResponseType string

const (
	TypeNotFound        ResponseType = "NOT_FOUND"
	TypeValidationError ResponseType = "VALIDATION"
	TypeUnauthorized    ResponseType = "UNAUTHORIZED"
	TypeInternal        ResponseType = "INTERNAL"
	TypeConflict        ResponseType = "CONFLICT"
)

const (
	CodeMalformedJSON         = "MALFORMED_JSON"
	CodeEmptyRequestBody      = "EMPTY_REQUEST_BODY"
	CodeMissingIdempotencyKey = "MISSING_IDEMPOTENCY_KEY"
	CodeInvalidIdempotencyKey = "INVALID_IDEMPOTENCY_KEY"
	CodeInvalidAttributes     = "INVALID_ATTRIBUTES"

	CodeConcurrentRequest = "CONCURRENT_REQUEST"
	CodeMissingUserId     = "MISSING_USER_ID"
	CodeInvalidCurrency   = "INVALID_CURRENCY"

	CodeInternalServerError = "INTERNAL_SERVER_ERROR"

	CodeSystemHealthy   = "SYSTEM_HEALTHY"
	CodePaymentAccepted = "PAYMENT_ACCEPTED"
	CodePaymentSettled  = "PAYMENT_SETTLED"
	CodeAccountCreated  = "ACCOUNT_CREATED"

	CodeDataBaseError       = "DATABASE_ERROR"
	CodeIdempotencyConflict = "IDEMPOTENCY_CONFLICT"
)

func GetMessage(code string) string {
	switch code {

	case CodeSystemHealthy:
		return "System is operational and ready to process requests."
	case CodePaymentAccepted:
		return "Payment request successfully validated and placed in queue."
	case CodePaymentSettled:
		return "Payment processed and settled successfully."
	case CodeAccountCreated:
		return "Account created successfully."

	case CodeMalformedJSON:
		return "The request payload could not be parsed due to malformed JSON syntax."
	case CodeEmptyRequestBody:
		return "The request body is empty."
	case CodeMissingIdempotencyKey:
		return "Header 'Idempotency-Key' is required for this operation."
	case CodeInvalidIdempotencyKey:
		return "Header 'Idempotency-Key' format is invalid."
	case CodeConcurrentRequest:
		return "A request with this Idempotency-Key is currently being processed."
	case CodeMissingUserId:
		return "Missing user_id."
	case CodeInvalidCurrency:
		return "Currency must be a 3-character ISO-4217 code"
	case CodeInvalidAttributes:
		return "The request has invalid attributes."

	case CodeInternalServerError:
		return "An unexpected internal error occurred."
	case CodeIdempotencyConflict:
		return "A request with this idempotency key is currently being processed."
	default:
		return "An unexpected error occurred."
	}
}
