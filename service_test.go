package expofier_test

import (
	"testing"

	"github.com/orsinium-labs/expofier"
)

func TestService(t *testing.T) {
	service := expofier.NewService()
	go service.Run(t.Context())
	msg := expofier.Message{
		To:   []expofier.Token{"ExpoPushToken[a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11]"},
		Body: "Hello world!",
	}
	promise := service.Send(t.Context(), msg)
	promise.Wait(t.Context())
	err := promise.Err()
	if err != expofier.ErrDeviceNotRegistered {
		t.Fatalf("error: %v", err)
	}
}
