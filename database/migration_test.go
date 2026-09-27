package database

import (
	"testing"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigration(t *testing.T) {
	suite.Run(t, &MigrationSuite{})
}

type MigrationSuite struct {
	suite.Suite
	tmpDir test.TmpDir
}

func (s *MigrationSuite) BeforeTest(suiteName, testName string) {
	s.tmpDir = test.NewTmpDir("gotify_migrationsuite")
	db, err := gorm.Open(sqlite.Open(s.tmpDir.Path("test_obsolete.db")), &gorm.Config{})
	assert.NoError(s.T(), err)
	sqlDB, err := db.DB()
	assert.NoError(s.T(), err)
	defer sqlDB.Close()

	assert.Nil(s.T(), db.Migrator().CreateTable(new(model.User)))
	assert.Nil(s.T(), db.Create(&model.User{
		Name:  "test_user",
		Admin: true,
	}).Error)

	// we should not be able to create applications by now
	assert.False(s.T(), db.Migrator().HasTable(new(model.Application)))
}

func (s *MigrationSuite) AfterTest(suiteName, testName string) {
	assert.Nil(s.T(), s.tmpDir.Clean())
}

func (s *MigrationSuite) TestMigration() {
	db, err := New("sqlite3", s.tmpDir.Path("test_obsolete.db"), "admin", "admin", 6, true, fixedNow)
	assert.Nil(s.T(), err)
	defer db.Close()

	assert.True(s.T(), db.DB.Migrator().HasTable(new(model.Application)))

	// a user already exist, not adding a new user
	if user, err := db.GetUserByName("admin"); assert.NoError(s.T(), err) {
		assert.Nil(s.T(), user)
	}

	// the old user should persist
	if user, err := db.GetUserByName("test_user"); assert.NoError(s.T(), err) {
		assert.Equal(s.T(), true, user.Admin)
	}

	// we should be able to create applications
	if user, err := db.GetUserByName("test_user"); assert.NoError(s.T(), err) {
		assert.Nil(s.T(), db.CreateApplication(&model.Application{
			Token:       "A1234",
			UserID:      user.ID,
			Description: "this is a test application",
			Name:        "test application",
		}))
	}
	if app, err := db.GetApplicationByToken("A1234"); assert.NoError(s.T(), err) {
		assert.Equal(s.T(), "test application", app.Name)
	}
}


func (s *MigrationSuite) TestMigrationFromPreviewApplicationMembershipSchema() {
	path := s.tmpDir.Path("test_preview_membership.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	assert.NoError(s.T(), err)

	assert.NoError(s.T(), db.AutoMigrate(new(model.User), new(model.Application)))
	user := &model.User{Name: "preview_user", Admin: true}
	assert.NoError(s.T(), db.Create(user).Error)
	app := &model.Application{
		Token:       "PREVIEW123",
		UserID:      user.ID,
		Description: "preview channel",
		Name:        "preview channel",
	}
	assert.NoError(s.T(), db.Create(app).Error)

	assert.NoError(s.T(), db.Exec(`
		CREATE TABLE application_memberships (
			application_id integer NOT NULL,
			user_id integer NOT NULL,
			receive_notifications numeric NOT NULL,
			auto_assigned numeric NOT NULL,
			created_at datetime,
			updated_at datetime,
			PRIMARY KEY (application_id, user_id)
		)
	`).Error)
	assert.NoError(s.T(), db.Exec(
		"INSERT INTO application_memberships (application_id, user_id, receive_notifications, auto_assigned) VALUES (?, ?, ?, ?)",
		app.ID, user.ID, true, false,
	).Error)

	sqlDB, err := db.DB()
	assert.NoError(s.T(), err)
	assert.NoError(s.T(), sqlDB.Close())

	migrated, err := New("sqlite3", path, "admin", "admin", 6, false, fixedNow)
	assert.NoError(s.T(), err)
	if err != nil {
		return
	}
	defer migrated.Close()

	assert.True(s.T(), migrated.DB.Migrator().HasColumn(new(model.ApplicationMembership), "group_assigned"))
	assert.True(s.T(), migrated.DB.Migrator().HasColumn(new(model.ApplicationMembership), "group_receive_notifications"))

	var membership model.ApplicationMembership
	assert.NoError(s.T(), migrated.DB.Where(
		"application_id = ? AND user_id = ?", app.ID, user.ID,
	).First(&membership).Error)
	assert.False(s.T(), membership.GroupAssigned)
	assert.False(s.T(), membership.GroupReceiveNotifications)
	assert.Equal(s.T(), model.ChannelRoleMember, membership.Role)
}
