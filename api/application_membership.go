package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
)

type ApplicationMembershipDatabase interface {
	GetApplicationByID(id uint) (*model.Application, error)
	GetUserByID(id uint) (*model.User, error)
	GetUsers() ([]*model.User, error)
	GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error)
	GetApplicationMemberships(applicationID uint) ([]*model.ApplicationMembership, error)
	UpsertApplicationMembership(membership *model.ApplicationMembership) error
	DeleteApplicationMembership(applicationID, userID uint) error
	SetApplicationAutoAssign(applicationID uint, enabled bool) error
	SetApplicationMembershipNotifications(applicationID, userID uint, enabled bool) error
	TransferApplicationOwnership(applicationID, newOwnerID uint) error
	SetApplicationMemberPosting(applicationID uint, enabled bool) error
	GetApplicationGroupGrants(applicationID uint) ([]*model.ApplicationGroupGrant, error)
	UpsertApplicationGroupGrant(grant *model.ApplicationGroupGrant) error
	DeleteApplicationGroupGrant(applicationID, groupID uint) error
	GetUserGroupByID(id uint) (*model.UserGroup, error)
}

type ApplicationMembershipAPI struct {
	DB ApplicationMembershipDatabase
}

type ApplicationMemberParams struct {
	UserID               uint   `json:"userId" binding:"required"`
	ReceiveNotifications *bool  `json:"receiveNotifications,omitempty"`
	Role                 string `json:"role"`
}

type ApplicationMemberExternal struct {
	UserID               uint   `json:"userId"`
	Name                 string `json:"name"`
	Owner                bool   `json:"owner"`
	ReceiveNotifications bool   `json:"receiveNotifications"`
	AutoAssigned         bool   `json:"autoAssigned"`
	Role                 string `json:"role"`
}

type ApplicationAutoAssignParams struct {
	Enabled bool `json:"enabled"`
}

type ApplicationNotificationParams struct {
	Enabled bool `json:"enabled"`
}


type ApplicationOwnerParams struct {
	UserID uint `json:"userId" binding:"required"`
}

type ApplicationMemberPostingParams struct {
	Enabled bool `json:"enabled"`
}

type ApplicationGroupGrantParams struct {
	GroupID              uint   `json:"groupId" binding:"required"`
	Role                 string `json:"role" binding:"required"`
	ReceiveNotifications *bool  `json:"receiveNotifications,omitempty"`
}

type ApplicationGroupGrantExternal struct {
	GroupID              uint   `json:"groupId"`
	Name                 string `json:"name"`
	Role                 string `json:"role"`
	ReceiveNotifications bool   `json:"receiveNotifications"`
}

func (a *ApplicationMembershipAPI) authorizeOwnerOrAdmin(
	userID uint,
	app *model.Application,
) (bool, error) {
	if app == nil {
		return false, nil
	}
	if app.UserID == userID {
		return true, nil
	}

	user, err := a.DB.GetUserByID(userID)
	if err != nil {
		return false, err
	}
	if user != nil && user.Admin {
		return true, nil
	}
	membership, err := a.DB.GetApplicationMembership(app.ID, userID)
	if err != nil {
		return false, err
	}
	return membership != nil && membership.Role == model.ApplicationRoleManager, nil
}

func (a *ApplicationMembershipAPI) getAuthorizedApplication(
	ctx *gin.Context,
	id uint,
) (*model.Application, bool) {
	app, err := a.DB.GetApplicationByID(id)
	if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
		return nil, false
	}

	allowed, err := a.authorizeOwnerOrAdmin(auth.GetUserID(ctx), app)
	if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
		return nil, false
	}
	if app == nil || !allowed {
		ctx.AbortWithError(http.StatusNotFound, errors.New("application does not exist"))
		return nil, false
	}
	return app, true
}

func (a *ApplicationMembershipAPI) GetMembers(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, ok := a.getAuthorizedApplication(ctx, id)
		if !ok {
			return
		}

		memberships, err := a.DB.GetApplicationMemberships(id)
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}

		result := make([]ApplicationMemberExternal, 0, len(memberships))
		for _, membership := range memberships {
			user, err := a.DB.GetUserByID(membership.UserID)
			if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
				return
			}
			if user == nil {
				continue
			}

			result = append(result, ApplicationMemberExternal{
				UserID:               user.ID,
				Name:                 user.Name,
				Owner:                user.ID == app.UserID,
				ReceiveNotifications: membership.ReceiveNotifications,
				AutoAssigned:         membership.AutoAssigned,
				Role: func() string {
					if user.ID == app.UserID { return "owner" }
					if membership.Role == "" { return model.ApplicationRoleMember }
					return membership.Role
				}(),
			})
		}
		ctx.JSON(http.StatusOK, result)
	})
}

