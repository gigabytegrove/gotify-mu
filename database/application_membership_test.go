package database

import (
	"testing"

	"github.com/gotify/server/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *DatabaseSuite) TestSharedApplicationMembership() {
	owner := &model.User{Name: "mu-owner", Pass: []byte{1}}
	member := &model.User{Name: "mu-member", Pass: []byte{1}}
	require.NoError(s.T(), s.db.CreateUser(owner))
	require.NoError(s.T(), s.db.CreateUser(member))

	app := &model.Application{UserID: owner.ID, Token: "MUAPP0000001", Name: "shared"}
	require.NoError(s.T(), s.db.CreateApplication(app))

	ownerMembership, err := s.db.GetApplicationMembership(app.ID, owner.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), ownerMembership)
	assert.True(s.T(), ownerMembership.ReceiveNotifications)

	accessible, err := s.db.GetAccessibleApplicationsByUser(member.ID)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), accessible)

	require.NoError(s.T(), s.db.UpsertApplicationMembership(&model.ApplicationMembership{
		ApplicationID: app.ID, UserID: member.ID, ReceiveNotifications: true,
	}))

	accessible, err = s.db.GetAccessibleApplicationsByUser(member.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), accessible, 1)
	assert.Equal(s.T(), app.ID, accessible[0].ID)

	recipients, err := s.db.GetApplicationRecipientUserIDs(app.ID)
	require.NoError(s.T(), err)
	assert.ElementsMatch(s.T(), []uint{owner.ID, member.ID}, recipients)
}

func (s *DatabaseSuite) TestAutoAssignedApplicationMembership() {
	owner := &model.User{Name: "mu-admin", Pass: []byte{1}, Admin: true}
	existing := &model.User{Name: "mu-existing", Pass: []byte{1}}
	require.NoError(s.T(), s.db.CreateUser(owner))
	require.NoError(s.T(), s.db.CreateUser(existing))

	app := &model.Application{UserID: owner.ID, Token: "MUAPP0000002", Name: "global"}
	require.NoError(s.T(), s.db.CreateApplication(app))
	require.NoError(s.T(), s.db.SetApplicationAutoAssign(app.ID, true))

	membership, err := s.db.GetApplicationMembership(app.ID, existing.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), membership)
	assert.True(s.T(), membership.AutoAssigned)

	createdLater := &model.User{Name: "mu-later", Pass: []byte{1}}
	require.NoError(s.T(), s.db.CreateUser(createdLater))
	membership, err = s.db.GetApplicationMembership(app.ID, createdLater.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), membership)
	assert.True(s.T(), membership.AutoAssigned)

	require.NoError(s.T(), s.db.SetApplicationAutoAssign(app.ID, false))
	membership, err = s.db.GetApplicationMembership(app.ID, existing.ID)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), membership)

	ownerMembership, err := s.db.GetApplicationMembership(app.ID, owner.ID)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), ownerMembership)
}

func (s *DatabaseSuite) TestSharedMessageDismissalIsPerUser() {
	owner := &model.User{Name: "mu-msg-owner", Pass: []byte{1}}
	member := &model.User{Name: "mu-msg-member", Pass: []byte{1}}
	require.NoError(s.T(), s.db.CreateUser(owner))
	require.NoError(s.T(), s.db.CreateUser(member))

	app := &model.Application{UserID: owner.ID, Token: "MUAPP0000003", Name: "shared-msg"}
	require.NoError(s.T(), s.db.CreateApplication(app))
	require.NoError(s.T(), s.db.UpsertApplicationMembership(&model.ApplicationMembership{
		ApplicationID: app.ID, UserID: member.ID, ReceiveNotifications: true,
	}))

	message := &model.Message{ApplicationID: app.ID, Message: "hello"}
	require.NoError(s.T(), s.db.CreateMessage(message))

	memberMessages, err := s.db.GetMessagesByUser(member.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), memberMessages, 1)

	require.NoError(s.T(), s.db.DismissMessageForUser(member.ID, message.ID))
	memberMessages, err = s.db.GetMessagesByUser(member.ID)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), memberMessages)

	ownerMessages, err := s.db.GetMessagesByUser(owner.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), ownerMessages, 1)
	assert.Equal(s.T(), message.ID, ownerMessages[0].ID)
}

func TestApplicationMembershipModelNames(t *testing.T) {
	assert.Equal(t, "application_memberships", (model.ApplicationMembership{}).TableName())
	assert.Equal(t, "message_dismissals", (model.MessageDismissal{}).TableName())
}
