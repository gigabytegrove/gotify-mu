package database_test

import (
	"testing"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/test/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditEvents(t *testing.T) {
	db := testdb.NewDBWithDefaultUser(t)
	defer db.Close()

	require.NoError(t, db.CreateAuditEvent(&model.AuditEvent{
		UserID:   1,
		Username: "admin",
		Action:   "post",
		Target:   "/group",
	}))
	require.NoError(t, db.CreateAuditEvent(&model.AuditEvent{
		UserID:   1,
		Username: "admin",
		Action:   "delete",
		Target:   "/client/:id",
	}))

	all, err := db.GetAuditEvents(50, "", "")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	filtered, err := db.GetAuditEvents(50, "delete", "")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, "/client/:id", filtered[0].Target)
}
