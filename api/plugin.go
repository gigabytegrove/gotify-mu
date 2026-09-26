package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/location"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/plugin"
	"github.com/gotify/server/v3/plugin/compat"
	"gopkg.in/yaml.v3"
)

// The PluginDatabase interface for encapsulating database access.
type PluginDatabase interface {
	GetPluginConfByUser(userid uint) ([]*model.PluginConf, error)
	UpdatePluginConf(p *model.PluginConf) error
	GetPluginConfByID(id uint) (*model.PluginConf, error)
}

// The PluginAPI provides handlers for managing plugins.
type PluginAPI struct {
	Notifier Notifier
	Manager  *plugin.Manager
	DB       PluginDatabase
}

// InstallPlugin installs a server-wide plugin binary uploaded by an administrator.
// The route is protected by elevated administrator authentication in router.Create.
func (c *PluginAPI) InstallPlugin(ctx *gin.Context) {
	header, err := ctx.FormFile("plugin")
	if err != nil {
		ctx.AbortWithError(400, errors.New("plugin file is required"))
		return
	}
	if header.Size <= 0 {
		ctx.AbortWithError(400, errors.New("plugin file is empty"))
		return
	}
	if header.Size > plugin.MaxPluginUploadBytes {
		ctx.AbortWithError(400, fmt.Errorf("plugin exceeds the %d MiB upload limit", plugin.MaxPluginUploadBytes>>20))
		return
	}

	file, err := header.Open()
	if err != nil {
		ctx.AbortWithError(500, err)
		return
	}
	defer file.Close()

	info, warnings, checksum, err := c.Manager.InstallVerifiedPlugin(
		header.Filename,
		file,
		plugin.InstallVerification{
			ExpectedSHA256: ctx.PostForm("sha256"),
			Signature: ctx.PostForm("signature"),
			PublicKey: ctx.PostForm("publicKey"),
		},
	)
	if err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	ctx.JSON(201, gin.H{
		"name":       info.String(),
		"modulePath": info.ModulePath,
		"sha256":     checksum,
		"warnings":   warnings,
	})
}

