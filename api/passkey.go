package api

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
)

type PasskeyDatabase interface {
	GetUserByID(id uint) (*model.User, error)
	GetUserByName(name string) (*model.User, error)
	GetPasskeysByUser(userID uint) ([]*model.PasskeyCredential, error)
	GetPasskeyByID(id uint) (*model.PasskeyCredential, error)
	GetPasskeyByCredentialID(credentialID string) (*model.PasskeyCredential, error)
	SavePasskey(item *model.PasskeyCredential) error
	DeletePasskey(userID, id uint) error
	SaveWebAuthnChallenge(item *model.WebAuthnChallenge) error
	GetWebAuthnChallenge(challenge string, now time.Time) (*model.WebAuthnChallenge, error)
	DeleteWebAuthnChallenge(challenge string) error
	CleanupWebAuthnChallenges(now time.Time) error
	GetSecurityPolicy() (model.SecurityPolicy, error)
	CreateClient(client *model.Client) error
	UpdateClientElevatedUntil(id uint, elevatedUntil *time.Time) error
	CreateAuditEvent(event *model.AuditEvent) error
}

type PasskeyAPI struct {
	DB            PasskeyDatabase
	SecureCookie  bool
	NotifyDeleted func(uint, string)
}

func webAuthnContext(ctx *gin.Context) (origin, rpID string, err error) {
	origin = strings.TrimSpace(ctx.GetHeader("Origin"))
	if origin == "" {
		scheme := "http"
		if ctx.Request.TLS != nil { scheme = "https" }
		if forwarded := strings.TrimSpace(ctx.GetHeader("X-Forwarded-Proto")); forwarded != "" {
			scheme = strings.Split(forwarded,",")[0]
		}
		origin = scheme + "://" + ctx.Request.Host
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname()=="" { return "","",errors.New("could not determine passkey relying-party origin") }
	return origin, parsed.Hostname(), nil
}

func decodeBase64URL(value string) ([]byte,error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(strings.TrimSpace(value),"="))
}
func encodeBase64URL(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }

func challengeFromClientData(raw []byte)(string,error){
	var payload struct{Challenge string `json:"challenge"`}
	if err:=json.Unmarshal(raw,&payload);err!=nil{return "",err}
	if payload.Challenge==""{return "",errors.New("passkey challenge is missing")}
	return payload.Challenge,nil
}

func userHandle(id uint) string {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:],uint64(id))
	return encodeBase64URL(raw[:])
}

type passkeyOptionsParams struct {
	Username string `json:"username"`
	ClientName string `json:"clientName"`
}

func (a *PasskeyAPI) LoginOptions(ctx *gin.Context) {
	var params passkeyOptionsParams
	if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
	user,err:=a.DB.GetUserByName(strings.TrimSpace(params.Username))
	if !successOrAbort(ctx,500,err){return}
	if user==nil{ctx.AbortWithError(404,errors.New("account not found"));return}
	credentials,err:=a.DB.GetPasskeysByUser(user.ID)
	if !successOrAbort(ctx,500,err){return}
	if len(credentials)==0{ctx.AbortWithError(400,errors.New("no passkeys are registered for this account"));return}
	origin,rpID,err:=webAuthnContext(ctx);if !successOrAbort(ctx,400,err){return}
	challenge,err:=security.NewWebAuthnChallenge();if !successOrAbort(ctx,500,err){return}
	item:=&model.WebAuthnChallenge{Challenge:challenge,UserID:user.ID,Purpose:"login",RPID:rpID,Origin:origin,ClientName:params.ClientName,ExpiresAt:time.Now().Add(5*time.Minute)}
	if !successOrAbort(ctx,500,a.DB.SaveWebAuthnChallenge(item)){return}
	allow:=make([]gin.H,0,len(credentials))
	for _,credential:=range credentials{allow=append(allow,gin.H{"type":"public-key","id":credential.CredentialID})}
	ctx.JSON(200,gin.H{"challenge":challenge,"rpId":rpID,"timeout":60000,"userVerification":"preferred","allowCredentials":allow})
}

