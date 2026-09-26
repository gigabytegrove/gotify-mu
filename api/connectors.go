package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/connectors"
	"github.com/gotify/server/v3/model"
)

type ConnectorDatabase interface {
	GetApplicationByID(id uint) (*model.Application, error)

	GetEmailGateways() ([]*model.EmailGateway, error)
	GetEmailGatewayByID(id uint) (*model.EmailGateway, error)
	SaveEmailGateway(item *model.EmailGateway) error
	DeleteEmailGateway(id uint) error

	GetSMTPRoutes() ([]*model.SMTPRoute, error)
	GetSMTPRouteByID(id uint) (*model.SMTPRoute, error)
	SaveSMTPRoute(item *model.SMTPRoute) error
	DeleteSMTPRoute(id uint) error

	GetRSSMonitors() ([]*model.RSSMonitor, error)
	GetRSSMonitorByID(id uint) (*model.RSSMonitor, error)
	SaveRSSMonitor(item *model.RSSMonitor) error
	DeleteRSSMonitor(id uint) error

	GetSyslogRoutes() ([]*model.SyslogRoute, error)
	GetSyslogRouteByID(id uint) (*model.SyslogRoute, error)
	SaveSyslogRoute(item *model.SyslogRoute) error
	DeleteSyslogRoute(id uint) error

	GetCalendarMonitors() ([]*model.CalendarMonitor, error)
	GetCalendarMonitorByID(id uint) (*model.CalendarMonitor, error)
	SaveCalendarMonitor(item *model.CalendarMonitor) error
	DeleteCalendarMonitor(id uint) error
}

type ConnectorRuntime interface {
	TestEmailGateway(id uint) error
}

type ConnectorAPI struct {
	DB      ConnectorDatabase
	Runtime ConnectorRuntime
}

func (a *ConnectorAPI) channel(ctx *gin.Context, id uint) bool {
	item, err := a.DB.GetApplicationByID(id)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return false }
	if item == nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.New("channel not found"))
		return false
	}
	return true
}

type emailGatewayParams struct {
	Name string `json:"name" binding:"required"`
	SourceApplicationID uint `json:"sourceApplicationId" binding:"required"`
	SMTPHost string `json:"smtpHost" binding:"required"`
	SMTPPort int `json:"smtpPort"`
	TLSMode string `json:"tlsMode"`
	Username string `json:"username"`
	Password string `json:"password"`
	FromAddress string `json:"fromAddress" binding:"required"`
	ToAddresses string `json:"toAddresses" binding:"required"`
	MinPriority int `json:"minPriority"`
	Enabled bool `json:"enabled"`
}

func emailGatewayView(item *model.EmailGateway) model.EmailGatewayView {
	copy := *item
	copy.Password = ""
	return model.EmailGatewayView{EmailGateway:copy,PasswordConfigured:item.Password!=""}
}
func (a *ConnectorAPI) GetEmailGateways(ctx *gin.Context) {
	items,err:=a.DB.GetEmailGateways();if !successOrAbort(ctx,500,err){return}
	out:=make([]model.EmailGatewayView,0,len(items));for _,item:=range items{out=append(out,emailGatewayView(item))}
	ctx.JSON(200,out)
}
func (a *ConnectorAPI) CreateEmailGateway(ctx *gin.Context) {
	var p emailGatewayParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return}
	if !a.channel(ctx,p.SourceApplicationID){return}
	mode:=strings.ToLower(strings.TrimSpace(p.TLSMode));if mode==""{mode="starttls"};if mode!="none"&&mode!="starttls"&&mode!="tls"{ctx.AbortWithError(400,errors.New("TLS mode must be none, starttls, or tls"));return}
	item:=&model.EmailGateway{Name:p.Name,SourceApplicationID:p.SourceApplicationID,SMTPHost:p.SMTPHost,SMTPPort:p.SMTPPort,TLSMode:mode,Username:p.Username,Password:p.Password,FromAddress:p.FromAddress,ToAddresses:p.ToAddresses,MinPriority:p.MinPriority,Enabled:p.Enabled}
	if !successOrAbort(ctx,500,a.DB.SaveEmailGateway(item)){return};ctx.JSON(201,emailGatewayView(item))
}
func (a *ConnectorAPI) UpdateEmailGateway(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){item,err:=a.DB.GetEmailGatewayByID(id);if !successOrAbort(ctx,500,err){return};if item==nil{ctx.AbortWithStatus(404);return}
		var p emailGatewayParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.SourceApplicationID){return}
		mode:=strings.ToLower(strings.TrimSpace(p.TLSMode));if mode==""{mode="starttls"};if mode!="none"&&mode!="starttls"&&mode!="tls"{ctx.AbortWithError(400,errors.New("TLS mode must be none, starttls, or tls"));return}
		item.Name=p.Name;item.SourceApplicationID=p.SourceApplicationID;item.SMTPHost=p.SMTPHost;item.SMTPPort=p.SMTPPort;item.TLSMode=mode;item.Username=p.Username;if p.Password!=""{item.Password=p.Password};item.FromAddress=p.FromAddress;item.ToAddresses=p.ToAddresses;item.MinPriority=p.MinPriority;item.Enabled=p.Enabled
		if !successOrAbort(ctx,500,a.DB.SaveEmailGateway(item)){return};ctx.JSON(200,emailGatewayView(item))
	})
}
func (a *ConnectorAPI) DeleteEmailGateway(ctx *gin.Context){withID(ctx,"id",func(id uint){successOrAbort(ctx,500,a.DB.DeleteEmailGateway(id))})}
func (a *ConnectorAPI) TestEmailGateway(ctx *gin.Context){withID(ctx,"id",func(id uint){if !successOrAbort(ctx,502,a.Runtime.TestEmailGateway(id)){return};ctx.JSON(200,gin.H{"sent":true})})}

