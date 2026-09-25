package api

import (
	"errors"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
)

type rssParams struct {
	Name string `json:"name" binding:"required"`
	ApplicationID uint `json:"applicationId" binding:"required"`
	URL string `json:"url" binding:"required"`
	PollMinutes int `json:"pollMinutes"`
	TitlePrefix string `json:"titlePrefix"`
	Enabled bool `json:"enabled"`
}

func (a *AutomationAPI) GetRSS(ctx *gin.Context) {
	items, err := a.DB.GetRSSIntegrations()
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, items)
}

func (a *AutomationAPI) CreateRSS(ctx *gin.Context) {
	var params rssParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if !a.channelExists(ctx, params.ApplicationID) { return }
	if !validHTTPURL(params.URL) { ctx.AbortWithError(400, errors.New("feed URL must use http or https")); return }
	if params.PollMinutes < 1 { params.PollMinutes = 5 }
	item := &model.RSSIntegration{
		Name:params.Name, ApplicationID:params.ApplicationID, URL:params.URL,
		PollMinutes:params.PollMinutes, TitlePrefix:params.TitlePrefix, Enabled:params.Enabled,
	}
	if !successOrAbort(ctx, 500, a.DB.SaveRSSIntegration(item)) { return }
	a.Engine.ReloadIntegrations()
	ctx.JSON(201, item)
}

func (a *AutomationAPI) UpdateRSS(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetRSSIntegrationByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("RSS Monitor not found")); return }
		var params rssParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if !a.channelExists(ctx, params.ApplicationID) { return }
		if !validHTTPURL(params.URL) { ctx.AbortWithError(400, errors.New("feed URL must use http or https")); return }
		if params.PollMinutes < 1 { params.PollMinutes = 5 }
		item.Name, item.ApplicationID, item.URL = params.Name, params.ApplicationID, params.URL
		item.PollMinutes, item.TitlePrefix, item.Enabled = params.PollMinutes, params.TitlePrefix, params.Enabled
		if !successOrAbort(ctx, 500, a.DB.SaveRSSIntegration(item)) { return }
		a.Engine.ReloadIntegrations()
		ctx.JSON(200, item)
	})
}

func (a *AutomationAPI) DeleteRSS(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if successOrAbort(ctx, 500, a.DB.DeleteRSSIntegration(id)) { a.Engine.ReloadIntegrations() }
	})
}

type calendarParams struct {
	Name string `json:"name" binding:"required"`
	ApplicationID uint `json:"applicationId" binding:"required"`
	URL string `json:"url" binding:"required"`
	PollMinutes int `json:"pollMinutes"`
	AdvanceMinutes int `json:"advanceMinutes"`
	Enabled bool `json:"enabled"`
}

func (a *AutomationAPI) GetCalendars(ctx *gin.Context) {
	items, err := a.DB.GetCalendarIntegrations()
	if !successOrAbort(ctx, 500, err) { return }
	ctx.JSON(200, items)
}
func (a *AutomationAPI) CreateCalendar(ctx *gin.Context) {
	var params calendarParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if !a.channelExists(ctx, params.ApplicationID) { return }
	if !validHTTPURL(params.URL) { ctx.AbortWithError(400, errors.New("calendar URL must use http or https")); return }
	if params.PollMinutes < 1 { params.PollMinutes = 5 }
	if params.AdvanceMinutes < 0 { params.AdvanceMinutes = 0 }
	item := &model.CalendarIntegration{
		Name:params.Name, ApplicationID:params.ApplicationID, URL:params.URL,
		PollMinutes:params.PollMinutes, AdvanceMinutes:params.AdvanceMinutes, Enabled:params.Enabled,
	}
	if !successOrAbort(ctx, 500, a.DB.SaveCalendarIntegration(item)) { return }
	a.Engine.ReloadIntegrations()
	ctx.JSON(201, item)
}
func (a *AutomationAPI) UpdateCalendar(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		item, err := a.DB.GetCalendarIntegrationByID(id)
		if !successOrAbort(ctx, 500, err) { return }
		if item == nil { ctx.AbortWithError(404, errors.New("Calendar connection not found")); return }
		var params calendarParams
		if err := ctx.ShouldBindJSON(&params); err != nil { return }
		if !a.channelExists(ctx, params.ApplicationID) { return }
		if !validHTTPURL(params.URL) { ctx.AbortWithError(400, errors.New("calendar URL must use http or https")); return }
		if params.PollMinutes < 1 { params.PollMinutes = 5 }
		if params.AdvanceMinutes < 0 { params.AdvanceMinutes = 0 }
		item.Name, item.ApplicationID, item.URL = params.Name, params.ApplicationID, params.URL
		item.PollMinutes, item.AdvanceMinutes, item.Enabled = params.PollMinutes, params.AdvanceMinutes, params.Enabled
		if !successOrAbort(ctx, 500, a.DB.SaveCalendarIntegration(item)) { return }
		a.Engine.ReloadIntegrations()
		ctx.JSON(200, item)
	})
}
func (a *AutomationAPI) DeleteCalendar(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if successOrAbort(ctx, 500, a.DB.DeleteCalendarIntegration(id)) { a.Engine.ReloadIntegrations() }
	})
}

