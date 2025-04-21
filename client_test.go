package expofier_test

import (
	"testing"

	"github.com/orsinium-labs/expofier"
)

func TestClient_SendMessage(t *testing.T) {
	c := expofier.NewClient()
	_, err := c.SendMessage(t.Context(), expofier.Message{
		To:   []expofier.Token{"ExpoPushToken[a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11]"},
		Body: "Hello world!",
	})
	if err != expofier.ErrDeviceNotRegistered {
		t.Fatalf("error: %v", err)
	}
}

func TestClient_SendMessages(t *testing.T) {
	c := expofier.NewClient()
	resps, err := c.SendMessages(t.Context(), []expofier.Message{{
		To:   []expofier.Token{"ExpoPushToken[a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11]"},
		Body: "Hello world!",
	}})
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if len(resps) != 1 {
		t.Fatalf("resp len: %d", len(resps))
	}
	if resps[0].Error != expofier.ErrDeviceNotRegistered {
		t.Fatalf("message error: %v", resps[0].Error)
	}
}
