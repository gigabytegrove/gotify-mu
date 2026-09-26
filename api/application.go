package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/h2non/filetype"
	"gorm.io/gorm"
)

// The ApplicationDatabase interface for encapsulating database access.
type ApplicationDatabase interface {
	CreateApplication(application *model.Application) error
	GetApplicationByToken(token string) (*model.Application, error)
	GetApplicationByID(id uint) (*model.Application, error)
	GetApplicationsByUser(userID uint) ([]*model.Application, error)
	GetAccessibleApplicationsByUser(userID uint) ([]*model.Application, error)
	GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error)
	DeleteApplicationByID(id uint) error
	UpdateApplication(application *model.Application) error
	GetUserByID(id uint) (*model.User, error)
}

// The ApplicationAPI provides handlers for managing applications.
type ApplicationAPI struct {
	DB       ApplicationDatabase
	ImageDir string
	OnDelete func(uint)
}

// Application Params Model
//
// Params allowed to create or update Applications.
//
// swagger:model ApplicationParams
type ApplicationParams struct {
	// The application name. This is how the application should be displayed to the user.
	//
	// required: true
	// example: Backup Server
	Name string `form:"name" query:"name" json:"name" binding:"required"`
	// The description of the application.
	//
	// example: Backup server for the interwebs
	Description string `form:"description" query:"description" json:"description"`
	// The default priority of messages sent by this application. Defaults to 0.
	//
	// example: 5
	DefaultPriority int `form:"defaultPriority" query:"defaultPriority" json:"defaultPriority"`
	// The sortKey for the application. Uses fractional indexing.
	//
	// example: a1
	SortKey string `form:"sortKey" query:"sortKey" json:"sortKey"`
	// Whether this Gotify MU channel should be automatically assigned to every user.
	AutoAssign bool `form:"autoAssign" query:"autoAssign" json:"autoAssign"`
	// Whether assigned users may publish messages to this Gotify MU channel.
	AllowMemberPost bool `form:"allowMemberPost" query:"allowMemberPost" json:"allowMemberPost"`
	// Number of days to retain message history. Zero keeps messages indefinitely.
	RetentionDays int `form:"retentionDays" query:"retentionDays" json:"retentionDays" binding:"min=0,max=36500"`
}

// CreateApplication creates an application and returns the access token.
// swagger:operation POST /application application createApp
//
// Create an application.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: body
//	  in: body
//	  description: the application to add
//	  required: true
//	  schema:
//	    $ref: "#/definitions/ApplicationParams"
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	        $ref: "#/definitions/Application"
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *ApplicationAPI) CreateApplication(ctx *gin.Context) {
	applicationParams := ApplicationParams{}
	if err := ctx.Bind(&applicationParams); err == nil {
		if applicationParams.AutoAssign || applicationParams.AllowMemberPost {
			current, err := a.DB.GetUserByID(auth.GetUserID(ctx))
			if success := successOrAbort(ctx, 500, err); !success {
				return
			}
			if current == nil || !current.Admin {
				ctx.AbortWithError(
					http.StatusForbidden,
					errors.New("only administrators can create global or member-posting channels"),
				)
				return
			}
		}
		tokenPublic, tokenPrivate := generateApplicationToken()
		app := model.Application{
			Name:            applicationParams.Name,
			Description:     applicationParams.Description,
			DefaultPriority: applicationParams.DefaultPriority,
			SortKey:         applicationParams.SortKey,
			Token:           tokenPublic,
			UserID:          auth.GetUserID(ctx),
			Internal:        false,
			AutoAssign:      applicationParams.AutoAssign,
			AllowMemberPost: applicationParams.AllowMemberPost,
			RetentionDays: applicationParams.RetentionDays,
		}

		if err := a.DB.CreateApplication(&app); err != nil {
			handleApplicationError(ctx, err)
			return
		}
		app.Token = tokenPrivate
		ctx.JSON(200, withResolvedImage(&app))
	}
}

// GetApplications returns all applications a user has.
// swagger:operation GET /application application getApps
//
// Return all applications.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	      type: array
//	      items:
//	        $ref: "#/definitions/Application"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *ApplicationAPI) GetApplications(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	apps, err := a.DB.GetAccessibleApplicationsByUser(userID)
	if success := successOrAbort(ctx, 500, err); !success {
		return
	}
	for _, app := range apps {
		membership, err := a.DB.GetApplicationMembership(app.ID, userID)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if membership != nil {
			receiveNotifications := membership.ReceiveNotifications
			app.ReceiveNotifications = &receiveNotifications
		}
		app.CurrentRole = model.EffectiveChannelRole(app.UserID == userID, membership)
		app.Token = ""
		withResolvedImage(app)
	}
	ctx.JSON(200, apps)
}

