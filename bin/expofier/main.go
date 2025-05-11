package main

import (
	"context"
	"flag"
	"log"

	"github.com/orsinium-labs/expofier"
)

func main() {
	var token string
	msg := expofier.Message{}

	flag.StringVar(&token, "token", "", "The push token of the target device.")
	flag.StringVar(&msg.Title, "title", "", "The title to display in the notification.")
	flag.StringVar(&msg.Body, "body", "", "The message to display in the notification.")
	flag.StringVar(&msg.Sound, "sound", "", "A sound to play when the recipient receives this notification.")
	flag.StringVar(&msg.ChannelID, "channel-id", "", "ID of the Notification Channel through which to display this notification on Android devices.")

	flag.Parse()
	if token == "" {
		log.Fatal("token is required")
	}
	msg.To = []expofier.Token{expofier.Token(token)}

	service := expofier.NewService()
	ctx := context.Background()
	go service.Run(ctx)

	promise := service.Send(ctx, msg)
	promise.Wait(ctx)
	err := promise.Err()
	if err != nil {
		log.Fatal(err)
	}
	println("delivered!")
}
