package api

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
)

type MUPresenceDatabase interface {
	GetApplicationByID(id uint) (*model.Application, error)
	GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error)
	GetApplicationMemberships(applicationID uint) ([]*model.ApplicationMembership, error)
	GetUserByID(id uint) (*model.User, error)
}

type MUEventNotifier interface {
	NotifyMUEvent(userID uint, event any)
}

type MUPresenceAPI struct {
	DB       MUPresenceDatabase
	Notifier MUEventNotifier
}

type typingRequest struct {
	Typing bool `json:"typing"`
}

type TypingEvent struct {
	Type          string    `json:"type"`
	ApplicationID uint      `json:"applicationId"`
	UserID        uint      `json:"userId"`
	UserName      string    `json:"userName"`
	Typing        bool      `json:"typing"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

func (a *MUPresenceAPI) SetTyping(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		params := typingRequest{}
		if err := ctx.BindJSON(&params); err != nil {
			return
		}

		userID := auth.GetUserID(ctx)
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if app == nil {
			ctx.AbortWithError(404, errors.New("application does not exist"))
			return
		}

		membership, err := a.DB.GetApplicationMembership(id, userID)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if membership == nil {
			ctx.AbortWithError(404, errors.New("application does not exist"))
			return
		}

		if !app.AllowMemberPost {
			ctx.AbortWithError(400, errors.New("typing presence is only available for chat channels"))
			return
		}

		user, err := a.DB.GetUserByID(userID)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if user == nil {
			ctx.AbortWithError(404, errors.New("user does not exist"))
			return
		}

		memberships, err := a.DB.GetApplicationMemberships(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}

		expiry := time.Now().UTC()
		if params.Typing {
			expiry = expiry.Add(6 * time.Second)
		}
		event := &TypingEvent{
			Type:          "typing",
			ApplicationID: id,
			UserID:        userID,
			UserName:      user.Name,
			Typing:        params.Typing,
			ExpiresAt:     expiry,
		}
		for _, target := range memberships {
			if target.UserID == userID {
				continue
			}
			a.Notifier.NotifyMUEvent(target.UserID, event)
		}
		ctx.JSON(200, event)
	})
}
