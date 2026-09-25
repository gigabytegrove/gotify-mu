package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/auth/ldap"
	"github.com/gotify/server/v3/auth/password"
	"github.com/gotify/server/v3/config"
	"github.com/gotify/server/v3/database"
	"github.com/gotify/server/v3/model"
)

type LDAPAPI struct {
	DB                 *database.GormDatabase
	UserChangeNotifier *UserChangeNotifier
	Config             config.LDAP
	SecureCookie       bool
	PasswordStrength   int
}

func NewLDAP(conf *config.Configuration, db *database.GormDatabase, notifier *UserChangeNotifier) *LDAPAPI {
	return &LDAPAPI{
		DB:db,
		UserChangeNotifier:notifier,
		Config:conf.LDAP,
		SecureCookie:conf.Server.SecureCookie,
		PasswordStrength:conf.PassStrength,
	}
}

func (a *LDAPAPI) ldapConfig() ldapauth.Config {
	return ldapauth.Config{
		URL:a.Config.URL,
		BindDN:a.Config.BindDN,
		BindPassword:a.Config.BindPassword,
		BaseDN:a.Config.BaseDN,
		UserFilter:a.Config.UserFilter,
		DisplayNameAttribute:a.Config.DisplayNameAttribute,
		GroupAttribute:a.Config.GroupAttribute,
		AdminGroupDN:a.Config.AdminGroupDN,
		UserGroupDN:a.Config.UserGroupDN,
		CAFile:a.Config.CAFile,
		InsecureSkipVerify:a.Config.InsecureSkipVerify,
		Timeout:10*time.Second,
	}
}

func randomUnusablePassword(strength int) ([]byte,error) {
	raw:=make([]byte,32)
	if _,err:=rand.Read(raw);err!=nil{return nil,err}
	return password.CreatePassword(hex.EncodeToString(raw),strength)
}

func (a *LDAPAPI) resolveUser(directoryUser *ldapauth.User) (*model.User,error) {
	ldapID:=strings.ToLower(strings.TrimSpace(directoryUser.DN))
	user,err:=a.DB.GetUserByLDAP(ldapID)
	if err!=nil{return nil,err}
	if user==nil && a.Config.LinkByUsername {
		candidate,findErr:=a.DB.GetUserByName(directoryUser.Username)
		if findErr!=nil{return nil,findErr}
		if candidate!=nil && candidate.LDAPID==nil && candidate.OIDCID==nil {
			candidate.LDAPID=&ldapID
			user=candidate
		}
	}
	if user==nil {
		if !a.Config.AutoRegister{return nil,errors.New("directory account is not registered in Gotify MU")}
		if existing,findErr:=a.DB.GetUserByName(directoryUser.Username);findErr!=nil{return nil,findErr}else if existing!=nil{
			return nil,errors.New("a Gotify MU user already uses this directory username")
		}
		pass,passErr:=randomUnusablePassword(a.PasswordStrength);if passErr!=nil{return nil,passErr}
		user=&model.User{
			Name:directoryUser.Username,
			DisplayName:directoryUser.DisplayName,
			Pass:pass,
			Admin:directoryUser.Admin,
			LDAPID:&ldapID,
		}
		if err:=a.DB.CreateUser(user);err!=nil{return nil,err}
		if a.UserChangeNotifier!=nil {
			if err:=a.UserChangeNotifier.fireUserAdded(user.ID);err!=nil{return nil,err}
		}
		return user,nil
	}
	changed:=false
	if user.DisplayName!=directoryUser.DisplayName{user.DisplayName=directoryUser.DisplayName;changed=true}
	if user.Admin!=directoryUser.Admin{user.Admin=directoryUser.Admin;changed=true}
	if user.LDAPID==nil||*user.LDAPID!=ldapID{user.LDAPID=&ldapID;changed=true}
	if changed {
		if err:=a.DB.UpdateUser(user);err!=nil{return nil,err}
	}
	return user,nil
}

func (a *LDAPAPI) Login(ctx *gin.Context) {
	if !a.Config.Enabled {
		ctx.AbortWithError(http.StatusNotFound,errors.New("directory authentication is disabled"))
		return
	}
	username,passwordValue,ok:=ctx.Request.BasicAuth()
	if !ok || strings.TrimSpace(username)=="" || passwordValue=="" {
		ctx.AbortWithError(http.StatusUnauthorized,errors.New("directory username and password required"))
		return
	}

	directoryUser,err:=ldapauth.Authenticate(a.ldapConfig(),username,passwordValue)
	if err!=nil {
		_ = a.DB.CreateAuditEvent(&model.AuditEvent{
			Username:username,Action:"login_failed",Target:"directory_auth",IPAddress:ctx.ClientIP(),
		})
		ctx.AbortWithError(http.StatusUnauthorized,errors.New("directory sign-in failed"))
		return
	}
	user,err:=a.resolveUser(directoryUser)
	if !successOrAbort(ctx,http.StatusInternalServerError,err){return}

	var clientParams ClientParams
	if err:=ctx.Bind(&clientParams);err!=nil{return}
	policy,err:=a.DB.GetSecurityPolicy()
	if !successOrAbort(ctx,500,err){return}
	elevationMinutes:=policy.ElevationMinutes;if elevationMinutes<=0{elevationMinutes=60}
	sessionMinutes:=policy.SessionInactivityMinutes;if sessionMinutes<=0{sessionMinutes=auth.CookieMaxAge/60}
	elevatedUntil:=time.Now().Add(time.Duration(elevationMinutes)*time.Minute)
	tokenPublic,tokenPrivate:=generateClientToken()
	client:=&model.Client{
		Name:clientParams.Name,Token:tokenPublic,UserID:user.ID,ElevatedUntil:&elevatedUntil,
		ExpiresAfterInactivitySeconds:uint(sessionMinutes*60),MFAAuthenticated:true,
	}
	if !successOrAbort(ctx,500,a.DB.CreateClient(client)){return}
	auth.SetCookie(ctx.Writer,tokenPrivate,sessionMinutes*60,a.SecureCookie)
	_ = a.DB.CreateAuditEvent(&model.AuditEvent{
		UserID:user.ID,Username:user.Name,Action:"login_success",Target:"directory_auth",IPAddress:ctx.ClientIP(),
	})
	ctx.JSON(200,&model.CurrentUserExternal{
		ID:user.ID,Name:user.Name,DisplayName:user.DisplayName,Admin:user.Admin,CreatedAt:user.CreatedAt,
		ClientID:client.ID,ElevatedUntil:client.ElevatedUntil,
	})
}