func (a *ApplicationMembershipAPI) UpsertMember(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, ok := a.getAuthorizedApplication(ctx, id)
		if !ok {
			return
		}

		params := ApplicationMemberParams{}
		if err := ctx.Bind(&params); err != nil {
			return
		}
		if app.Internal {
			ctx.AbortWithError(
				http.StatusBadRequest,
				errors.New("internal applications cannot be shared"),
			)
			return
		}
		if params.UserID == app.UserID {
			ctx.AbortWithError(
				http.StatusBadRequest,
				errors.New("the application owner is always a member"),
			)
			return
		}

		user, err := a.DB.GetUserByID(params.UserID)
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}
		if user == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("user does not exist"))
			return
		}

		receive := true
		if params.ReceiveNotifications != nil {
			receive = *params.ReceiveNotifications
		}
		role := params.Role
		if role == "" {
			role = model.ApplicationRoleMember
		}
		if !model.ValidApplicationRole(role) {
			ctx.AbortWithError(http.StatusBadRequest, errors.New("invalid Channel role"))
			return
		}

		membership := &model.ApplicationMembership{
			ApplicationID:        id,
			UserID:               params.UserID,
			ReceiveNotifications: receive,
			Role:                 role,
		}
		if success := successOrAbort(
			ctx,
			http.StatusInternalServerError,
			a.DB.UpsertApplicationMembership(membership),
		); !success {
			return
		}

		ctx.JSON(http.StatusOK, ApplicationMemberExternal{
			UserID:               user.ID,
			Name:                 user.Name,
			ReceiveNotifications: membership.ReceiveNotifications,
			Role:                 membership.Role,
		})
	})
}

func (a *ApplicationMembershipAPI) DeleteMember(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, ok := a.getAuthorizedApplication(ctx, id)
		if !ok {
			return
		}

		withID(ctx, "userId", func(userID uint) {
			if userID == app.UserID {
				ctx.AbortWithError(
					http.StatusBadRequest,
					errors.New("the application owner cannot be removed"),
				)
				return
			}

			membership, err := a.DB.GetApplicationMembership(id, userID)
			if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
				return
			}
			if membership == nil {
				ctx.AbortWithError(http.StatusNotFound, errors.New("membership does not exist"))
				return
			}
			if app.AutoAssign {
				ctx.AbortWithError(
					http.StatusBadRequest,
					errors.New("members cannot be removed while auto-assign is enabled"),
				)
				return
			}

			successOrAbort(
				ctx,
				http.StatusInternalServerError,
				a.DB.DeleteApplicationMembership(id, userID),
			)
		})
	})
}

func (a *ApplicationMembershipAPI) SetAutoAssign(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}

		current, err := a.DB.GetUserByID(auth.GetUserID(ctx))
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}
		if app == nil || current == nil || !current.Admin {
			ctx.AbortWithError(http.StatusNotFound, errors.New("application does not exist"))
			return
		}
		if app.Internal {
			ctx.AbortWithError(
				http.StatusBadRequest,
				errors.New("internal applications cannot be auto-assigned"),
			)
			return
		}

		params := ApplicationAutoAssignParams{}
		if err := ctx.Bind(&params); err != nil {
			return
		}
		if success := successOrAbort(
			ctx,
			http.StatusInternalServerError,
			a.DB.SetApplicationAutoAssign(id, params.Enabled),
		); !success {
			return
		}

		ctx.JSON(http.StatusOK, ApplicationAutoAssignParams{Enabled: params.Enabled})
	})
}

func (a *ApplicationMembershipAPI) GetAssignableUsers(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if _, ok := a.getAuthorizedApplication(ctx, id); !ok {
			return
		}

		users, err := a.DB.GetUsers()
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}

		result := make([]*model.UserExternal, 0, len(users))
		for _, user := range users {
			result = append(result, toExternalUser(user))
		}
		ctx.JSON(http.StatusOK, result)
	})
}

func (a *ApplicationMembershipAPI) SetCurrentUserNotifications(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		userID := auth.GetUserID(ctx)
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}
		if app == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("application does not exist"))
			return
		}

		membership, err := a.DB.GetApplicationMembership(id, userID)
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}
		if membership == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("channel membership does not exist"))
			return
		}

		params := ApplicationNotificationParams{}
		if err := ctx.Bind(&params); err != nil {
			return
		}
		if success := successOrAbort(
			ctx,
			http.StatusInternalServerError,
			a.DB.SetApplicationMembershipNotifications(id, userID, params.Enabled),
		); !success {
			return
		}
		ctx.JSON(http.StatusOK, params)
	})
}

