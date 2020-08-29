package commands

// MessageResponse contains information to respond to a command
// Idiomatic usage is to send for each message/reaction, and not to cache
type MessageResponse struct {
	// Reaction contains a reaction to add or remove (or both)
	// It is intentionally singular
	Reaction ReactionResponse
	// Message is the message to send
	// It is intentionally singular
	Message   string
	ChannelID string
}

// ReactionResponse contains information to add or remove reactions
type ReactionResponse struct {
	Add       string
	Remove    string
	MessageID string
}
