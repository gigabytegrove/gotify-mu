package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/test"
	"github.com/gotify/server/v3/test/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureMUEvents struct {
	events map[uint][]any
}

func (c *captureMUEvents) NotifyMUEvent(userID uint, event any) {
	if c.events == nil {
		c.events = make(map[uint][]any)
	}
	c.events[userID] = append(c.events[userID], event)
}

func TestMUPresenceTypingBroadcastsToOtherChatMembers(t *testing.T) {
	db := testdb.NewDB(t)
	defer db.Close()

	owner := db.NewUser(1)
	member := db.NewUser(2)
	app := &model.Application{
		UserID:          owner.ID,
		Token:           "MUTYPING00001",
		Name:            "chat",
		ChannelType:     model.ChannelTypeChat,
		AllowMemberPost: true,
	}
	require.NoError(t, db.CreateApplication(app))
	require.NoError(t, db.UpsertApplicationMembership(&model.ApplicationMembership{
		ApplicationID:        app.ID,
		UserID:               member.ID,
		ReceiveNotifications: true,
	}))

	notifier := &captureMUEvents{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	test.WithUser(ctx, member.ID)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = httptest.NewRequest(
		"POST",
		"/application/1/typing",
		strings.NewReader(`{"typing":true}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &MUPresenceAPI{DB: db, Notifier: notifier}
	handler.SetTyping(ctx)

	assert.Equal(t, 200, recorder.Code)
	require.Len(t, notifier.events[owner.ID], 1)
	assert.Empty(t, notifier.events[member.ID])
	event, ok := notifier.events[owner.ID][0].(*TypingEvent)
	require.True(t, ok)
	assert.Equal(t, app.ID, event.ApplicationID)
	assert.Equal(t, member.ID, event.UserID)
	assert.Equal(t, member.Name, event.UserName)
	assert.True(t, event.Typing)
}

func TestMUPresenceRejectsNotificationChannel(t *testing.T) {
	db := testdb.NewDB(t)
	defer db.Close()

	owner := db.NewUser(1)
	app := &model.Application{
		UserID:      owner.ID,
		Token:       "MUTYPING00002",
		Name:        "alerts",
		ChannelType: model.ChannelTypeNotification,
	}
	require.NoError(t, db.CreateApplication(app))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	test.WithUser(ctx, owner.ID)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = httptest.NewRequest(
		"POST",
		"/application/1/typing",
		strings.NewReader(`{"typing":true}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &MUPresenceAPI{DB: db, Notifier: &captureMUEvents{}}
	handler.SetTyping(ctx)

	assert.Equal(t, 400, recorder.Code)
}