type pluginCatalogEntry struct {
	Name        string `json:"name"`
	ModulePath  string `json:"modulePath"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Website     string `json:"website,omitempty"`
	DownloadURL string `json:"downloadUrl"`
	SHA256      string `json:"sha256"`
	Signature   string `json:"signature"`
	PublicKey   string `json:"publicKey"`
}

func loadPluginCatalog(ctx *gin.Context) ([]pluginCatalogEntry, error) {
	rawURL := strings.TrimSpace(os.Getenv("GOTIFY_MU_PLUGIN_CATALOG_URL"))
	if rawURL == "" {
		return []pluginCatalogEntry{}, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, errors.New("plugin catalog URL must use https")
	}
	request, err := http.NewRequestWithContext(ctx.Request.Context(), http.MethodGet, rawURL, nil)
	if err != nil { return nil, err }
	request.Header.Set("User-Agent", "Gotify-MU/0.5")
	client := &http.Client{Timeout:15*time.Second}
	response, err := client.Do(request)
	if err != nil { return nil, err }
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plugin catalog returned HTTP %d", response.StatusCode)
	}
	var entries []pluginCatalogEntry
	decoder := json.NewDecoder(io.LimitReader(response.Body, 2<<20))
	if err := decoder.Decode(&entries); err != nil { return nil, err }
	for _, entry := range entries {
		if strings.TrimSpace(entry.ModulePath)=="" || strings.TrimSpace(entry.DownloadURL)=="" ||
			strings.TrimSpace(entry.SHA256)=="" || strings.TrimSpace(entry.Signature)=="" ||
			strings.TrimSpace(entry.PublicKey)=="" {
			return nil, errors.New("plugin catalog contains an incomplete entry")
		}
	}
	return entries, nil
}

func (c *PluginAPI) GetCatalog(ctx *gin.Context) {
	entries, err := loadPluginCatalog(ctx)
	if !successOrAbort(ctx, http.StatusBadGateway, err) { return }
	type catalogView struct {
		pluginCatalogEntry
		Installed bool `json:"installed"`
	}
	out := make([]catalogView,0,len(entries))
	for _, entry := range entries {
		out = append(out,catalogView{pluginCatalogEntry:entry,Installed:c.Manager.HasPlugin(entry.ModulePath)})
	}
	ctx.JSON(http.StatusOK,out)
}

type catalogInstallParams struct {
	ModulePath string `json:"modulePath" binding:"required"`
	Version    string `json:"version"`
}

func (c *PluginAPI) InstallCatalogPlugin(ctx *gin.Context) {
	var params catalogInstallParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	entries, err := loadPluginCatalog(ctx)
	if !successOrAbort(ctx, http.StatusBadGateway, err) { return }
	var selected *pluginCatalogEntry
	for i := range entries {
		if entries[i].ModulePath == params.ModulePath && (params.Version=="" || entries[i].Version==params.Version) {
			selected=&entries[i];break
		}
	}
	if selected==nil { ctx.AbortWithError(http.StatusNotFound,errors.New("plugin not found in configured catalog"));return }
	parsed, err := url.Parse(selected.DownloadURL)
	if err != nil || parsed.Scheme!="https" || parsed.Host=="" {
		ctx.AbortWithError(http.StatusBadRequest,errors.New("plugin download URL must use https"));return
	}
	request, err := http.NewRequestWithContext(ctx.Request.Context(),http.MethodGet,selected.DownloadURL,nil)
	if !successOrAbort(ctx,500,err){return}
	request.Header.Set("User-Agent","Gotify-MU/0.5")
	response, err := (&http.Client{Timeout:2*time.Minute}).Do(request)
	if !successOrAbort(ctx,http.StatusBadGateway,err){return}
	defer response.Body.Close()
	if response.StatusCode!=http.StatusOK{ctx.AbortWithError(http.StatusBadGateway,fmt.Errorf("plugin download returned HTTP %d",response.StatusCode));return}
	verification:=plugin.InstallVerification{ExpectedSHA256:selected.SHA256,Signature:selected.Signature,PublicKey:selected.PublicKey}
	filename:=filepath.Base(parsed.Path)
	if filename==""||filename=="."||!strings.HasSuffix(strings.ToLower(filename),".so"){filename="plugin.so"}
	if c.Manager.HasPlugin(selected.ModulePath) {
		checksum, stageErr:=c.Manager.StagePluginUpdate(selected.ModulePath,filename,response.Body,verification)
		if !successOrAbort(ctx,400,stageErr){return}
		ctx.JSON(202,gin.H{"modulePath":selected.ModulePath,"sha256":checksum,"restartRequired":true})
		return
	}
	info,warnings,checksum,installErr:=c.Manager.InstallVerifiedPlugin(filename,response.Body,verification)
	if !successOrAbort(ctx,400,installErr){return}
	ctx.JSON(201,gin.H{"name":info.String(),"modulePath":info.ModulePath,"sha256":checksum,"warnings":warnings})
}

func (c *PluginAPI) UninstallPlugin(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){
		conf,err:=c.DB.GetPluginConfByID(id)
		if !successOrAbort(ctx,500,err){return}
		if conf==nil{ctx.AbortWithStatus(404);return}
		if !successOrAbort(ctx,500,c.Manager.UninstallPlugin(conf.ModulePath)){return}
		ctx.Status(http.StatusNoContent)
	})
}

func (c *PluginAPI) StagePluginUpdate(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){
		conf,err:=c.DB.GetPluginConfByID(id)
		if !successOrAbort(ctx,500,err){return}
		if conf==nil{ctx.AbortWithStatus(404);return}
		header,err:=ctx.FormFile("plugin")
		if err!=nil{ctx.AbortWithError(400,errors.New("plugin file is required"));return}
		file,err:=header.Open()
		if !successOrAbort(ctx,500,err){return}
		defer file.Close()
		checksum,err:=c.Manager.StagePluginUpdate(conf.ModulePath,header.Filename,file,plugin.InstallVerification{
			ExpectedSHA256:ctx.PostForm("sha256"),Signature:ctx.PostForm("signature"),PublicKey:ctx.PostForm("publicKey"),
		})
		if !successOrAbort(ctx,400,err){return}
		ctx.JSON(202,gin.H{"sha256":checksum,"restartRequired":true})
	})
}

// GetPlugins returns all plugins a user has.
// swagger:operation GET /plugin plugin getPlugins
//
// Return all plugins.
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
//	        $ref: "#/definitions/PluginConf"
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
//	    description: Internal Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (c *PluginAPI) GetPlugins(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	plugins, err := c.DB.GetPluginConfByUser(userID)
	if success := successOrAbort(ctx, 500, err); !success {
		return
	}
	result := make([]model.PluginConfExternal, 0)
	for _, conf := range plugins {
		if inst, err := c.Manager.Instance(conf.ID); err == nil {
			info := c.Manager.PluginInfo(conf.ModulePath)
			result = append(result, model.PluginConfExternal{
				ID:           conf.ID,
				CreatedAt:    conf.CreatedAt,
				Name:         info.String(),
				Token:        conf.Token,
				ModulePath:   conf.ModulePath,
				Author:       info.Author,
				Website:      info.Website,
				License:      info.License,
				Enabled:      conf.Enabled,
				Capabilities: inst.Supports().Strings(),
			})
		}
	}
	ctx.JSON(200, result)
}

// EnablePlugin enables a plugin.
// swagger:operation POST /plugin/{id}/enable plugin enablePlugin
//
// Enable a plugin.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	parameters:
//	- name: id
//	  in: path
//	  description: the plugin id
//	  required: true
//	  type: integer
//	  format: int64
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	responses:
//	  200:
//	    description: Ok
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
//	    description: Internal Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (c *PluginAPI) EnablePlugin(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		conf, err := c.DB.GetPluginConfByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if conf == nil || !isPluginOwner(ctx, conf) {
			ctx.AbortWithError(404, errors.New("unknown plugin"))
			return
		}
		_, err = c.Manager.Instance(id)
		if err != nil {
			ctx.AbortWithError(404, errors.New("plugin instance not found"))
			return
		}
		if err := c.Manager.SetPluginEnabled(id, true); err == plugin.ErrAlreadyEnabledOrDisabled {
			ctx.AbortWithError(400, err)
		} else if err != nil {
			ctx.AbortWithError(500, err)
		}
	})
}

// DisablePlugin disables a plugin.
// swagger:operation POST /plugin/{id}/disable plugin disablePlugin
//
// Disable a plugin.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	parameters:
//	- name: id
//	  in: path
//	  description: the plugin id
//	  required: true
//	  type: integer
//	  format: int64
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	responses:
//	  200:
//	    description: Ok
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
//	    description: Internal Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (c *PluginAPI) DisablePlugin(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		conf, err := c.DB.GetPluginConfByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if conf == nil || !isPluginOwner(ctx, conf) {
			ctx.AbortWithError(404, errors.New("unknown plugin"))
			return
		}
		_, err = c.Manager.Instance(id)
		if err != nil {
			ctx.AbortWithError(404, errors.New("plugin instance not found"))
			return
		}
		if err := c.Manager.SetPluginEnabled(id, false); err == plugin.ErrAlreadyEnabledOrDisabled {
			ctx.AbortWithError(400, err)
		} else if err != nil {
			ctx.AbortWithError(500, err)
		}
	})
}

// GetDisplay get display info for Displayer plugin.
// swagger:operation GET /plugin/{id}/display plugin getPluginDisplay
//
// Get display info for a Displayer plugin.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	parameters:
//	- name: id
//	  in: path
//	  description: the plugin id
//	  required: true
//	  type: integer
//	  format: int64
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	      type: string
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
//	    description: Internal Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (c *PluginAPI) GetDisplay(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		conf, err := c.DB.GetPluginConfByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if conf == nil || !isPluginOwner(ctx, conf) {
			ctx.AbortWithError(404, errors.New("unknown plugin"))
			return
		}
		instance, err := c.Manager.Instance(id)
		if err != nil {
			ctx.AbortWithError(404, errors.New("plugin instance not found"))
			return
		}
		ctx.JSON(200, instance.GetDisplay(location.Get(ctx)))
	})
}

// GetConfig returns Configurer plugin configuration in YAML format.
// swagger:operation GET /plugin/{id}/config plugin getPluginConfig
//
// Get YAML configuration for Configurer plugin.
//
//	---
//	consumes: [application/json]
//	produces: [application/x-yaml]
//	parameters:
//	- name: id
//	  in: path
//	  description: the plugin id
//	  required: true
//	  type: integer
//	  format: int64
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	        type: object
//	        description: plugin configuration
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
//	    description: Internal Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (c *PluginAPI) GetConfig(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		conf, err := c.DB.GetPluginConfByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if conf == nil || !isPluginOwner(ctx, conf) {
			ctx.AbortWithError(404, errors.New("unknown plugin"))
			return
		}
		instance, err := c.Manager.Instance(id)
		if err != nil {
			ctx.AbortWithError(404, errors.New("plugin instance not found"))
			return
		}

		if aborted := supportOrAbort(ctx, instance, compat.Configurer); aborted {
			return
		}

		ctx.Header("content-type", "application/x-yaml")
		ctx.Writer.Write(conf.Config)
	})
}

// UpdateConfig updates Configurer plugin configuration in YAML format.
// swagger:operation POST /plugin/{id}/config plugin updatePluginConfig
//
// Update YAML configuration for Configurer plugin.
//
//	---
//	consumes: [application/x-yaml]
//	produces: [application/json]
//	parameters:
//	- name: id
//	  in: path
//	  description: the plugin id
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
//	    description: Internal Server Error
//	    schema:
//	        $ref: "#/definitions/Error"
func (c *PluginAPI) UpdateConfig(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		conf, err := c.DB.GetPluginConfByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if conf == nil || !isPluginOwner(ctx, conf) {
			ctx.AbortWithError(404, errors.New("unknown plugin"))
			return
		}
		instance, err := c.Manager.Instance(id)
		if err != nil {
			ctx.AbortWithError(404, errors.New("plugin instance not found"))
			return
		}

		if aborted := supportOrAbort(ctx, instance, compat.Configurer); aborted {
			return
		}

		newConf := instance.DefaultConfig()
		newconfBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.AbortWithError(500, err)
			return
		}
		if err := yaml.Unmarshal(newconfBytes, newConf); err != nil {
			ctx.AbortWithError(400, err)
			return
		}
		if err := instance.ValidateAndSetConfig(newConf); err != nil {
			ctx.AbortWithError(400, err)
			return
		}
		conf.Config = newconfBytes
		successOrAbort(ctx, 500, c.DB.UpdatePluginConf(conf))
	})
}

func isPluginOwner(ctx *gin.Context, conf *model.PluginConf) bool {
	return conf.UserID == auth.GetUserID(ctx)
}

func supportOrAbort(ctx *gin.Context, instance compat.PluginInstance, module compat.Capability) (aborted bool) {
	if compat.HasSupport(instance, module) {
		return false
	}
	ctx.AbortWithError(400, fmt.Errorf("plugin does not support %s", module))
	return true
}
