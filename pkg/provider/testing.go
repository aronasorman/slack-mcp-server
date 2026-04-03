package provider

import (
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// NewForTest creates an ApiProvider suitable for unit tests. It accepts a mock
// SlackAPI implementation and initialises the users/channels caches to empty
// snapshots so that handler code that loads from them does not panic.
func NewForTest(client SlackAPI, logger *zap.Logger) *ApiProvider {
	ap := &ApiProvider{
		transport:  "test",
		client:     client,
		logger:     logger,
		usersReady: true,
	}
	ap.usersSnapshot.Store(&UsersCache{
		Users:    map[string]slack.User{},
		UsersInv: map[string]string{},
	})
	ap.channelsSnapshot.Store(&ChannelsCache{
		Channels:    map[string]Channel{},
		ChannelsInv: map[string]string{},
	})
	return ap
}