type smtpRouteParams struct {
	Name string `json:"name" binding:"required"`
	ApplicationID uint `json:"applicationId" binding:"required"`
	Recipient string `json:"recipient" binding:"required"`
	AllowedCIDRs string `json:"allowedCidrs"`
	SenderContains string `json:"senderContains"`
	SubjectContains string `json:"subjectContains"`
	MaxMessageBytes int `json:"maxMessageBytes"`
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled bool `json:"enabled"`
}
func smtpView(item *model.SMTPRoute) model.SMTPRouteView { copy:=*item;copy.Password="";return model.SMTPRouteView{SMTPRoute:copy,PasswordConfigured:item.Password!=""} }
func (a *ConnectorAPI) GetSMTPRoutes(ctx *gin.Context){items,err:=a.DB.GetSMTPRoutes();if !successOrAbort(ctx,500,err){return};out:=make([]model.SMTPRouteView,0,len(items));for _,item:=range items{out=append(out,smtpView(item))};ctx.JSON(200,out)}
func (a *ConnectorAPI) CreateSMTPRoute(ctx *gin.Context){var p smtpRouteParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.ApplicationID){return};if p.MaxMessageBytes<=0{p.MaxMessageBytes=5<<20};if p.MaxMessageBytes>25<<20{ctx.AbortWithError(400,errors.New("SMTP maximum message size cannot exceed 25 MiB"));return};item:=&model.SMTPRoute{Name:p.Name,ApplicationID:p.ApplicationID,Recipient:strings.ToLower(strings.TrimSpace(p.Recipient)),AllowedCIDRs:p.AllowedCIDRs,SenderContains:strings.TrimSpace(p.SenderContains),SubjectContains:strings.TrimSpace(p.SubjectContains),MaxMessageBytes:p.MaxMessageBytes,Username:p.Username,Password:p.Password,Enabled:p.Enabled};if !successOrAbort(ctx,500,a.DB.SaveSMTPRoute(item)){return};ctx.JSON(201,smtpView(item))}
func (a *ConnectorAPI) UpdateSMTPRoute(ctx *gin.Context){withID(ctx,"id",func(id uint){item,err:=a.DB.GetSMTPRouteByID(id);if !successOrAbort(ctx,500,err){return};if item==nil{ctx.AbortWithStatus(404);return};var p smtpRouteParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.ApplicationID){return};if p.MaxMessageBytes<=0{p.MaxMessageBytes=5<<20};if p.MaxMessageBytes>25<<20{ctx.AbortWithError(400,errors.New("SMTP maximum message size cannot exceed 25 MiB"));return};item.Name=p.Name;item.ApplicationID=p.ApplicationID;item.Recipient=strings.ToLower(strings.TrimSpace(p.Recipient));item.AllowedCIDRs=p.AllowedCIDRs;item.SenderContains=strings.TrimSpace(p.SenderContains);item.SubjectContains=strings.TrimSpace(p.SubjectContains);item.MaxMessageBytes=p.MaxMessageBytes;item.Username=p.Username;if p.Password!=""{item.Password=p.Password};item.Enabled=p.Enabled;if !successOrAbort(ctx,500,a.DB.SaveSMTPRoute(item)){return};ctx.JSON(200,smtpView(item))})}
func (a *ConnectorAPI) DeleteSMTPRoute(ctx *gin.Context){withID(ctx,"id",func(id uint){successOrAbort(ctx,500,a.DB.DeleteSMTPRoute(id))})}

