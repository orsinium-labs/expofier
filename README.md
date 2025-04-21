# expofier

[ [📄 docs](https://pkg.go.dev/github.com/orsinium-labs/expofier) ] [ [🐙 github](https://github.com/orsinium-labs/expofier) ]

Go package for sending [push notifications](https://docs.expo.dev/push-notifications/overview/) to [Expo](https://expo.dev/) (React Native) apps.

It provides both a low-level API client as well as a backgorund service taking care of delivery guarantees.

## Installation

```bash
go get github.com/orsinium-labs/expofier
```

## Usage

See [the official Expo docs](https://docs.expo.dev/push-notifications/push-notifications-setup/) for instructions on configuring the client app and obtaining the push token.

Low-level usage:

```go
client := expofier.NewClient()

// Send message:
msg := expofier.Message{
    To:   []expofier.Token{"ExpoPushToken[a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11]"},
    Body: "Hello world!",
}
ctx := context.Background()
ticket, err := client.SendMessage(ctx, msg)
if err != nil {
    return fmt.Errorf("send message: %w", err)
}

// Check message delivery status:
receipt := client.FetchReceipt(ctx, msg)
if receipt != nil {
    if receipt.Error != nil {
        return fmt.Errorf("deliver message: %w", err)
    }
    fmt.Println("message delivered")
} else {
    fmt.Println("message not delivered yet")
}
```

For high-level usage, the package provides `Service` which takes care of grouping messages (to minimize network requests), chunking messages, retries, and checking the delivery status.

```go
service := expofier.NewService()
ctx := context.Background()
go service.Run(ctx)

msg := expofier.Message{
    To:   []expofier.Token{"ExpoPushToken[a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11]"},
    Body: "Hello world!",
}
promise := service.Send(ctx, msg)
promise.Wait(ctx)
err := promise.Err()
if err != nil {
    return err
}
```

It is recommended to check for `ErrDeviceNotRegistered` and remove invalid tokens from your database:

```go
if err == expofier.ErrDeviceNotRegistered {
    removeTokenFromDB(token)
}
```
