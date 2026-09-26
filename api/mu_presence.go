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
		req := typingRequest{}
		if err := ctx.BindJSON(&req); err != nil {
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

		isChat := app.ChannelType == model.ChannelTypeChat ||
			(app.ChannelType == "" && app.AllowMemberPost)
		if !isChat {
			ctx.AbortWithError(400, errors.New("typing presence is only available for chat channels"))
			return
		}
		if app.UserID != userID {
			role := membership.EffectiveRole
			canPost := role == model.ChannelRoleManager ||
				role == model.ChannelRolePublisher ||
				(role == model.ChannelRoleMember && app.AllowMemberPost)
			if !canPost {
				ctx.AbortWithError(403, errors.New("your Channel role does not allow posting"))
				return
			}
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
		if req.Typing {
			expiry = expiry.Add(6 * time.Second)
		}

		event := &TypingEvent{
			Type:          "typing",
			ApplicationID: id,
			UserID:        userID,
			UserName:      user.Name,
			Typing:        req.Typing,
			ExpiresAt:     expiry,
		}

		for _, member := range memberships {
			if member.UserID == userID {
				continue
			}
			a.Notifier.NotifyMUEvent(member.UserID, event)
		}

		ctx.JSON(200, event)
	})
}