type rssParams struct { Name string `json:"name" binding:"required"`; ApplicationID uint `json:"applicationId" binding:"required"`; URL string `json:"url" binding:"required"`; IntervalMinutes int `json:"intervalMinutes"`; TitleContains string `json:"titleContains"`; CategoryContains string `json:"categoryContains"`; Priority int `json:"priority"`; Enabled bool `json:"enabled"` }
func (a *ConnectorAPI) GetRSS(ctx *gin.Context){items,err:=a.DB.GetRSSMonitors();if !successOrAbort(ctx,500,err){return};ctx.JSON(200,items)}
func (a *ConnectorAPI) CreateRSS(ctx *gin.Context){var p rssParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.ApplicationID){return};if err:=connectors.ValidateConnectorURL(p.URL);err!=nil{ctx.AbortWithError(400,err);return};if p.IntervalMinutes<1{p.IntervalMinutes=15};item:=&model.RSSMonitor{Name:p.Name,ApplicationID:p.ApplicationID,URL:p.URL,IntervalMinutes:p.IntervalMinutes,TitleContains:strings.TrimSpace(p.TitleContains),CategoryContains:strings.TrimSpace(p.CategoryContains),Priority:p.Priority,Enabled:p.Enabled};if !successOrAbort(ctx,500,a.DB.SaveRSSMonitor(item)){return};ctx.JSON(201,item)}
func (a *ConnectorAPI) UpdateRSS(ctx *gin.Context){withID(ctx,"id",func(id uint){item,err:=a.DB.GetRSSMonitorByID(id);if !successOrAbort(ctx,500,err){return};if item==nil{ctx.AbortWithStatus(404);return};var p rssParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.ApplicationID){return};if err:=connectors.ValidateConnectorURL(p.URL);err!=nil{ctx.AbortWithError(400,err);return};if p.IntervalMinutes<1{p.IntervalMinutes=15};item.Name=p.Name;item.ApplicationID=p.ApplicationID;item.URL=p.URL;item.IntervalMinutes=p.IntervalMinutes;item.TitleContains=strings.TrimSpace(p.TitleContains);item.CategoryContains=strings.TrimSpace(p.CategoryContains);item.Priority=p.Priority;item.Enabled=p.Enabled;if !successOrAbort(ctx,500,a.DB.SaveRSSMonitor(item)){return};ctx.JSON(200,item)})}
func (a *ConnectorAPI) DeleteRSS(ctx *gin.Context){withID(ctx,"id",func(id uint){successOrAbort(ctx,500,a.DB.DeleteRSSMonitor(id))})}

type syslogParams struct { Name string `json:"name" binding:"required"`; ApplicationID uint `json:"applicationId" binding:"required"`; Facility int `json:"facility"`; MaxSeverity int `json:"maxSeverity"`; AllowedCIDRs string `json:"allowedCidrs"`; DeduplicateSeconds int `json:"deduplicateSeconds"`; Enabled bool `json:"enabled"` }
func (a *ConnectorAPI) GetSyslog(ctx *gin.Context){items,err:=a.DB.GetSyslogRoutes();if !successOrAbort(ctx,500,err){return};ctx.JSON(200,items)}
func (a *ConnectorAPI) CreateSyslog(ctx *gin.Context){var p syslogParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.ApplicationID){return};if p.Facility < -1 || p.Facility>23 || p.MaxSeverity<0 || p.MaxSeverity>7{ctx.AbortWithError(400,errors.New("invalid syslog facility or severity"));return};if p.DeduplicateSeconds<0||p.DeduplicateSeconds>86400{ctx.AbortWithError(400,errors.New("syslog duplicate window must be between 0 and 86400 seconds"));return};item:=&model.SyslogRoute{Name:p.Name,ApplicationID:p.ApplicationID,Facility:p.Facility,MaxSeverity:p.MaxSeverity,AllowedCIDRs:p.AllowedCIDRs,DeduplicateSeconds:p.DeduplicateSeconds,Enabled:p.Enabled};if !successOrAbort(ctx,500,a.DB.SaveSyslogRoute(item)){return};ctx.JSON(201,item)}
func (a *ConnectorAPI) UpdateSyslog(ctx *gin.Context){withID(ctx,"id",func(id uint){item,err:=a.DB.GetSyslogRouteByID(id);if !successOrAbort(ctx,500,err){return};if item==nil{ctx.AbortWithStatus(404);return};var p syslogParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.ApplicationID){return};if p.Facility < -1 || p.Facility>23 || p.MaxSeverity<0 || p.MaxSeverity>7{ctx.AbortWithError(400,errors.New("invalid syslog facility or severity"));return};if p.DeduplicateSeconds<0||p.DeduplicateSeconds>86400{ctx.AbortWithError(400,errors.New("syslog duplicate window must be between 0 and 86400 seconds"));return};item.Name=p.Name;item.ApplicationID=p.ApplicationID;item.Facility=p.Facility;item.MaxSeverity=p.MaxSeverity;item.AllowedCIDRs=p.AllowedCIDRs;item.DeduplicateSeconds=p.DeduplicateSeconds;item.Enabled=p.Enabled;if !successOrAbort(ctx,500,a.DB.SaveSyslogRoute(item)){return};ctx.JSON(200,item)})}
func (a *ConnectorAPI) DeleteSyslog(ctx *gin.Context){withID(ctx,"id",func(id uint){successOrAbort(ctx,500,a.DB.DeleteSyslogRoute(id))})}

