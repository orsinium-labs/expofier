package expofier

import "errors"

var (
	ErrDeviceNotRegistered = errors.New("device cannot receive push notifications anymore")
	ErrMessageTooBig       = errors.New("total notification payload was too large")
	ErrMessageRateExceeded = errors.New("you are sending messages too frequently to the given device")
	ErrMismatchSenderID    = errors.New("there is an issue with your FCM push credentials")
	ErrInvalidCredentials  = errors.New("push notification credentials for your standalone app are invalid")
	ErrNoRecipients        = errors.New("message has no recipients")
	ErrEmptyToken          = errors.New("push token is empty")
	ErrTooManyMessages     = errors.New("too many messages in a single request")
	ErrTooManyTickets      = errors.New("too many tickets in a single request")
	ErrServerError         = errors.New("server error")
	ErrTooManyRequests     = errors.New("too many requests")
	ErrInvalidTicketCount  = errors.New("the number of tickets doesn't match the number of messages")
	ErrBadRequest          = errors.New("invalid request")
	ErrUnknown             = errors.New("unknown error")
	ErrExpired             = errors.New("timed out delivering message")
)