type passkeyAssertionParams struct {
	CredentialID string `json:"credentialId" binding:"required"`
	ClientDataJSON string `json:"clientDataJSON" binding:"required"`
	AuthenticatorData string `json:"authenticatorData" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

func (a *PasskeyAPI) verifyAssertion(ctx *gin.Context, params passkeyAssertionParams, purpose string) (*model.User,*model.PasskeyCredential,*model.WebAuthnChallenge,uint32,error) {
	clientData,err:=decodeBase64URL(params.ClientDataJSON);if err!=nil{return nil,nil,nil,0,err}
	challengeValue,err:=challengeFromClientData(clientData);if err!=nil{return nil,nil,nil,0,err}
	challenge,err:=a.DB.GetWebAuthnChallenge(challengeValue,time.Now());if err!=nil{return nil,nil,nil,0,err}
	if challenge==nil||challenge.Purpose!=purpose{return nil,nil,nil,0,errors.New("passkey challenge expired or invalid")}
	if err:=security.VerifyWebAuthnClientData(clientData,"webauthn.get",challenge.Challenge,challenge.Origin);err!=nil{return nil,nil,nil,0,err}
	credential,err:=a.DB.GetPasskeyByCredentialID(params.CredentialID);if err!=nil{return nil,nil,nil,0,err}
	if credential==nil||credential.UserID!=challenge.UserID{return nil,nil,nil,0,errors.New("passkey credential does not match challenge")}
	authData,err:=decodeBase64URL(params.AuthenticatorData);if err!=nil{return nil,nil,nil,0,err}
	signature,err:=decodeBase64URL(params.Signature);if err!=nil{return nil,nil,nil,0,err}
	signCount,err:=security.VerifyWebAuthnAssertion(authData,clientData,signature,credential.PublicKeyX,credential.PublicKeyY,challenge.RPID,credential.SignCount)
	if err!=nil{return nil,nil,nil,0,err}
	user,err:=a.DB.GetUserByID(credential.UserID);if err!=nil{return nil,nil,nil,0,err};if user==nil{return nil,nil,nil,0,errors.New("passkey user no longer exists")}
	return user,credential,challenge,signCount,nil
}

func (a *PasskeyAPI) LoginVerify(ctx *gin.Context) {
	var params passkeyAssertionParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
	user,credential,challenge,signCount,err:=a.verifyAssertion(ctx,params,"login")
	if err!=nil{ctx.AbortWithError(http.StatusUnauthorized,err);return}
	credential.SignCount=signCount;now:=time.Now();credential.LastUsedAt=&now
	if !successOrAbort(ctx,500,a.DB.SavePasskey(credential)){return}
	_ = a.DB.DeleteWebAuthnChallenge(challenge.Challenge)
	policy,err:=a.DB.GetSecurityPolicy();if !successOrAbort(ctx,500,err){return}
	sessionMinutes:=policy.SessionInactivityMinutes;if sessionMinutes<=0{sessionMinutes=auth.CookieMaxAge/60}
	elevationMinutes:=policy.ElevationMinutes;if elevationMinutes<=0{elevationMinutes=60}
	elevatedUntil:=time.Now().Add(time.Duration(elevationMinutes)*time.Minute)
	publicToken,privateToken:=generateClientToken()
	client:=&model.Client{Name:challenge.ClientName,Token:publicToken,UserID:user.ID,ElevatedUntil:&elevatedUntil,ExpiresAfterInactivitySeconds:uint(sessionMinutes*60),MFAAuthenticated:true}
	if !successOrAbort(ctx,500,a.DB.CreateClient(client)){return}
	auth.SetCookie(ctx.Writer,privateToken,sessionMinutes*60,a.SecureCookie)
	_ = a.DB.CreateAuditEvent(&model.AuditEvent{UserID:user.ID,Username:user.Name,Action:"login_success",Target:"passkey",IPAddress:ctx.ClientIP()})
	ctx.JSON(200,&model.CurrentUserExternal{ID:user.ID,Name:user.Name,DisplayName:user.DisplayName,Admin:user.Admin,CreatedAt:user.CreatedAt,ClientID:client.ID,ElevatedUntil:client.ElevatedUntil,AuthProvider:"passkey"})
}

func (a *PasskeyAPI) RegistrationOptions(ctx *gin.Context) {
	userID:=auth.GetUserID(ctx);user,err:=a.DB.GetUserByID(userID);if !successOrAbort(ctx,500,err){return};if user==nil{ctx.AbortWithStatus(404);return}
	existing,err:=a.DB.GetPasskeysByUser(userID);if !successOrAbort(ctx,500,err){return}
	origin,rpID,err:=webAuthnContext(ctx);if !successOrAbort(ctx,400,err){return}
	challenge,err:=security.NewWebAuthnChallenge();if !successOrAbort(ctx,500,err){return}
	if !successOrAbort(ctx,500,a.DB.SaveWebAuthnChallenge(&model.WebAuthnChallenge{Challenge:challenge,UserID:userID,Purpose:"register",RPID:rpID,Origin:origin,ExpiresAt:time.Now().Add(5*time.Minute)})){return}
	exclude:=make([]gin.H,0,len(existing));for _,credential:=range existing{exclude=append(exclude,gin.H{"type":"public-key","id":credential.CredentialID})}
	display:=user.DisplayName;if display==""{display=user.Name}
	ctx.JSON(200,gin.H{
		"challenge":challenge,"rp":gin.H{"name":"Gotify MU","id":rpID},
		"user":gin.H{"id":userHandle(user.ID),"name":user.Name,"displayName":display},
		"pubKeyCredParams":[]gin.H{{"type":"public-key","alg":-7}},
		"timeout":60000,"attestation":"none","authenticatorSelection":gin.H{"residentKey":"preferred","userVerification":"preferred"},
		"excludeCredentials":exclude,
	})
}

type passkeyRegistrationParams struct {
	Name string `json:"name"`
	CredentialID string `json:"credentialId" binding:"required"`
	ClientDataJSON string `json:"clientDataJSON" binding:"required"`
	AttestationObject string `json:"attestationObject" binding:"required"`
}

func (a *PasskeyAPI) RegistrationVerify(ctx *gin.Context) {
	var params passkeyRegistrationParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
	clientData,err:=decodeBase64URL(params.ClientDataJSON);if err!=nil{ctx.AbortWithError(400,err);return}
	challengeValue,err:=challengeFromClientData(clientData);if err!=nil{ctx.AbortWithError(400,err);return}
	challenge,err:=a.DB.GetWebAuthnChallenge(challengeValue,time.Now());if !successOrAbort(ctx,500,err){return}
	if challenge==nil||challenge.Purpose!="register"||challenge.UserID!=auth.GetUserID(ctx){ctx.AbortWithError(400,errors.New("passkey challenge expired or invalid"));return}
	if err:=security.VerifyWebAuthnClientData(clientData,"webauthn.create",challenge.Challenge,challenge.Origin);err!=nil{ctx.AbortWithError(400,err);return}
	attestation,err:=decodeBase64URL(params.AttestationObject);if err!=nil{ctx.AbortWithError(400,err);return}
	registration,err:=security.ParseWebAuthnAttestation(attestation);if err!=nil{ctx.AbortWithError(400,err);return}
	if err:=security.VerifyWebAuthnRPID(registration.RPIDHash,challenge.RPID);err!=nil{ctx.AbortWithError(400,err);return}
	if encodeBase64URL(registration.CredentialID)!=params.CredentialID{ctx.AbortWithError(400,errors.New("credential id does not match attestation"));return}
	if existing,findErr:=a.DB.GetPasskeyByCredentialID(params.CredentialID);findErr!=nil{ctx.AbortWithError(500,findErr);return}else if existing!=nil{ctx.AbortWithError(409,errors.New("passkey is already registered"));return}
	name:=strings.TrimSpace(params.Name);if name==""{name="Passkey"}
	item:=&model.PasskeyCredential{UserID:challenge.UserID,Name:name,CredentialID:params.CredentialID,PublicKeyX:registration.PublicKeyX,PublicKeyY:registration.PublicKeyY,SignCount:registration.SignCount}
	if !successOrAbort(ctx,500,a.DB.SavePasskey(item)){return}
	_ = a.DB.DeleteWebAuthnChallenge(challenge.Challenge)
	ctx.JSON(201,model.PasskeyView{ID:item.ID,Name:item.Name,CreatedAt:item.CreatedAt})
}

func (a *PasskeyAPI) List(ctx *gin.Context) {
	items,err:=a.DB.GetPasskeysByUser(auth.GetUserID(ctx));if !successOrAbort(ctx,500,err){return}
	out:=make([]model.PasskeyView,0,len(items));for _,item:=range items{out=append(out,model.PasskeyView{ID:item.ID,Name:item.Name,CreatedAt:item.CreatedAt,LastUsedAt:item.LastUsedAt})}
	ctx.JSON(200,out)
}

func (a *PasskeyAPI) Delete(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){if !successOrAbort(ctx,500,a.DB.DeletePasskey(auth.GetUserID(ctx),id)){return};ctx.Status(http.StatusNoContent)})
}

func (a *PasskeyAPI) ElevationOptions(ctx *gin.Context) {
	userID:=auth.GetUserID(ctx);credentials,err:=a.DB.GetPasskeysByUser(userID);if !successOrAbort(ctx,500,err){return}
	if len(credentials)==0{ctx.AbortWithError(400,errors.New("no passkeys registered"));return}
	origin,rpID,err:=webAuthnContext(ctx);if !successOrAbort(ctx,400,err){return}
	challenge,err:=security.NewWebAuthnChallenge();if !successOrAbort(ctx,500,err){return}
	client:=auth.GetClient(ctx);if client==nil{ctx.AbortWithError(403,errors.New("browser session required"));return}
	if !successOrAbort(ctx,500,a.DB.SaveWebAuthnChallenge(&model.WebAuthnChallenge{Challenge:challenge,UserID:userID,Purpose:"elevate",RPID:rpID,Origin:origin,ClientID:client.ID,ExpiresAt:time.Now().Add(5*time.Minute)})){return}
	allow:=make([]gin.H,0,len(credentials));for _,credential:=range credentials{allow=append(allow,gin.H{"type":"public-key","id":credential.CredentialID})}
	ctx.JSON(200,gin.H{"challenge":challenge,"rpId":rpID,"timeout":60000,"userVerification":"preferred","allowCredentials":allow})
}

func (a *PasskeyAPI) ElevationVerify(ctx *gin.Context) {
	var params passkeyAssertionParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
	user,credential,challenge,signCount,err:=a.verifyAssertion(ctx,params,"elevate")
	if err!=nil{ctx.AbortWithError(401,err);return}
	client:=auth.GetClient(ctx);if client==nil||client.ID!=challenge.ClientID||client.UserID!=user.ID{ctx.AbortWithError(403,errors.New("passkey challenge does not match current session"));return}
	policy,err:=a.DB.GetSecurityPolicy();if !successOrAbort(ctx,500,err){return};minutes:=policy.ElevationMinutes;if minutes<=0{minutes=60}
	until:=time.Now().Add(time.Duration(minutes)*time.Minute)
	if !successOrAbort(ctx,500,a.DB.UpdateClientElevatedUntil(client.ID,&until)){return}
	credential.SignCount=signCount;now:=time.Now();credential.LastUsedAt=&now;_ = a.DB.SavePasskey(credential);_ = a.DB.DeleteWebAuthnChallenge(challenge.Challenge)
	_ = a.DB.CreateAuditEvent(&model.AuditEvent{UserID:user.ID,Username:user.Name,Action:"session_elevated",Target:"passkey",IPAddress:ctx.ClientIP()})
	ctx.Status(http.StatusNoContent)
}
