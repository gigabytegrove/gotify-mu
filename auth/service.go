package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
)

const serviceAccountContextKey = "gotify-service-account"

type ServiceAccountDatabase interface {
	GetServiceAccountByTokenHash(hash string, now time.Time) (*model.ServiceAccount, *model.ServiceAccountToken, error)
	TouchServiceAccountToken(accountID, tokenID uint, now time.Time) error
}

func GenerateServiceAccountToken() (token, hash, prefix string, err error) {
	raw:=make([]byte,32)
	if _,err=rand.Read(raw);err!=nil{return}
	token="SA."+base64.RawURLEncoding.EncodeToString(raw)
	sum:=sha256.Sum256([]byte(token))
	hash=hex.EncodeToString(sum[:])
	prefix=token
	if len(prefix)>14{prefix=prefix[:14]}
	return
}

func HashServiceAccountToken(token string) string {
	sum:=sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func serviceTokenFromRequest(ctx *gin.Context) string {
	if raw:=strings.TrimSpace(ctx.GetHeader("Authorization"));len(raw)>7&&strings.EqualFold(raw[:7],"Bearer "){
		token:=strings.TrimSpace(raw[7:]);if strings.HasPrefix(token,"SA."){return token}
	}
	if token:=strings.TrimSpace(ctx.GetHeader("X-Gotify-Key"));strings.HasPrefix(token,"SA."){return token}
	if token:=strings.TrimSpace(ctx.Query("token"));strings.HasPrefix(token,"SA."){return token}
	return ""
}

func ServiceScopes(account *model.ServiceAccount) map[string]bool {
	result:=map[string]bool{}
	if account==nil{return result}
	for _,scope:=range strings.FieldsFunc(account.Scopes,func(r rune)bool{return r==','||r==';'||r==' '||r=='\n'||r=='\t'}){
		scope=strings.TrimSpace(scope);if scope!=""{result[scope]=true}
	}
	return result
}
func ServiceHasScope(account *model.ServiceAccount, scope string) bool {
	scopes:=ServiceScopes(account)
	return scopes["*"]||scopes[scope]
}
func ServiceAllowsApplication(account *model.ServiceAccount, applicationID uint) bool {
	if account==nil{return false}
	raw:=strings.TrimSpace(account.AllowedApplicationIDs);if raw==""||raw=="*"{return true}
	target:=strconv.FormatUint(uint64(applicationID),10)
	for _,part:=range strings.FieldsFunc(raw,func(r rune)bool{return r==','||r==';'||r==' '}){
		if strings.TrimSpace(part)==target{return true}
	}
	return false
}

func RequireServiceScope(db ServiceAccountDatabase, scope string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token:=serviceTokenFromRequest(ctx)
		if token==""{ctx.AbortWithError(http.StatusUnauthorized,errors.New("service token required"));return}
		now:=time.Now()
		account,tokenRecord,err:=db.GetServiceAccountByTokenHash(HashServiceAccountToken(token),now)
		if err!=nil{ctx.AbortWithError(http.StatusInternalServerError,err);return}
		if account==nil||tokenRecord==nil{ctx.AbortWithError(http.StatusUnauthorized,errors.New("invalid or expired service token"));return}
		if !ServiceHasScope(account,scope){ctx.AbortWithError(http.StatusForbidden,errors.New("service account scope does not allow this action"));return}
		ctx.Set(serviceAccountContextKey,account)
		_ = db.TouchServiceAccountToken(account.ID,tokenRecord.ID,now)
		ctx.Next()
	}
}

func GetServiceAccount(ctx *gin.Context) *model.ServiceAccount {
	value,ok:=ctx.Get(serviceAccountContextKey);if !ok{return nil}
	account,_:=value.(*model.ServiceAccount);return account
}