type calendarParams struct { Name string `json:"name" binding:"required"`; ApplicationID uint `json:"applicationId" binding:"required"`; URL string `json:"url" binding:"required"`; IntervalMinutes int `json:"intervalMinutes"`; NotifyBeforeMinutes int `json:"notifyBeforeMinutes"`; TitleContains string `json:"titleContains"`; LocationContains string `json:"locationContains"`; Priority int `json:"priority"`; Enabled bool `json:"enabled"` }
func (a *ConnectorAPI) GetCalendars(ctx *gin.Context){items,err:=a.DB.GetCalendarMonitors();if !successOrAbort(ctx,500,err){return};ctx.JSON(200,items)}
func (a *ConnectorAPI) CreateCalendar(ctx *gin.Context){var p calendarParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.ApplicationID){return};if err:=connectors.ValidateConnectorURL(p.URL);err!=nil{ctx.AbortWithError(400,err);return};if p.IntervalMinutes<1{p.IntervalMinutes=15};if p.NotifyBeforeMinutes<0{p.NotifyBeforeMinutes=0};item:=&model.CalendarMonitor{Name:p.Name,ApplicationID:p.ApplicationID,URL:p.URL,IntervalMinutes:p.IntervalMinutes,NotifyBeforeMinutes:p.NotifyBeforeMinutes,TitleContains:strings.TrimSpace(p.TitleContains),LocationContains:strings.TrimSpace(p.LocationContains),Priority:p.Priority,Enabled:p.Enabled};if !successOrAbort(ctx,500,a.DB.SaveCalendarMonitor(item)){return};ctx.JSON(201,item)}
func (a *ConnectorAPI) UpdateCalendar(ctx *gin.Context){withID(ctx,"id",func(id uint){item,err:=a.DB.GetCalendarMonitorByID(id);if !successOrAbort(ctx,500,err){return};if item==nil{ctx.AbortWithStatus(404);return};var p calendarParams;if err:=ctx.ShouldBindJSON(&p);err!=nil{return};if !a.channel(ctx,p.ApplicationID){return};if err:=connectors.ValidateConnectorURL(p.URL);err!=nil{ctx.AbortWithError(400,err);return};if p.IntervalMinutes<1{p.IntervalMinutes=15};item.Name=p.Name;item.ApplicationID=p.ApplicationID;item.URL=p.URL;item.IntervalMinutes=p.IntervalMinutes;item.NotifyBeforeMinutes=p.NotifyBeforeMinutes;item.TitleContains=strings.TrimSpace(p.TitleContains);item.LocationContains=strings.TrimSpace(p.LocationContains);item.Priority=p.Priority;item.Enabled=p.Enabled;if !successOrAbort(ctx,500,a.DB.SaveCalendarMonitor(item)){return};ctx.JSON(200,item)})}
func (a *ConnectorAPI) DeleteCalendar(ctx *gin.Context){withID(ctx,"id",func(id uint){successOrAbort(ctx,500,a.DB.DeleteCalendarMonitor(id))})}
