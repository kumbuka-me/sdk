package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestUserAndNotificationCapabilities verifies typed directory and notification routing.
func TestUserAndNotificationCapabilities(t *testing.T) {
	createdAt := time.Date(2026, time.September, 25, 10, 30, 0, 0, time.UTC)
	client := NewClient(func(method string, params, result any) error {
		switch method {
		case "users.search":
			query := params.(UserQuery)
			require.Equal(t, "ali", query.Query)
			require.Equal(t, 10, query.Limit)
			*result.(*[]User) = []User{{ID: 42, Mention: "@alice", DisplayName: "Alice"}}
		case "users.resolve-mention":
			require.Equal(t, "@Alice", params.(UserMention).Mention)
			*result.(*User) = User{ID: 42, Mention: "@alice", DisplayName: "Alice"}
		case "notifications.send":
			input := params.(NotificationInput)
			require.Equal(t, int64(42), input.RecipientUserID)
			require.Equal(t, "task:one:assigned", input.IdempotencyKey)
			*result.(*Notification) = Notification{ID: 7, RecipientUserID: 42, CreatedAt: createdAt}
		default:
			require.FailNowf(t, "unexpected call", "unexpected method %q", method)
		}
		return nil
	})

	users, err := client.Users().Search(UserQuery{Query: "ali", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, "@alice", users[0].Mention)
	user, err := client.Users().ResolveMention("@Alice")
	require.NoError(t, err)
	require.Equal(t, int64(42), user.ID)
	notification, err := client.Notifications().Send(NotificationInput{RecipientUserID: 42, Title: "Assigned", IdempotencyKey: "task:one:assigned"})
	require.NoError(t, err)
	require.Equal(t, createdAt, notification.CreatedAt)

	for method, permission := range map[string]string{
		"users.search":          "users:read",
		"users.resolve-mention": "users:read",
		"notifications.send":    "notifications:send",
	} {
		actual, ok := PermissionFor(method)
		require.True(t, ok)
		require.Equal(t, permission, actual)
		require.True(t, ValidPermission(permission))
	}
}