type emailGatewayParams struct {
	Name string `json:"name" binding:"required"`
	ApplicationID uint `json:"applicationId" binding:"required"`
	Host string `json:"host" binding:"required"`
	Port int `json:"port"`
	UseTLS bool `json:"useTls"`
	StartTLS bool `json:"startTls"`
	Username string `json:"username"`
	Password string `json:"password"`
	FromAddress string `json:"fromAddress" binding:"required"`
	ToAddresses string `json:"toAddresses" binding:"required"`
	MinPriority int `json:"minPriority"`
	Enabled bool `json:"enabled"`
}

func emailGatewayView(item *model.EmailGateway) model.EmailGatewayView {
	return model.EmailGatewayView{
		ID:item.ID, Name:item.Name, ApplicationID:item.ApplicationID, Host:item.Host, Port:item.Port,
		UseTLS:item.UseTLS, StartTLS:item.StartTLS, Username:item.Username, PasswordConfigured:item.Password!="",
		FromAddress:item.FromAddress, ToAddresses:item.ToAddresses, MinPriority:item.MinPriority, Enabled:item.Enabled,
		CreatedAt:item.CreatedAt, UpdatedAt:item.UpdatedAt,
	}
}

func validateEmailGateway(params emailGatewayParams) error {
	if strings.TrimSpace(params.Host) == "" { return errors.New("SMTP host is required") }
	if params.Port < 0 || params.Port > 65535 { return errors.New("SMTP port is invalid") }
	if _, err := mail.ParseAddress(params.FromAddress); err != nil { return errors.New("From address is invalid") }
	validTo := 0
	for _, part := range strings.FieldsFunc(params.ToAddresses, func(r rune) bool { return r==',' || r==';' || r=='\n' }) {
		if _, err := mail.ParseAddress(strings.TrimSpace(part)); err == nil { validTo++ }
	}
	if validTo == 0 { return errors.New("at least one valid recipient is required") }
	if params.UseTLS && params.StartTLS { return errors.New("choose implicit TLS or STARTTLS, not both") }
	return nil
}

func (a *AutomationAPI) GetEmailGateways(ctx *gin.Context) {
	items, err := a.DB.GetEmailGateways()
	if !successOrAbort(ctx, 500, err) { return }
	out := make([]model.EmailGatewayView,0,len(items))
	for _,item := range items { out=append(out,emailGatewayView(item)) }
	ctx.JSON(200,out)
}
func (a *AutomationAPI) CreateEmailGateway(ctx *gin.Context) {
	var params emailGatewayParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	if !a.channelExists(ctx, params.ApplicationID) { return }
	if err := validateEmailGateway(params); err != nil { ctx.AbortWithError(400,err); return }
	protected, err := security.Protect(params.Password)
	if !successOrAbort(ctx,500,err){return}
	item:=&model.EmailGateway{
		Name:params.Name,ApplicationID:params.ApplicationID,Host:params.Host,Port:params.Port,
		UseTLS:params.UseTLS,StartTLS:params.StartTLS,Username:params.Username,Password:protected,
		FromAddress:params.FromAddress,ToAddresses:params.ToAddresses,MinPriority:params.MinPriority,Enabled:params.Enabled,
	}
	if !successOrAbort(ctx,500,a.DB.SaveEmailGateway(item)){return}
	ctx.JSON(201,emailGatewayView(item))
}
func (a *AutomationAPI) UpdateEmailGateway(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){
		item,err:=a.DB.GetEmailGatewayByID(id);if !successOrAbort(ctx,500,err){return}
		if item==nil{ctx.AbortWithError(404,errors.New("Email Gateway not found"));return}
		var params emailGatewayParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
		if !a.channelExists(ctx,params.ApplicationID){return}
		if err:=validateEmailGateway(params);err!=nil{ctx.AbortWithError(400,err);return}
		item.Name,item.ApplicationID,item.Host,item.Port=params.Name,params.ApplicationID,params.Host,params.Port
		item.UseTLS,item.StartTLS,item.Username=params.UseTLS,params.StartTLS,params.Username
		item.FromAddress,item.ToAddresses,item.MinPriority,item.Enabled=params.FromAddress,params.ToAddresses,params.MinPriority,params.Enabled
		if params.Password!=""{
			protected,protectErr:=security.Protect(params.Password);if !successOrAbort(ctx,500,protectErr){return};item.Password=protected
		}
		if !successOrAbort(ctx,500,a.DB.SaveEmailGateway(item)){return}
		ctx.JSON(200,emailGatewayView(item))
	})
}
func (a *AutomationAPI) DeleteEmailGateway(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){successOrAbort(ctx,500,a.DB.DeleteEmailGateway(id))})
}

