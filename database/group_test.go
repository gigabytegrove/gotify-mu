package database_test

import (
	"testing"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/test/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserGroupsCRUDAndMembership(t *testing.T) {
	db := testdb.NewDBWithDefaultUser(t)
	defer db.Close()

	db.NewUserWithName(2, "operator")

	group := &model.UserGroup{Name: "Operators", Description: "On-call operators"}
	require.NoError(t, db.CreateUserGroup(group))
	require.NotZero(t, group.ID)

	require.NoError(t, db.AddUserGroupMember(group.ID, 2))
	require.NoError(t, db.AddUserGroupMember(group.ID, 2))

	count, err := db.CountUserGroupMembers(group.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	members, err := db.GetUserGroupMembers(group.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "operator", members[0].Name)

	require.NoError(t, db.RemoveUserGroupMember(group.ID, 2))
	count, err = db.CountUserGroupMembers(group.ID)
	require.NoError(t, err)
	assert.Zero(t, count)

	require.NoError(t, db.DeleteUserGroup(group.ID))
	found, err := db.GetUserGroupByID(group.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}
