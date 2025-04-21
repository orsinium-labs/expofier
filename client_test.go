package expofier_test

import (
	"testing"

	"github.com/orsinium-labs/expofier"
)

func TestClient_SendMessage(t *testing.T) {
	client := expofier.NewClient()
	msg := expofier.Message{
		To:   []expofier.Token{"ExpoPushToken[a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11]"},
		Body: "Hello world!",
	}
	_, err := client.SendMessage(t.Context(), msg)
	if err != expofier.ErrDeviceNotRegistered {
		t.Fatalf("error: %v", err)
	}
}

func TestClient_SendMessages(t *testing.T) {
	client := expofier.NewClient()
	resps, err := client.SendMessages(t.Context(), []expofier.Message{{
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

func TestClient_FetchReceipt(t *testing.T) {
	client := expofier.NewClient()
	receipt := client.FetchReceipt(t.Context(), "hello-world")
	if receipt != nil {
		t.Fatalf("error: %v", receipt.Error)
	}
}

func TestClient_FetchReceipts(t *testing.T) {
	client := expofier.NewClient()
	ticket := expofier.Ticket("hello-world")
	receipts, err := client.FetchReceipts(t.Context(), []expofier.Ticket{ticket})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	receipt, found := receipts[ticket]
	if found {
		t.Fatalf("error: %v", receipt.Error)
	}
}
