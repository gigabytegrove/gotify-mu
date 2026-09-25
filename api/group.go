package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
)

type UserGroupDatabase interface {
	GetUserGroups() ([]*model.UserGroup, error)
	GetUserGroupByID(id uint) (*model.UserGroup, error)
	CreateUserGroup(group *model.UserGroup) error
	UpdateUserGroup(group *model.UserGroup) error
	DeleteUserGroup(id uint) error
	CountUserGroupMembers(groupID uint) (int64, error)
	GetUserGroupMembers(groupID uint) ([]*model.User, error)
	AddUserGroupMember(groupID, userID uint) error
	RemoveUserGroupMember(groupID, userID uint) error
}

type UserGroupAPI struct {
	DB UserGroupDatabase
}

type userGroupInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type groupMemberInput struct {
	UserID uint `json:"userId" binding:"required"`
}

func (a *UserGroupAPI) GetGroups(ctx *gin.Context) {
	groups, err := a.DB.GetUserGroups()
	if success := successOrAbort(ctx, 500, err); !success {
		return
	}

	out := make([]model.UserGroupExternal, 0, len(groups))
	for _, group := range groups {
		count, err := a.DB.CountUserGroupMembers(group.ID)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		out = append(out, model.UserGroupExternal{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			MemberCount: count,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		})
	}
	ctx.JSON(http.StatusOK, out)
}

func (a *UserGroupAPI) CreateGroup(ctx *gin.Context) {
	var input userGroupInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	group := &model.UserGroup{Name: input.Name, Description: input.Description}
	if success := successOrAbort(ctx, 500, a.DB.CreateUserGroup(group)); !success {
		return
	}
	ctx.JSON(http.StatusCreated, model.UserGroupExternal{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		MemberCount: 0,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	})
}

func (a *UserGroupAPI) UpdateGroup(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		group, err := a.DB.GetUserGroupByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if group == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("group does not exist"))
			return
		}

		var input userGroupInput
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}

		group.Name = input.Name
		group.Description = input.Description
		if success := successOrAbort(ctx, 500, a.DB.UpdateUserGroup(group)); !success {
			return
		}

		count, err := a.DB.CountUserGroupMembers(group.ID)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		ctx.JSON(http.StatusOK, model.UserGroupExternal{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			MemberCount: count,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		})
	})
}

func (a *UserGroupAPI) DeleteGroup(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		group, err := a.DB.GetUserGroupByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if group == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("group does not exist"))
			return
		}
		successOrAbort(ctx, 500, a.DB.DeleteUserGroup(id))
	})
}

func (a *UserGroupAPI) GetMembers(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		group, err := a.DB.GetUserGroupByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if group == nil {
			ctx.AbortWithError(http.StatusNotFound, errors.New("group does not exist"))
			return
		}

		users, err := a.DB.GetUserGroupMembers(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		out := make([]model.UserGroupMemberExternal, 0, len(users))
		for _, user := range users {
			out = append(out, model.UserGroupMemberExternal{
				UserID:      user.ID,
				Name:        user.Name,
				DisplayName: user.DisplayName,
				Admin:       user.Admin,
			})
		}
		ctx.JSON(http.StatusOK, out)
	})
}

func (a *UserGroupAPI) AddMember(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		var input groupMemberInput
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
		if success := successOrAbort(ctx, 500, a.DB.AddUserGroupMember(id, input.UserID)); !success {
			return
		}
		ctx.Status(http.StatusNoContent)
	})
}

func (a *UserGroupAPI) RemoveMember(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		withID(ctx, "userId", func(userID uint) {
			successOrAbort(ctx, 500, a.DB.RemoveUserGroupMember(id, userID))
		})
	})
}