// DeleteApplication deletes an application by its id.
// swagger:operation DELETE /application/{id} application deleteApp
//
// Delete an application.
//
// Requires elevated authentication.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	parameters:
//	- name: id
//	  in: path
//	  description: the application id
//	  required: true
//	  type: integer
//	  format: int64
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	responses:
//	  200:
//	    description: Ok
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
//	  404:
//	    description: Not Found
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *ApplicationAPI) DeleteApplication(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		allowed, err := a.isOwnerOrAdmin(auth.GetUserID(ctx), app)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if app != nil && allowed {
			if app.Internal {
				ctx.AbortWithError(400, errors.New("cannot delete internal application"))
				return
			}
			if app.AutoAssign {
				current, err := a.DB.GetUserByID(auth.GetUserID(ctx))
				if success := successOrAbort(ctx, 500, err); !success {
					return
				}
				if current == nil || !current.Admin {
					ctx.AbortWithError(
						http.StatusForbidden,
						errors.New("global channels can only be deleted by an administrator"),
					)
					return
				}
			}
			if success := successOrAbort(ctx, 500, a.DB.DeleteApplicationByID(id)); !success {
				return
			}
			if a.OnDelete != nil {
				a.OnDelete(id)
			}
			if app.Image != "" {
				os.Remove(a.ImageDir + app.Image)
			}
		} else {
			ctx.AbortWithError(404, fmt.Errorf("app with id %d doesn't exists", id))
		}
	})
}

// UpdateApplication updates an application info by its id.
// swagger:operation PUT /application/{id} application updateApplication
//
// Update an application.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: body
//	  in: body
//	  description: the application to update
//	  required: true
//	  schema:
//	    $ref: "#/definitions/ApplicationParams"
//	- name: id
//	  in: path
//	  description: the application id
//	  required: true
//	  type: integer
//	  format: int64
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	        $ref: "#/definitions/Application"
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
//	  404:
//	    description: Not Found
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *ApplicationAPI) UpdateApplication(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		allowed, err := a.canManageApplication(auth.GetUserID(ctx), app)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if app != nil && allowed {
			applicationParams := ApplicationParams{}
			if err := ctx.Bind(&applicationParams); err == nil {
				app.Description = applicationParams.Description
				app.Name = applicationParams.Name
				app.DefaultPriority = applicationParams.DefaultPriority
				app.RetentionDays = applicationParams.RetentionDays
				if applicationParams.SortKey != "" {
					app.SortKey = applicationParams.SortKey
				}

				if err := a.DB.UpdateApplication(app); err != nil {
					handleApplicationError(ctx, err)
					return
				}
				ctx.JSON(200, withResolvedImage(app))
			}
		} else {
			ctx.AbortWithError(404, fmt.Errorf("app with id %d doesn't exists", id))
		}
	})
}

// UpdateApplicationSecurity performs security updates on an application.
// swagger:operation PUT /application/{id}/security application updateAppSecurity
//
// Perform security updates on an application.
//
// Requires elevated authentication.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: body
//	  in: body
//	  description: security update action descriptor
//	  required: true
//	  schema:
//	    $ref: "#/definitions/SecurityUpdateAction"
//	- name: id
//	  in: path
//	  description: the application id
//	  required: true
//	  type: integer
//	  format: int64
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	        $ref: "#/definitions/SecurityUpdateActionResponse"
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
//	  404:
//	    description: Not Found
//	    schema:
//	        $ref: "#/definitions/Error"
//	  500:
//	    description: Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *ApplicationAPI) UpdateApplicationSecurity(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		allowed, err := a.isOwnerOrAdmin(auth.GetUserID(ctx), app)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if app == nil || !allowed {
			ctx.AbortWithError(404, fmt.Errorf("app with id %d doesn't exists", id))
			return
		}

		action := model.SecurityUpdateAction{}
		response := model.SecurityUpdateActionResponse{}
		if err := ctx.Bind(&action); err == nil {
			if action.RegenerateToken {
				tokenPublic, tokenPrivate := generateApplicationToken()
				app.Token = tokenPublic
				response.RegenerateToken = &model.RegenerateTokenResponse{
					Token: tokenPrivate,
				}
			}
			if success := successOrAbort(ctx, 500, a.DB.UpdateApplication(app)); !success {
				return
			}
			ctx.JSON(200, response)
		}
	})
}