type smtpReceiverParams struct {
	ListenAddress string `json:"listenAddress" binding:"required"`
	Username string `json:"username"`
	Password string `json:"password"`
	AllowedCIDRs string `json:"allowedCidrs"`
	MaxMessageBytes int64 `json:"maxMessageBytes"`
	Enabled bool `json:"enabled"`
}
func smtpReceiverView(item *model.SMTPReceiver) model.SMTPReceiverView {
	return model.SMTPReceiverView{
		ID:item.ID,ListenAddress:item.ListenAddress,Username:item.Username,PasswordConfigured:item.Password!="",
		AllowedCIDRs:item.AllowedCIDRs,MaxMessageBytes:item.MaxMessageBytes,Enabled:item.Enabled,UpdatedAt:item.UpdatedAt,
	}
}
func (a *AutomationAPI) GetSMTPReceiver(ctx *gin.Context) {
	item,err:=a.DB.GetSMTPReceiver();if !successOrAbort(ctx,500,err){return};ctx.JSON(200,smtpReceiverView(item))
}
func (a *AutomationAPI) SaveSMTPReceiver(ctx *gin.Context) {
	var params smtpReceiverParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
	if err:=validateAllowedCIDRs(params.AllowedCIDRs);err!=nil{ctx.AbortWithError(400,err);return}
	if params.MaxMessageBytes<=0{params.MaxMessageBytes=10<<20}
	if params.MaxMessageBytes>50<<20{ctx.AbortWithError(400,errors.New("maximum SMTP message size may not exceed 50 MiB"));return}
	item,err:=a.DB.GetSMTPReceiver();if !successOrAbort(ctx,500,err){return}
	item.ListenAddress=params.ListenAddress;item.Username=params.Username;item.AllowedCIDRs=params.AllowedCIDRs
	item.MaxMessageBytes=params.MaxMessageBytes;item.Enabled=params.Enabled
	if params.Password!=""{protected,protectErr:=security.Protect(params.Password);if !successOrAbort(ctx,500,protectErr){return};item.Password=protected}
	if item.Enabled&&item.Username!=""&&item.Password==""{ctx.AbortWithError(400,errors.New("SMTP Receiver password is required when a username is configured"));return}
	if !successOrAbort(ctx,500,a.DB.SaveSMTPReceiver(item)){return};a.Engine.ReloadIntegrations();ctx.JSON(200,smtpReceiverView(item))
}
func (a *AutomationAPI) GetSMTPRoutes(ctx *gin.Context) {
	items,err:=a.DB.GetSMTPRoutes();if !successOrAbort(ctx,500,err){return};ctx.JSON(200,items)
}
type smtpRouteParams struct {
	Recipient string `json:"recipient" binding:"required"`
	ApplicationID uint `json:"applicationId" binding:"required"`
	Enabled bool `json:"enabled"`
}
func (a *AutomationAPI) CreateSMTPRoute(ctx *gin.Context) {
	var params smtpRouteParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
	if !a.channelExists(ctx,params.ApplicationID){return}
	address,err:=mail.ParseAddress(params.Recipient);if err!=nil{ctx.AbortWithError(400,errors.New("recipient address is invalid"));return}
	item:=&model.SMTPRoute{Recipient:strings.ToLower(address.Address),ApplicationID:params.ApplicationID,Enabled:params.Enabled}
	if !successOrAbort(ctx,500,a.DB.SaveSMTPRoute(item)){return};ctx.JSON(201,item)
}
func (a *AutomationAPI) UpdateSMTPRoute(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){
		var params smtpRouteParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
		if !a.channelExists(ctx,params.ApplicationID){return}
		address,err:=mail.ParseAddress(params.Recipient);if err!=nil{ctx.AbortWithError(400,errors.New("recipient address is invalid"));return}
		item:=&model.SMTPRoute{ID:id,Recipient:strings.ToLower(address.Address),ApplicationID:params.ApplicationID,Enabled:params.Enabled}
		if !successOrAbort(ctx,500,a.DB.SaveSMTPRoute(item)){return};ctx.JSON(200,item)
	})
}
func (a *AutomationAPI) DeleteSMTPRoute(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){successOrAbort(ctx,500,a.DB.DeleteSMTPRoute(id))})
}

