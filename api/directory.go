package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/directory"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
)

type DirectoryDatabase interface {
	GetDirectoryConfig() (*model.DirectoryConfig, error)
	SaveDirectoryConfig(item *model.DirectoryConfig) error
}

type DirectoryAPI struct {
	DB      DirectoryDatabase
	Service *directory.Service
}

type directoryParams struct {
	Enabled              bool   `json:"enabled"`
	URL                  string `json:"url"`
	StartTLS             bool   `json:"startTls"`
	BindDN               string `json:"bindDn"`
	BindPassword         string `json:"bindPassword"`
	UserBaseDN           string `json:"userBaseDn"`
	UserAttribute        string `json:"userAttribute"`
	DisplayNameAttribute string `json:"displayNameAttribute"`
	AdminGroupDN         string `json:"adminGroupDn"`
	AutoRegister         bool   `json:"autoRegister"`
	LinkByUsername       bool   `json:"linkByUsername"`
	CACertificatePEM     string `json:"caCertificatePem"`
}

type directoryTestParams struct {
	directoryParams
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (a *DirectoryAPI) Get(ctx *gin.Context) {
	item, err := a.DB.GetDirectoryConfig()
	if !successOrAbort(ctx, 500, err) { return }
	if item == nil {
		item = &model.DirectoryConfig{
			UserAttribute: "uid",
			DisplayNameAttribute: "displayName",
			AutoRegister: true,
		}
	}
	ctx.JSON(200, directoryView(item))
}

func (a *DirectoryAPI) Save(ctx *gin.Context) {
	var params directoryParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }

	existing, err := a.DB.GetDirectoryConfig()
	if !successOrAbort(ctx, 500, err) { return }
	item, err := directoryConfigFromParams(params, existing)
	if err != nil {
		ctx.AbortWithError(400, err)
		return
	}
	if !successOrAbort(ctx, 500, a.DB.SaveDirectoryConfig(item)) { return }
	ctx.JSON(200, directoryView(item))
}

func (a *DirectoryAPI) Test(ctx *gin.Context) {
	if a.Service == nil {
		ctx.AbortWithError(503, errors.New("directory authentication service is unavailable"))
		return
	}
	var params directoryTestParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	existing, err := a.DB.GetDirectoryConfig()
	if !successOrAbort(ctx, 500, err) { return }
	item, err := directoryConfigFromParams(params.directoryParams, existing)
	if err != nil {
		ctx.AbortWithError(400, err)
		return
	}
	if err := a.Service.Test(item, params.Username, params.Password); err != nil {
		ctx.AbortWithError(400, err)
		return
	}
	ctx.JSON(200, gin.H{"connected": true})
}

func directoryConfigFromParams(params directoryParams, existing *model.DirectoryConfig) (*model.DirectoryConfig, error) {
	item := &model.DirectoryConfig{}
	if existing != nil { *item = *existing }
	item.Enabled = params.Enabled
	item.URL = strings.TrimSpace(params.URL)
	item.StartTLS = params.StartTLS
	item.BindDN = strings.TrimSpace(params.BindDN)
	item.UserBaseDN = strings.TrimSpace(params.UserBaseDN)
	item.UserAttribute = valueOr(params.UserAttribute, "uid")
	item.DisplayNameAttribute = valueOr(params.DisplayNameAttribute, "displayName")
	item.AdminGroupDN = strings.TrimSpace(params.AdminGroupDN)
	item.AutoRegister = params.AutoRegister
	item.LinkByUsername = params.LinkByUsername
	item.CACertificatePEM = strings.TrimSpace(params.CACertificatePEM)

	if item.Enabled {
		lower := strings.ToLower(item.URL)
		if !strings.HasPrefix(lower, "ldap://") && !strings.HasPrefix(lower, "ldaps://") {
			return nil, errors.New("directory URL must use ldap:// or ldaps://")
		}
		if item.UserBaseDN == "" {
			return nil, errors.New("directory user base DN is required")
		}
	}
	if strings.TrimSpace(params.BindPassword) != "" {
		protected, err := security.Protect(params.BindPassword)
		if err != nil { return nil, err }
		item.BindPassword = protected
	}
	return item, nil
}

func directoryView(item *model.DirectoryConfig) model.DirectoryConfigView {
	return model.DirectoryConfigView{
		ID:item.ID,
		Enabled:item.Enabled,
		URL:item.URL,
		StartTLS:item.StartTLS,
		BindDN:item.BindDN,
		BindPasswordConfigured:item.BindPassword!="",
		UserBaseDN:item.UserBaseDN,
		UserAttribute:item.UserAttribute,
		DisplayNameAttribute:item.DisplayNameAttribute,
		AdminGroupDN:item.AdminGroupDN,
		AutoRegister:item.AutoRegister,
		LinkByUsername:item.LinkByUsername,
		CACertificatePEM:item.CACertificatePEM,
		CreatedAt:item.CreatedAt,
		UpdatedAt:item.UpdatedAt,
	}
}
