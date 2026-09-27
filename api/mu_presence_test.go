package api

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/test"
)

type presenceTestDB struct {
	app         *model.Application
	membership  *model.ApplicationMembership
	memberships []*model.ApplicationMembership
	user        *model.User
}

func (d *presenceTestDB) GetApplicationByID(id uint) (*model.Application, error) {
	if d.app != nil && d.app.ID == id {
		return d.app, nil
	}
	return nil, nil
}

func (d *presenceTestDB) GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error) {
	if d.membership != nil && d.membership.ApplicationID == applicationID && d.membership.UserID == userID {
		return d.membership, nil
	}
	return nil, nil
}

func (d *presenceTestDB) GetApplicationMemberships(applicationID uint) ([]*model.ApplicationMembership, error) {
	return d.memberships, nil
}

func (d *presenceTestDB) GetUserByID(id uint) (*model.User, error) {
	if d.user != nil && d.user.ID == id {
		return d.user, nil
	}
	return nil, nil
}

type presenceNotification struct {
	userID uint
	event  *TypingEvent
}

type presenceTestNotifier struct {
	notifications []presenceNotification
}

func (n *presenceTestNotifier) NotifyMUEvent(userID uint, event any) {
	typing, ok := event.(*TypingEvent)
	if !ok {
		return
	}
	n.notifications = append(n.notifications, presenceNotification{userID: userID, event: typing})
}

func newPresenceContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/application/7/typing", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "7"}}
	test.WithUser(ctx, 1)
	return ctx, recorder
}

func TestMUPresenceSendsTypingToOtherChatMembers(t *testing.T) {
	db := &presenceTestDB{
		app: &model.Application{ID: 7, UserID: 9, ChannelType: model.ChannelTypeChat, AllowMemberPost: true},
		membership: &model.ApplicationMembership{
			ApplicationID: 7,
			UserID:        1,
			EffectiveRole: model.ChannelRoleMember,
		},
		memberships: []*model.ApplicationMembership{
			{ApplicationID: 7, UserID: 1},
			{ApplicationID: 7, UserID: 2},
		},
		user: &model.User{ID: 1, Name: "alice"},
	}
	notifier := &presenceTestNotifier{}
	handler := MUPresenceAPI{DB: db, Notifier: notifier}
	ctx, recorder := newPresenceContext(`{"typing":true}`)

	handler.SetTyping(ctx)

	if recorder.Code != 200 {
		t.Fatalf("expected HTTP 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(notifier.notifications) != 1 {
		t.Fatalf("expected one peer notification, got %d", len(notifier.notifications))
	}
	got := notifier.notifications[0]
	if got.userID != 2 || got.event.Type != "typing" || got.event.ApplicationID != 7 ||
		got.event.UserID != 1 || got.event.UserName != "alice" || !got.event.Typing {
		t.Fatalf("unexpected typing event: %#v", got)
	}
	remaining := time.Until(got.event.ExpiresAt)
	if remaining < 4*time.Second || remaining > 7*time.Second {
		t.Fatalf("unexpected typing expiry window: %s", remaining)
	}
}

func TestMUPresenceRejectsNotificationChannel(t *testing.T) {
	db := &presenceTestDB{
		app: &model.Application{ID: 7, UserID: 9, ChannelType: model.ChannelTypeNotification},
		membership: &model.ApplicationMembership{
			ApplicationID: 7,
			UserID:        1,
			EffectiveRole: model.ChannelRoleMember,
		},
		user: &model.User{ID: 1, Name: "alice"},
	}
	handler := MUPresenceAPI{DB: db, Notifier: &presenceTestNotifier{}}
	ctx, recorder := newPresenceContext(`{"typing":true}`)

	handler.SetTyping(ctx)

	if recorder.Code != 400 {
		t.Fatalf("expected HTTP 400, got %d", recorder.Code)
	}
}

func TestMUPresenceRejectsReadOnlyMember(t *testing.T) {
	db := &presenceTestDB{
		app: &model.Application{ID: 7, UserID: 9, ChannelType: model.ChannelTypeChat, AllowMemberPost: true},
		membership: &model.ApplicationMembership{
			ApplicationID: 7,
			UserID:        1,
			EffectiveRole: model.ChannelRoleReadOnly,
		},
		user: &model.User{ID: 1, Name: "alice"},
	}
	handler := MUPresenceAPI{DB: db, Notifier: &presenceTestNotifier{}}
	ctx, recorder := newPresenceContext(`{"typing":true}`)

	handler.SetTyping(ctx)

	if recorder.Code != 403 {
		t.Fatalf("expected HTTP 403, got %d", recorder.Code)
	}
}