func (a *ApplicationMembershipAPI) TransferOwnership(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, ok := a.getAuthorizedApplication(ctx, id)
		if !ok {
			return
		}
		if app.Internal {
			ctx.AbortWithError(
				http.StatusBadRequest,
				errors.New("internal applications cannot transfer ownership"),
			)
			return
		}

		params := ApplicationOwnerParams{}
		if err := ctx.Bind(&params); err != nil {
			return
		}
		if params.UserID == app.UserID {
			ctx.JSON(http.StatusOK, params)
			return
		}

		user, err := a.DB.GetUserByID(params.UserID)
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}
		if user == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("new owner does not exist"))
			return
		}

		if success := successOrAbort(
			ctx,
			http.StatusInternalServerError,
			a.DB.TransferApplicationOwnership(id, params.UserID),
		); !success {
			return
		}
		ctx.JSON(http.StatusOK, params)
	})
}

func (a *ApplicationMembershipAPI) SetMemberPosting(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}
		current, err := a.DB.GetUserByID(auth.GetUserID(ctx))
		if success := successOrAbort(ctx, http.StatusInternalServerError, err); !success {
			return
		}
		if app == nil || current == nil || !current.Admin {
			ctx.AbortWithError(http.StatusNotFound, errors.New("application does not exist"))
			return
		}
		if app.Internal {
			ctx.AbortWithError(
				http.StatusBadRequest,
				errors.New("internal applications cannot enable member posting"),
			)
			return
		}

		params := ApplicationMemberPostingParams{}
		if err := ctx.Bind(&params); err != nil {
			return
		}
		if success := successOrAbort(
			ctx,
			http.StatusInternalServerError,
			a.DB.SetApplicationMemberPosting(id, params.Enabled),
		); !success {
			return
		}
		ctx.JSON(http.StatusOK, params)
	})
}



func (a *ApplicationMembershipAPI) GetGroupGrants(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if _, ok := a.getAuthorizedApplication(ctx, id); !ok { return }
		grants, err := a.DB.GetApplicationGroupGrants(id)
		if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
		out := make([]ApplicationGroupGrantExternal, 0, len(grants))
		for _, grant := range grants {
			group, err := a.DB.GetUserGroupByID(grant.GroupID)
			if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
			if group == nil { continue }
			out = append(out, ApplicationGroupGrantExternal{
				GroupID: group.ID,
				Name: group.Name,
				Role: grant.Role,
				ReceiveNotifications: grant.ReceiveNotifications,
			})
		}
		ctx.JSON(http.StatusOK, out)
	})
}

func (a *ApplicationMembershipAPI) UpsertGroupGrant(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, ok := a.getAuthorizedApplication(ctx, id)
		if !ok { return }
		if app.Internal {
			ctx.AbortWithError(http.StatusBadRequest, errors.New("internal applications cannot be assigned to Groups"))
			return
		}
		var params ApplicationGroupGrantParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if !model.ValidApplicationRole(params.Role) {
			ctx.AbortWithError(http.StatusBadRequest, errors.New("invalid Channel role"))
			return
		}
		receive := true
		if params.ReceiveNotifications != nil { receive = *params.ReceiveNotifications }
		grant := &model.ApplicationGroupGrant{
			ApplicationID: id,
			GroupID: params.GroupID,
			Role: params.Role,
			ReceiveNotifications: receive,
		}
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.UpsertApplicationGroupGrant(grant)) { return }
		group, err := a.DB.GetUserGroupByID(params.GroupID)
		if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
		if group == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("Group does not exist"))
			return
		}
		ctx.JSON(http.StatusOK, ApplicationGroupGrantExternal{
			GroupID: group.ID,
			Name: group.Name,
			Role: grant.Role,
			ReceiveNotifications: grant.ReceiveNotifications,
		})
	})
}

func (a *ApplicationMembershipAPI) DeleteGroupGrant(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if _, ok := a.getAuthorizedApplication(ctx, id); !ok { return }
		withID(ctx, "groupId", func(groupID uint) {
			successOrAbort(ctx, http.StatusInternalServerError, a.DB.DeleteApplicationGroupGrant(id, groupID))
		})
	})
}
