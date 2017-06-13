package shared

import (
	"strconv"
	"strings"

	"code.google.com/p/go-uuid/uuid"
)

// ChannelHandshake handshake channel path
const ChannelHandshake = "/meta/handshake"

// ChannelConnect connect channel path
const ChannelConnect = "/meta/connect"

// ChannelDisconnect disconnect channel path
const ChannelDisconnect = "/meta/disconnect"

// ChannelSubscribe subscribe channel path
const ChannelSubscribe = "/meta/subscribe"

// ChannelUnsubscribe unsubscribe channel path
const ChannelUnsubscribe = "/meta/unsubscribe"

// Core shared functions
type Core struct {
	idCount int64
}

// NextMessageID returns the next message id incremented from the current running count on the instance of Core
func (c *Core) NextMessageID() string {
	c.idCount++
	return strconv.FormatInt(c.idCount, 10)
}

// GenerateClientID generates a new UUID string
func (c *Core) GenerateClientID() string {
	rawID := uuid.New()
	return strings.Replace(rawID, "-", "", -1)
}