// UploadApplicationImage uploads an image for an application.
// swagger:operation POST /application/{id}/image application uploadAppImage
//
// Upload an image for an application.
//
//	---
//	consumes:
//	- multipart/form-data
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: file
//	  in: formData
//	  description: the application image
//	  required: true
//	  type: file
//	- name: id
//	  in: path
//	  description: the application id
//	  required: true
//	  type: integer
//	  format: int64
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	        $ref: "#/definitions/Application"
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
//	  404:
//	    description: Not Found
//	    schema:
//	        $ref: "#/definitions/Error"
//	  500:
//	    description: Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *ApplicationAPI) UploadApplicationImage(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		allowed, err := a.canManageApplication(auth.GetUserID(ctx), app)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if app != nil && allowed {
			file, err := ctx.FormFile("file")
			if err == http.ErrMissingFile {
				ctx.AbortWithError(400, errors.New("file with key 'file' must be present"))
				return
			} else if err != nil {
				ctx.AbortWithError(500, err)
				return
			}
			head := make([]byte, 261)
			open, _ := file.Open()
			open.Read(head)
			if !filetype.IsImage(head) {
				ctx.AbortWithError(400, errors.New("file must be an image"))
				return
			}

			ext := filepath.Ext(file.Filename)
			if !ValidApplicationImageExt(ext) {
				ctx.AbortWithError(400, errors.New("invalid file extension"))
				return
			}

			name := generateNonExistingImageName(a.ImageDir, func() string {
				return generateImageName() + ext
			})

			err = ctx.SaveUploadedFile(file, a.ImageDir+name)
			if err != nil {
				ctx.AbortWithError(500, err)
				return
			}

			if app.Image != "" {
				os.Remove(a.ImageDir + app.Image)
			}

			app.Image = name
			if success := successOrAbort(ctx, 500, a.DB.UpdateApplication(app)); !success {
				return
			}
			ctx.JSON(200, withResolvedImage(app))
		} else {
			ctx.AbortWithError(404, fmt.Errorf("app with id %d doesn't exists", id))
		}
	})
}

// RemoveApplicationImage deletes an image of an application.
// swagger:operation DELETE /application/{id}/image application removeAppImage
//
// Deletes an image of an application.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	parameters:
//	- name: id
//	  in: path
//	  description: the application id
//	  required: true
//	  type: integer
//	  format: int64
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	responses:
//	  200:
//	    description: Ok
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
//	  404:
//	    description: Not Found
//	    schema:
//	        $ref: "#/definitions/Error"
//	  500:
//	    description: Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *ApplicationAPI) RemoveApplicationImage(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		app, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		allowed, err := a.canManageApplication(auth.GetUserID(ctx), app)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if app != nil && allowed {
			if app.Image == "" {
				ctx.AbortWithError(400, fmt.Errorf("app with id %d does not have a customized image", id))
				return
			}

			image := app.Image
			app.Image = ""
			if success := successOrAbort(ctx, 500, a.DB.UpdateApplication(app)); !success {
				return
			}
			os.Remove(a.ImageDir + image)
			ctx.JSON(200, withResolvedImage(app))
		} else {
			ctx.AbortWithError(404, fmt.Errorf("app with id %d doesn't exists", id))
		}
	})
}

func withResolvedImage(app *model.Application) *model.Application {
	if app.Image == "" {
		// This must stay in sync with the isDefaultImage check in ui/src/application/Applications.tsx.
		app.Image = "static/defaultapp.png"
	} else {
		app.Image = "image/" + app.Image
	}
	return app
}

func exist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func generateNonExistingImageName(imgDir string, gen func() string) string {
	for {
		name := gen()
		if !exist(imgDir + name) {
			return name
		}
	}
}

func ValidApplicationImageExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".gif", ".png", ".jpg", ".jpeg":
		return true
	default:
		return false
	}
}

func handleApplicationError(ctx *gin.Context, err error) {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		ctx.AbortWithError(400, errors.New("sort key is not unique"))
	} else {
		ctx.AbortWithError(500, err)
	}
}

func (a *ApplicationAPI) canManageApplication(userID uint, app *model.Application) (bool, error) {
	if app == nil { return false, nil }
	if app.UserID == userID { return true, nil }
	user, err := a.DB.GetUserByID(userID)
	if err != nil { return false, err }
	if user != nil && user.Admin { return true, nil }
	membership, err := a.DB.GetApplicationMembership(app.ID, userID)
	if err != nil { return false, err }
	return membership != nil && membership.EffectiveRole == model.ChannelRoleManager, nil
}

func (a *ApplicationAPI) isOwnerOrAdmin(userID uint, app *model.Application) (bool, error) {
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
	return user != nil && user.Admin, nil
}
