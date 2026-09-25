package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
)

type ServiceAccountDatabase interface {
	GetServiceAccounts() ([]*model.ServiceAccount,error)
	GetServiceAccountByID(id uint)(*model.ServiceAccount,error)
	SaveServiceAccount(item *model.ServiceAccount)error
	DeleteServiceAccount(id uint)error
	GetServiceAccountTokens(accountID uint)([]*model.ServiceAccountToken,error)
	SaveServiceAccountToken(item *model.ServiceAccountToken)error
	DeleteServiceAccountToken(accountID,id uint)error
	GetApplications() ([]*model.Application,error)
	GetMessagesByApplicationSince(appID uint, limit int, since uint) ([]*model.Message,error)
}

type ServicePublisher interface {
	Publish(applicationID uint,title,message string,priority int)(*model.Message,error)
}

type ServiceAccountAPI struct {
	DB ServiceAccountDatabase
	Publisher ServicePublisher
}

var allowedServiceScopes=map[string]bool{
	"message:write":true,
	"message:read":true,
	"channel:read":true,
}

type serviceAccountParams struct {
	Name string `json:"name" binding:"required"`
	Scopes string `json:"scopes" binding:"required"`
	AllowedApplicationIDs string `json:"allowedApplicationIds"`
	Enabled bool `json:"enabled"`
}

func normalizeServiceScopes(raw string)(string,error){
	var result []string
	seen:=map[string]bool{}
	for _,scope:=range strings.FieldsFunc(raw,func(r rune)bool{return r==','||r==';'||r==' '||r=='\n'||r=='\t'}){
		scope=strings.TrimSpace(scope)
		if !allowedServiceScopes[scope]{return "",errors.New("unsupported service scope "+scope)}
		if !seen[scope]{seen[scope]=true;result=append(result,scope)}
	}
	if len(result)==0{return "",errors.New("at least one service scope is required")}
	return strings.Join(result,","),nil
}
func normalizeServiceApplications(raw string) (string,error) {
	raw=strings.TrimSpace(raw);if raw==""||raw=="*"{return raw,nil}
	var ids []string;seen:=map[string]bool{}
	for _,part:=range strings.FieldsFunc(raw,func(r rune)bool{return r==','||r==';'||r==' '}){
		part=strings.TrimSpace(part);id,err:=strconv.ParseUint(part,10,64);if err!=nil||id==0{return "",errors.New("allowed Channel ids must be positive numbers")}
		if !seen[part]{seen[part]=true;ids=append(ids,part)}
	}
	return strings.Join(ids,","),nil
}

func (a *ServiceAccountAPI) GetAccounts(ctx *gin.Context){items,err:=a.DB.GetServiceAccounts();if !successOrAbort(ctx,500,err){return};ctx.JSON(200,items)}
func (a *ServiceAccountAPI) CreateAccount(ctx *gin.Context){
	var p serviceAccountParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return}
	scopes,err:=normalizeServiceScopes(p.Scopes);if err!=nil{ctx.AbortWithError(400,err);return}
	apps,err:=normalizeServiceApplications(p.AllowedApplicationIDs);if err!=nil{ctx.AbortWithError(400,err);return}
	item:=&model.ServiceAccount{Name:strings.TrimSpace(p.Name),Scopes:scopes,AllowedApplicationIDs:apps,Enabled:p.Enabled,CreatedBy:auth.GetUserID(ctx)}
	if !successOrAbort(ctx,500,a.DB.SaveServiceAccount(item)){return};ctx.JSON(201,item)
}
func (a *ServiceAccountAPI) UpdateAccount(ctx *gin.Context){withID(ctx,"id",func(id uint){
	item,err:=a.DB.GetServiceAccountByID(id);if !successOrAbort(ctx,500,err){return};if item==nil{ctx.AbortWithStatus(404);return}
	var p serviceAccountParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return}
	scopes,err:=normalizeServiceScopes(p.Scopes);if err!=nil{ctx.AbortWithError(400,err);return};apps,err:=normalizeServiceApplications(p.AllowedApplicationIDs);if err!=nil{ctx.AbortWithError(400,err);return}
	item.Name=strings.TrimSpace(p.Name);item.Scopes=scopes;item.AllowedApplicationIDs=apps;item.Enabled=p.Enabled
	if !successOrAbort(ctx,500,a.DB.SaveServiceAccount(item)){return};ctx.JSON(200,item)
})}
func (a *ServiceAccountAPI) DeleteAccount(ctx *gin.Context){withID(ctx,"id",func(id uint){if !successOrAbort(ctx,500,a.DB.DeleteServiceAccount(id)){return};ctx.Status(http.StatusNoContent)})}

