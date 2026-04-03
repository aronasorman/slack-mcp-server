package handler

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/korotovsky/slack-mcp-server/pkg/provider"
	"github.com/korotovsky/slack-mcp-server/pkg/provider/edge"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockSlackAPI implements provider.SlackAPI for testing searchMessagesEdge.
// Only SearchMessages is functional; other methods panic if called.
type mockSlackAPI struct {
	searchMessagesFunc func(ctx context.Context, query string, count, page int) ([]edge.MessageItem, edge.Pagination, error)
}

func (m *mockSlackAPI) SearchMessages(ctx context.Context, query string, count, page int) ([]edge.MessageItem, edge.Pagination, error) {
	return m.searchMessagesFunc(ctx, query, count, page)
}

// Stubs — not exercised by searchMessagesEdge.
func (m *mockSlackAPI) AuthTest() (*slack.AuthTestResponse, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) AuthTestContext(context.Context) (*slack.AuthTestResponse, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetUsersContext(context.Context, ...slack.GetUsersOption) ([]slack.User, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetUsersInfo(...string) (*[]slack.User, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) PostMessageContext(context.Context, string, ...slack.MsgOption) (string, string, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) MarkConversationContext(context.Context, string, string) error {
	panic("not implemented")
}
func (m *mockSlackAPI) AddReactionContext(context.Context, string, slack.ItemRef) error {
	panic("not implemented")
}
func (m *mockSlackAPI) RemoveReactionContext(context.Context, string, slack.ItemRef) error {
	panic("not implemented")
}
func (m *mockSlackAPI) GetConversationHistoryContext(context.Context, *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetConversationRepliesContext(context.Context, *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) SearchContext(context.Context, string, slack.SearchParameters) (*slack.SearchMessages, *slack.SearchFiles, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetFileInfoContext(context.Context, string, int, int) (*slack.File, []slack.Comment, *slack.Paging, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetFileContext(context.Context, string, io.Writer) error {
	panic("not implemented")
}
func (m *mockSlackAPI) GetConversationInfoContext(context.Context, *slack.GetConversationInfoInput) (*slack.Channel, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetConversationsContext(context.Context, *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetConversationsForUserContext(context.Context, *slack.GetConversationsForUserParameters) ([]slack.Channel, string, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) ClientUserBoot(context.Context) (*edge.ClientUserBootResponse, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) UsersSearch(context.Context, string, int) ([]slack.User, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) ClientCounts(context.Context) (edge.ClientCountsResponse, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetMutedChannels(context.Context) (map[string]bool, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetUserGroupsContext(context.Context, ...slack.GetUserGroupsOption) ([]slack.UserGroup, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) GetUserGroupMembersContext(context.Context, string, ...slack.GetUserGroupMembersOption) ([]string, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) CreateUserGroupContext(context.Context, slack.UserGroup, ...slack.CreateUserGroupOption) (slack.UserGroup, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) UpdateUserGroupContext(context.Context, string, ...slack.UpdateUserGroupsOption) (slack.UserGroup, error) {
	panic("not implemented")
}
func (m *mockSlackAPI) UpdateUserGroupMembersContext(context.Context, string, string, ...slack.UpdateUserGroupMembersOption) (slack.UserGroup, error) {
	panic("not implemented")
}

func newTestHandler(mock *mockSlackAPI) *ConversationsHandler {
	logger := zap.NewNop()
	ap := provider.NewForTest(mock, logger)
	return NewConversationsHandler(ap, logger)
}

func TestSearchMessagesEdge(t *testing.T) {
	t.Run("maps MessageItem fields to CSV output", func(t *testing.T) {
		mock := &mockSlackAPI{
			searchMessagesFunc: func(_ context.Context, query string, count, page int) ([]edge.MessageItem, edge.Pagination, error) {
				assert.Equal(t, "hello", query)
				assert.Equal(t, 10, count)
				assert.Equal(t, 1, page)
				return []edge.MessageItem{
					{
						Type:      "message",
						User:      "U123",
						Username:  "alice",
						Text:      "hello world",
						Timestamp: "1710000000.000100",
						Permalink: "https://team.slack.com/archives/C999/p1710000000000100",
						Channel:   edge.MessageChannel{ID: "C999", Name: "general"},
					},
				}, edge.Pagination{TotalCount: 1, Page: 1, PageCount: 1, PerPage: 10}, nil
			},
		}

		h := newTestHandler(mock)
		result, err := h.searchMessagesEdge(context.Background(), &searchParams{query: "hello", limit: 10, page: 1})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Content, 1)

		text := result.Content[0].(mcp.TextContent).Text
		records := parseCSV(t, text)
		require.Len(t, records, 2) // header + 1 row

		header := records[0]
		row := records[1]
		fieldIdx := csvIndex(header)

		assert.Equal(t, "1710000000.000100", row[fieldIdx["MsgID"]])
		assert.Equal(t, "U123", row[fieldIdx["UserID"]])
		assert.Equal(t, "hello world", row[fieldIdx["Text"]])
		assert.Equal(t, "#general", row[fieldIdx["Channel"]])
		assert.Equal(t, "", row[fieldIdx["Cursor"]], "last page should have no cursor")
	})

	t.Run("sets cursor when more pages exist", func(t *testing.T) {
		mock := &mockSlackAPI{
			searchMessagesFunc: func(_ context.Context, _ string, _, _ int) ([]edge.MessageItem, edge.Pagination, error) {
				return []edge.MessageItem{
					{
						Type:      "message",
						User:      "U1",
						Username:  "bob",
						Text:      "page one",
						Timestamp: "1710000000.000100",
						Permalink: "https://team.slack.com/archives/C1/p1710000000000100",
						Channel:   edge.MessageChannel{ID: "C1", Name: "test"},
					},
				}, edge.Pagination{TotalCount: 40, Page: 1, PageCount: 2, PerPage: 20}, nil
			},
		}

		h := newTestHandler(mock)
		result, err := h.searchMessagesEdge(context.Background(), &searchParams{query: "q", limit: 20, page: 1})
		require.NoError(t, err)

		text := result.Content[0].(mcp.TextContent).Text
		records := parseCSV(t, text)
		require.Len(t, records, 2)

		cursor := records[1][csvIndex(records[0])["Cursor"]]
		require.NotEmpty(t, cursor, "should have a next-page cursor")

		decoded, err := base64.StdEncoding.DecodeString(cursor)
		require.NoError(t, err)
		assert.Equal(t, "page:2", string(decoded))
	})

	t.Run("empty results", func(t *testing.T) {
		mock := &mockSlackAPI{
			searchMessagesFunc: func(_ context.Context, _ string, _, _ int) ([]edge.MessageItem, edge.Pagination, error) {
				return []edge.MessageItem{}, edge.Pagination{TotalCount: 0, Page: 1, PageCount: 0, PerPage: 20}, nil
			},
		}

		h := newTestHandler(mock)
		result, err := h.searchMessagesEdge(context.Background(), &searchParams{query: "nothing", limit: 20, page: 1})
		require.NoError(t, err)
		require.NotNil(t, result)

		// CSV with only the header row (no data rows).
		text := result.Content[0].(mcp.TextContent).Text
		records := parseCSV(t, text)
		assert.Len(t, records, 1, "should contain only the CSV header")
	})

	t.Run("propagates error from SearchMessages", func(t *testing.T) {
		mock := &mockSlackAPI{
			searchMessagesFunc: func(_ context.Context, _ string, _, _ int) ([]edge.MessageItem, edge.Pagination, error) {
				return nil, edge.Pagination{}, fmt.Errorf("network timeout")
			},
		}

		h := newTestHandler(mock)
		_, err := h.searchMessagesEdge(context.Background(), &searchParams{query: "q", limit: 20, page: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "network timeout")
	})

	t.Run("cursor round-trip with parseParamsToolSearch", func(t *testing.T) {
		// Encode a cursor the same way searchMessagesEdge does.
		cursor := base64.StdEncoding.EncodeToString([]byte("page:3"))

		// Decode it the same way parseParamsToolSearch does.
		decoded, err := base64.StdEncoding.DecodeString(cursor)
		require.NoError(t, err)

		parts := strings.Split(string(decoded), ":")
		require.Len(t, parts, 2)
		assert.Equal(t, "page", parts[0])
		assert.Equal(t, "3", parts[1])
	})
}

// parseCSV parses a CSV string into records.
func parseCSV(t *testing.T, s string) [][]string {
	t.Helper()
	r := csv.NewReader(strings.NewReader(s))
	records, err := r.ReadAll()
	require.NoError(t, err)
	return records
}

// csvIndex returns a map from column name to index for a CSV header row.
func csvIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[h] = i
	}
	return m
}