type syslogParams struct {
	Name string `json:"name" binding:"required"`
	ApplicationID uint `json:"applicationId" binding:"required"`
	ListenAddress string `json:"listenAddress" binding:"required"`
	Protocol string `json:"protocol" binding:"required"`
	AllowedCIDRs string `json:"allowedCidrs"`
	MinSeverity int `json:"minSeverity"`
	Enabled bool `json:"enabled"`
}
func (a *AutomationAPI) GetSyslog(ctx *gin.Context) {
	items,err:=a.DB.GetSyslogReceivers();if !successOrAbort(ctx,500,err){return};ctx.JSON(200,items)
}
func validateSyslog(params syslogParams)error{
	if params.Protocol!="udp"&&params.Protocol!="tcp"{return errors.New("Syslog protocol must be udp or tcp")}
	if err:=validateAllowedCIDRs(params.AllowedCIDRs);err!=nil{return err}
	if params.MinSeverity<0||params.MinSeverity>7{return errors.New("Syslog minimum severity must be between 0 and 7")}
	return nil
}
func (a *AutomationAPI) CreateSyslog(ctx *gin.Context) {
	var params syslogParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
	if !a.channelExists(ctx,params.ApplicationID){return};params.Protocol=strings.ToLower(params.Protocol)
	if err:=validateSyslog(params);err!=nil{ctx.AbortWithError(400,err);return}
	item:=&model.SyslogReceiver{Name:params.Name,ApplicationID:params.ApplicationID,ListenAddress:params.ListenAddress,Protocol:params.Protocol,AllowedCIDRs:params.AllowedCIDRs,MinSeverity:params.MinSeverity,Enabled:params.Enabled}
	if !successOrAbort(ctx,500,a.DB.SaveSyslogReceiver(item)){return};a.Engine.ReloadIntegrations();ctx.JSON(201,item)
}
func (a *AutomationAPI) UpdateSyslog(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){
		item,err:=a.DB.GetSyslogReceiverByID(id);if !successOrAbort(ctx,500,err){return};if item==nil{ctx.AbortWithError(404,errors.New("Syslog Receiver not found"));return}
		var params syslogParams;if err:=ctx.ShouldBindJSON(&params);err!=nil{return};params.Protocol=strings.ToLower(params.Protocol)
		if !a.channelExists(ctx,params.ApplicationID){return};if err:=validateSyslog(params);err!=nil{ctx.AbortWithError(400,err);return}
		item.Name,item.ApplicationID,item.ListenAddress,item.Protocol=params.Name,params.ApplicationID,params.ListenAddress,params.Protocol
		item.AllowedCIDRs,item.MinSeverity,item.Enabled=params.AllowedCIDRs,params.MinSeverity,params.Enabled
		if !successOrAbort(ctx,500,a.DB.SaveSyslogReceiver(item)){return};a.Engine.ReloadIntegrations();ctx.JSON(200,item)
	})
}
func (a *AutomationAPI) DeleteSyslog(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){if successOrAbort(ctx,500,a.DB.DeleteSyslogReceiver(id)){a.Engine.ReloadIntegrations()}})
}

func validRemoteURL(raw string) bool {
	parsed,err:=url.Parse(raw);return err==nil&&(parsed.Scheme=="http"||parsed.Scheme=="https")&&parsed.Host!=""
}

func parsePort(raw string, fallback int) int {
	value,err:=strconv.Atoi(strings.TrimSpace(raw));if err!=nil||value<=0{return fallback};return value
}