type serviceTokenParams struct {
	Name string `json:"name" binding:"required"`
	ExpiresInDays int `json:"expiresInDays"`
}
func (a *ServiceAccountAPI) GetTokens(ctx *gin.Context){withID(ctx,"id",func(id uint){items,err:=a.DB.GetServiceAccountTokens(id);if !successOrAbort(ctx,500,err){return};ctx.JSON(200,items)})}
func (a *ServiceAccountAPI) CreateToken(ctx *gin.Context){withID(ctx,"id",func(id uint){
	account,err:=a.DB.GetServiceAccountByID(id);if !successOrAbort(ctx,500,err){return};if account==nil{ctx.AbortWithStatus(404);return}
	var p serviceTokenParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return}
	token,hash,prefix,err:=auth.GenerateServiceAccountToken();if !successOrAbort(ctx,500,err){return}
	var expires *time.Time;if p.ExpiresInDays>0{if p.ExpiresInDays>3650{ctx.AbortWithError(400,errors.New("token expiration cannot exceed 10 years"));return};value:=time.Now().AddDate(0,0,p.ExpiresInDays);expires=&value}
	item:=&model.ServiceAccountToken{AccountID:id,TokenHash:hash,TokenPrefix:prefix,Name:p.Name,ExpiresAt:expires}
	if !successOrAbort(ctx,500,a.DB.SaveServiceAccountToken(item)){return}
	ctx.JSON(201,model.ServiceAccountTokenResult{ID:item.ID,Name:item.Name,Token:token,Prefix:prefix,CreatedAt:item.CreatedAt,ExpiresAt:item.ExpiresAt})
})}
func (a *ServiceAccountAPI) DeleteToken(ctx *gin.Context){withID(ctx,"id",func(accountID uint){
	tokenID64,err:=strconv.ParseUint(ctx.Param("tokenId"),10,64);if err!=nil{ctx.AbortWithError(400,err);return}
	if !successOrAbort(ctx,500,a.DB.DeleteServiceAccountToken(accountID,uint(tokenID64))){return};ctx.Status(http.StatusNoContent)
})}

func (a *ServiceAccountAPI) GetChannels(ctx *gin.Context){
	account:=auth.GetServiceAccount(ctx);if account==nil{ctx.AbortWithStatus(401);return}
	items,err:=a.DB.GetApplications();if !successOrAbort(ctx,500,err){return}
	out:=make([]gin.H,0)
	for _,item:=range items{if auth.ServiceAllowsApplication(account,item.ID){out=append(out,gin.H{"id":item.ID,"name":item.Name,"description":item.Description})}}
	ctx.JSON(200,out)
}

type serviceMessageParams struct { Title string `json:"title"`; Message string `json:"message" binding:"required"`; Priority int `json:"priority"` }
func (a *ServiceAccountAPI) PublishMessage(ctx *gin.Context){
	appID64,err:=strconv.ParseUint(ctx.Param("id"),10,64);if err!=nil{ctx.AbortWithError(400,err);return};appID:=uint(appID64)
	account:=auth.GetServiceAccount(ctx);if !auth.ServiceAllowsApplication(account,appID){ctx.AbortWithError(403,errors.New("service account is not allowed to use this Channel"));return}
	var p serviceMessageParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return}
	msg,err:=a.Publisher.Publish(appID,p.Title,p.Message,p.Priority);if !successOrAbort(ctx,500,err){return}
	ctx.JSON(200,toExternalMessage(msg))
}
func (a *ServiceAccountAPI) GetMessages(ctx *gin.Context){
	appID64,err:=strconv.ParseUint(ctx.Param("id"),10,64);if err!=nil{ctx.AbortWithError(400,err);return};appID:=uint(appID64)
	account:=auth.GetServiceAccount(ctx);if !auth.ServiceAllowsApplication(account,appID){ctx.AbortWithError(403,errors.New("service account is not allowed to use this Channel"));return}
	limit:=50;if raw:=ctx.Query("limit");raw!=""{if parsed,parseErr:=strconv.Atoi(raw);parseErr==nil&&parsed>0&&parsed<=200{limit=parsed}}
	var since uint;if raw:=ctx.Query("since");raw!=""{if parsed,parseErr:=strconv.ParseUint(raw,10,64);parseErr==nil{since=uint(parsed)}}
	items,err:=a.DB.GetMessagesByApplicationSince(appID,limit,since);if !successOrAbort(ctx,500,err){return}
	out:=make([]*model.MessageExternal,0,len(items));for _,item:=range items{out=append(out,toExternalMessage(item))}
	ctx.JSON(200,out)
}
