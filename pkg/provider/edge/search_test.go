package edge

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchResponseMessageParsing(t *testing.T) {
	raw := `{
		"ok": true,
		"module": "messages",
		"query": "hello",
		"filters": {},
		"pagination": {
			"total_count": 42,
			"page": 1,
			"per_page": 20,
			"page_count": 3,
			"first": 1,
			"last": 20
		},
		"items": [
			{
				"type": "message",
				"user": "U12345",
				"username": "alice",
				"text": "hello world",
				"ts": "1710000000.000100",
				"permalink": "https://myteam.slack.com/archives/C99999/p1710000000000100",
				"channel": {"id": "C99999", "name": "general"}
			},
			{
				"type": "message",
				"user": "U67890",
				"username": "bob",
				"text": "hello again",
				"ts": "1710000001.000200",
				"permalink": "https://myteam.slack.com/archives/C88888/p1710000001000200",
				"channel": {"id": "C88888", "name": "random"}
			}
		]
	}`

	var sr SearchResponse[MessageItem]
	err := json.Unmarshal([]byte(raw), &sr)
	require.NoError(t, err)

	assert.True(t, sr.Ok)
	assert.Equal(t, "messages", sr.Module)
	assert.Equal(t, "hello", sr.Query)

	// Pagination
	assert.Equal(t, int64(42), sr.Pagination.TotalCount)
	assert.Equal(t, 1, sr.Pagination.Page)
	assert.Equal(t, 20, sr.Pagination.PerPage)
	assert.Equal(t, 3, sr.Pagination.PageCount)

	// Items
	require.Len(t, sr.Items, 2)

	msg0 := sr.Items[0]
	assert.Equal(t, "message", msg0.Type)
	assert.Equal(t, "U12345", msg0.User)
	assert.Equal(t, "alice", msg0.Username)
	assert.Equal(t, "hello world", msg0.Text)
	assert.Equal(t, "1710000000.000100", msg0.Timestamp)
	assert.Equal(t, "C99999", msg0.Channel.ID)
	assert.Equal(t, "general", msg0.Channel.Name)

	msg1 := sr.Items[1]
	assert.Equal(t, "U67890", msg1.User)
	assert.Equal(t, "bob", msg1.Username)
	assert.Equal(t, "C88888", msg1.Channel.ID)
	assert.Equal(t, "random", msg1.Channel.Name)
}

func TestSearchResponseMessagePagination(t *testing.T) {
	t.Run("has next page", func(t *testing.T) {
		raw := `{
			"ok": true,
			"module": "messages",
			"query": "test",
			"filters": {},
			"pagination": {
				"total_count": 50,
				"page": 1,
				"per_page": 20,
				"page_count": 3,
				"first": 1,
				"last": 20
			},
			"items": []
		}`
		var sr SearchResponse[MessageItem]
		require.NoError(t, json.Unmarshal([]byte(raw), &sr))
		assert.Equal(t, 3, sr.Pagination.PageCount)
		assert.Equal(t, 1, sr.Pagination.Page)
		assert.True(t, sr.Pagination.Page < sr.Pagination.PageCount, "should have more pages")
	})

	t.Run("last page", func(t *testing.T) {
		raw := `{
			"ok": true,
			"module": "messages",
			"query": "test",
			"filters": {},
			"pagination": {
				"total_count": 50,
				"page": 3,
				"per_page": 20,
				"page_count": 3,
				"first": 41,
				"last": 50
			},
			"items": []
		}`
		var sr SearchResponse[MessageItem]
		require.NoError(t, json.Unmarshal([]byte(raw), &sr))
		assert.False(t, sr.Pagination.Page < sr.Pagination.PageCount, "should be last page")
	})
}

func TestSearchResponseMessageError(t *testing.T) {
	raw := `{
		"ok": false,
		"error": "invalid_auth",
		"response_metadata": {}
	}`
	var sr SearchResponse[MessageItem]
	require.NoError(t, json.Unmarshal([]byte(raw), &sr))
	err := sr.validate("search.modules.messages")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_auth")
}

func TestMessageItemDefaults(t *testing.T) {
	// Verify the default count and page values used by SearchMessages.
	assert.Equal(t, 20, defaultMessageSearchCount)
}
