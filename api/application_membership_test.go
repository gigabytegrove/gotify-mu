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

func TestApplicationMembershipSetCurrentUserNotifications(t *testing.T) {
	db := testdb.NewDB(t)
	defer db.Close()

	owner := db.NewUser(1)
	member := db.NewUser(2)
	app := &model.Application{UserID: owner.ID, Token: "MUAPI0000001", Name: "shared"}
	require.NoError(t, db.CreateApplication(app))
	require.NoError(t, db.UpsertApplicationMembership(&model.ApplicationMembership{
		ApplicationID:        app.ID,
		UserID:               member.ID,
		ReceiveNotifications: true,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	test.WithUser(ctx, member.ID)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = httptest.NewRequest(
		"PUT",
		"/application/1/notifications",
		strings.NewReader(`{"enabled":false}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &ApplicationMembershipAPI{DB: db}
	handler.SetCurrentUserNotifications(ctx)

	assert.Equal(t, 200, recorder.Code)
	membership, err := db.GetApplicationMembership(app.ID, member.ID)
	require.NoError(t, err)
	require.NotNil(t, membership)
	assert.False(t, membership.ReceiveNotifications)
}

func TestApplicationMembershipTransferOwnership(t *testing.T) {
	db := testdb.NewDB(t)
	defer db.Close()

	owner := db.NewUser(1)
	nextOwner := db.NewUser(2)
	app := &model.Application{UserID: owner.ID, Token: "MUAPI0000002", Name: "shared"}
	require.NoError(t, db.CreateApplication(app))
	require.NoError(t, db.UpsertApplicationMembership(&model.ApplicationMembership{
		ApplicationID:        app.ID,
		UserID:               nextOwner.ID,
		ReceiveNotifications: true,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	test.WithUser(ctx, owner.ID)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = httptest.NewRequest(
		"PUT",
		"/application/1/owner",
		strings.NewReader(`{"userId":2}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := &ApplicationMembershipAPI{DB: db}
	handler.TransferOwnership(ctx)

	assert.Equal(t, 200, recorder.Code)
	updated, err := db.GetApplicationByID(app.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, nextOwner.ID, updated.UserID)
}
