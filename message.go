package expofier

type Token string
type Data map[string]string

type Priority string

const (
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

type Message struct {
	// Expo push tokens specifying the recipient(s) of this message.
	To []Token `json:"to"`

	// The title to display in the notification.
	//
	// On iOS, this is displayed only on Apple Watch.
	Title string `json:"title,omitempty"`

	// The message to display in the notification.
	Body string `json:"body"`

	// A dict of extra data to pass inside of the push notification. The total notification payload must be at most 4096 bytes.
	Data Data `json:"data,omitempty"`

	// A sound to play when the recipient receives this notification.
	//
	// Specify "default" to play the device's default notification sound,
	// or omit this field to play no sound.
	Sound string `json:"sound,omitempty"`

	// The number of seconds for which the message may be kept around for redelivery
	// if it hasn't been delivered yet.
	TTL int `json:"ttl,omitempty"`

	// Delivery priority of the message.
	Priority Priority `json:"priority,omitempty"`

	// An integer representing the unread notification count.
	//
	// This currently only affects iOS. Specify 0 to clear the badge count.
	Badge int `json:"badge,omitempty"`

	// ID of the Notification Channel through which to display this notification on Android devices.
	ChannelID string `json:"channelId,omitempty"`
}
