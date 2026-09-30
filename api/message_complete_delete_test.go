package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/test/testdb"
	"github.com/stretchr/testify/require"
)

func TestDeleteMessageCompletelyRemovesAttachmentFiles(t *testing.T) {
	db := testdb.NewDB(t)
	defer db.Close()

	user := db.NewUser(1)
	app := &model.Application{
		UserID:      user.ID,
		Token:       "ADELETEIMAGE01",
		Name:        "Important Notices",
		ChannelType: model.ChannelTypeNotification,
	}
	require.NoError(t, db.CreateApplication(app))

	message := &model.Message{
		ApplicationID: app.ID,
		Title:         "Doorbell",
		Message:       "Someone is at the door.",
		Date:          time.Now(),
	}
	require.NoError(t, db.CreateMessage(message))

	dir := t.TempDir()
	storageName := "doorbell-image.bin"
	path := filepath.Join(dir, storageName)
	require.NoError(t, os.WriteFile(path, []byte("image-bytes"), 0o600))

	attachment := &model.MessageAttachment{
		MessageID:   message.ID,
		Filename:    "doorbell.jpg",
		ContentType: "image/jpeg",
		Size:        11,
		StorageName: storageName,
	}
	require.NoError(t, db.CreateMessageAttachment(attachment))

	handler := &MessageAPI{DB: db, AttachmentDir: dir}
	require.NoError(t, handler.deleteMessageCompletely(message.ID))

	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)

	storedAttachment, err := db.GetMessageAttachment(attachment.ID)
	require.NoError(t, err)
	require.Nil(t, storedAttachment)

	storedMessage, err := db.GetMessageByID(message.ID)
	require.NoError(t, err)
	require.Nil(t, storedMessage)
}
