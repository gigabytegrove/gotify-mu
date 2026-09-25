package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
)

const maxAttachmentBytes int64 = 25 << 20

type CollaborationDatabase interface {
	GetMessageByID(id uint) (*model.Message, error)
	GetApplicationByID(id uint) (*model.Application, error)
	GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error)
	GetUserByID(id uint) (*model.User, error)
	GetThreadMessagesForUser(userID, rootID uint) ([]*model.Message, error)
	AddMessageReaction(messageID, userID uint, emoji string) error
	DeleteMessageReaction(messageID, userID uint, emoji string) error
	SaveMessageWorkflow(item *model.MessageWorkflow) error
	GetMessageWorkflow(messageID uint) (*model.MessageWorkflow, error)
	MarkMessageRead(userID, messageID uint, at time.Time) error
	MarkMessageUnread(userID, messageID uint) error
	ReplaceMessageMentions(messageID uint, userIDs []uint) error
	CreateMessageAttachment(item *model.MessageAttachment) error
	GetMessageAttachment(id uint) (*model.MessageAttachment, error)
	DeleteMessageAttachment(id uint) error
	GetMessageTemplates(userID uint) ([]*model.MessageTemplate, error)
	GetMessageTemplateByID(userID, id uint) (*model.MessageTemplate, error)
	SaveMessageTemplate(item *model.MessageTemplate) error
	DeleteMessageTemplate(userID, id uint) error
	GetSavedMessageSearches(userID uint) ([]*model.SavedMessageSearch, error)
	GetSavedMessageSearchByID(userID, id uint) (*model.SavedMessageSearch, error)
	SaveSavedMessageSearch(item *model.SavedMessageSearch) error
	DeleteSavedMessageSearch(userID, id uint) error
	SearchMessages(userID uint, filter model.MessageSearchFilter) ([]*model.Message, error)
}

type CollaborationDispatcher interface {
	StoreAndDeliver(message *model.Message) (*model.MessageExternal, error)
}

type CollaborationAPI struct {
	DB            CollaborationDatabase
	Dispatcher    CollaborationDispatcher
	AttachmentDir string
}

func (a *CollaborationAPI) messageAccess(ctx *gin.Context, messageID uint) (*model.Message, *model.Application, *model.ApplicationMembership, *model.User, bool) {
	userID := auth.GetUserID(ctx)
	message, err := a.DB.GetMessageByID(messageID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) {
		return nil, nil, nil, nil, false
	}
	if message == nil {
		ctx.AbortWithError(http.StatusNotFound, errors.New("message not found"))
		return nil, nil, nil, nil, false
	}
	app, err := a.DB.GetApplicationByID(message.ApplicationID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) {
		return nil, nil, nil, nil, false
	}
	membership, err := a.DB.GetApplicationMembership(message.ApplicationID, userID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) {
		return nil, nil, nil, nil, false
	}
	user, err := a.DB.GetUserByID(userID)
	if !successOrAbort(ctx, http.StatusInternalServerError, err) {
		return nil, nil, nil, nil, false
	}
	if app == nil || membership == nil || user == nil {
		ctx.AbortWithError(http.StatusNotFound, errors.New("message not found"))
		return nil, nil, nil, nil, false
	}
	return message, app, membership, user, true
}

func canPostToChannel(app *model.Application, membership *model.ApplicationMembership, user *model.User) bool {
	if user == nil || app == nil || membership == nil {
		return false
	}
	if user.Admin || app.UserID == user.ID {
		return true
	}
	role := membership.EffectiveRole
	return role == model.ChannelRoleManager ||
		role == model.ChannelRolePublisher ||
		(role == model.ChannelRoleMember && app.AllowMemberPost)
}

func canManageMessage(app *model.Application, membership *model.ApplicationMembership, user *model.User, message *model.Message) bool {
	if user == nil || app == nil || membership == nil || message == nil {
		return false
	}
	if user.Admin || app.UserID == user.ID || message.SenderUserID == user.ID {
		return true
	}
	return membership.EffectiveRole == model.ChannelRoleManager
}

type replyParams struct {
	Message        string `json:"message" binding:"required"`
	Title          string `json:"title"`
	Priority       *int   `json:"priority"`
	MentionUserIDs []uint `json:"mentionUserIds"`
}

func (a *CollaborationAPI) Reply(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		parent, app, membership, user, ok := a.messageAccess(ctx, id)
		if !ok {
			return
		}
		if !canPostToChannel(app, membership, user) {
			ctx.AbortWithError(http.StatusForbidden, errors.New("your Channel role does not allow posting replies"))
			return
		}
		var params replyParams
		if err := ctx.ShouldBindJSON(&params); err != nil {
			return
		}
		priority := parent.Priority
		if params.Priority != nil {
			priority = *params.Priority
		}
		title := strings.TrimSpace(params.Title)
		if title == "" {
			title = "Re: " + parent.Title
		}
		root := parent.ThreadRootMessageID
		if root == 0 {
			root = parent.ID
		}
		message := &model.Message{
			ApplicationID: app.ID,
			Title: title,
			Message: params.Message,
			Priority: priority,
			Date: time.Now(),
			SenderUserID: user.ID,
			SenderName: displayUserName(user),
			ReplyToMessageID: parent.ID,
			ThreadRootMessageID: root,
		}
		external, err := a.Dispatcher.StoreAndDeliver(message)
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		if len(params.MentionUserIDs) > 0 {
			if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.ReplaceMessageMentions(message.ID, params.MentionUserIDs)) {
				return
			}
		}
		ctx.JSON(http.StatusCreated, external)
	})
}

func displayUserName(user *model.User) string {
	if user == nil {
		return ""
	}
	if strings.TrimSpace(user.DisplayName) != "" {
		return user.DisplayName
	}
	return user.Name
}

func (a *CollaborationAPI) Thread(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		message, _, _, _, ok := a.messageAccess(ctx, id)
		if !ok {
			return
		}
		root := message.ThreadRootMessageID
		if root == 0 {
			root = message.ID
		}
		items, err := a.DB.GetThreadMessagesForUser(auth.GetUserID(ctx), root)
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		ctx.JSON(http.StatusOK, toExternalMessages(items))
	})
}

type reactionParams struct {
	Emoji string `json:"emoji" binding:"required"`
}

func (a *CollaborationAPI) AddReaction(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if _, _, _, _, ok := a.messageAccess(ctx, id); !ok {
			return
		}
		var params reactionParams
		if err := ctx.ShouldBindJSON(&params); err != nil {
			return
		}
		emoji := strings.TrimSpace(params.Emoji)
		if emoji == "" || len([]rune(emoji)) > 16 {
			ctx.AbortWithError(http.StatusBadRequest, errors.New("invalid reaction"))
			return
		}
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.AddMessageReaction(id, auth.GetUserID(ctx), emoji)) {
			return
		}
		ctx.Status(http.StatusNoContent)
	})
}

func (a *CollaborationAPI) DeleteReaction(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if _, _, _, _, ok := a.messageAccess(ctx, id); !ok {
			return
		}
		emoji := strings.TrimSpace(ctx.Query("emoji"))
		if emoji == "" {
			ctx.AbortWithError(http.StatusBadRequest, errors.New("emoji is required"))
			return
		}
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.DeleteMessageReaction(id, auth.GetUserID(ctx), emoji)) {
			return
		}
		ctx.Status(http.StatusNoContent)
	})
}

type assignmentParams struct {
	UserID uint `json:"userId"`
}

func (a *CollaborationAPI) Assign(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		message, app, membership, user, ok := a.messageAccess(ctx, id)
		if !ok {
			return
		}
		var params assignmentParams
		if err := ctx.ShouldBindJSON(&params); err != nil {
			return
		}
		if params.UserID != 0 && params.UserID != user.ID && !canManageMessage(app, membership, user, message) {
			ctx.AbortWithError(http.StatusForbidden, errors.New("manager access is required to assign another user"))
			return
		}
		if params.UserID != 0 {
			targetMembership, err := a.DB.GetApplicationMembership(app.ID, params.UserID)
			if !successOrAbort(ctx, http.StatusInternalServerError, err) {
				return
			}
			if targetMembership == nil {
				ctx.AbortWithError(http.StatusBadRequest, errors.New("assigned user is not a Channel member"))
				return
			}
		}
		workflow, err := a.DB.GetMessageWorkflow(id)
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		if workflow == nil {
			workflow = &model.MessageWorkflow{MessageID: id, Status: "open"}
		}
		workflow.AssignedUserID = params.UserID
		workflow.UpdatedAt = time.Now()
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.SaveMessageWorkflow(workflow)) {
			return
		}
		ctx.JSON(http.StatusOK, workflow)
	})
}

type statusParams struct {
	Status string `json:"status" binding:"required"`
}

func (a *CollaborationAPI) SetStatus(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		_, _, _, user, ok := a.messageAccess(ctx, id)
		if !ok {
			return
		}
		var params statusParams
		if err := ctx.ShouldBindJSON(&params); err != nil {
			return
		}
		status := strings.ToLower(strings.TrimSpace(params.Status))
		if status != "open" && status != "resolved" {
			ctx.AbortWithError(http.StatusBadRequest, errors.New("status must be open or resolved"))
			return
		}
		workflow, err := a.DB.GetMessageWorkflow(id)
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		if workflow == nil {
			workflow = &model.MessageWorkflow{MessageID: id}
		}
		workflow.Status = status
		workflow.UpdatedAt = time.Now()
		if status == "resolved" {
			now := time.Now()
			workflow.ResolvedBy = user.ID
			workflow.ResolvedAt = &now
		} else {
			workflow.ResolvedBy = 0
			workflow.ResolvedAt = nil
		}
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.SaveMessageWorkflow(workflow)) {
			return
		}
		ctx.JSON(http.StatusOK, workflow)
	})
}

func (a *CollaborationAPI) MarkRead(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if _, _, _, _, ok := a.messageAccess(ctx, id); !ok {
			return
		}
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.MarkMessageRead(auth.GetUserID(ctx), id, time.Now())) {
			return
		}
		ctx.Status(http.StatusNoContent)
	})
}

func (a *CollaborationAPI) MarkUnread(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		if _, _, _, _, ok := a.messageAccess(ctx, id); !ok {
			return
		}
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.MarkMessageUnread(auth.GetUserID(ctx), id)) {
			return
		}
		ctx.Status(http.StatusNoContent)
	})
}

func randomStorageName() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func safeFilename(header *multipart.FileHeader) string {
	name := filepath.Base(strings.TrimSpace(header.Filename))
	if name == "" || name == "." {
		return "attachment"
	}
	if len(name) > 240 {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(base) > 200 {
			base = base[:200]
		}
		name = base + ext
	}
	return name
}

func (a *CollaborationAPI) UploadAttachment(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		message, app, membership, user, ok := a.messageAccess(ctx, id)
		if !ok {
			return
		}
		if !canManageMessage(app, membership, user, message) {
			ctx.AbortWithError(http.StatusForbidden, errors.New("you cannot add attachments to this message"))
			return
		}
		header, err := ctx.FormFile("attachment")
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, errors.New("attachment is required"))
			return
		}
		if header.Size <= 0 || header.Size > maxAttachmentBytes {
			ctx.AbortWithError(http.StatusRequestEntityTooLarge, errors.New("attachment must be between 1 byte and 25 MiB"))
			return
		}
		source, err := header.Open()
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		defer source.Close()
		if err := os.MkdirAll(a.AttachmentDir, 0o700); !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		storageName, err := randomStorageName()
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		targetPath := filepath.Join(a.AttachmentDir, storageName)
		target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		written, copyErr := io.Copy(target, io.LimitReader(source, maxAttachmentBytes+1))
		closeErr := target.Close()
		if copyErr != nil || closeErr != nil || written > maxAttachmentBytes {
			_ = os.Remove(targetPath)
			if written > maxAttachmentBytes {
				ctx.AbortWithError(http.StatusRequestEntityTooLarge, errors.New("attachment exceeds 25 MiB"))
			} else {
				ctx.AbortWithError(http.StatusInternalServerError, errors.New("attachment could not be saved"))
			}
			return
		}
		item := &model.MessageAttachment{
			MessageID: id,
			Filename: safeFilename(header),
			ContentType: header.Header.Get("Content-Type"),
			Size: written,
			StorageName: storageName,
		}
		if err := a.DB.CreateMessageAttachment(item); err != nil {
			_ = os.Remove(targetPath)
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		ctx.JSON(http.StatusCreated, model.MessageAttachmentView{
			ID:item.ID, Filename:item.Filename, ContentType:item.ContentType, Size:item.Size,
			URL:"/message/"+strconv.FormatUint(uint64(id),10)+"/attachment/"+strconv.FormatUint(uint64(item.ID),10),
		})
	})
}

func (a *CollaborationAPI) DownloadAttachment(ctx *gin.Context) {
	withID(ctx, "id", func(messageID uint) {
		if _, _, _, _, ok := a.messageAccess(ctx, messageID); !ok {
			return
		}
		attachmentID64, err := strconv.ParseUint(ctx.Param("attachmentId"), 10, 64)
		if err != nil {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		item, err := a.DB.GetMessageAttachment(uint(attachmentID64))
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		if item == nil || item.MessageID != messageID {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		path := filepath.Join(a.AttachmentDir, filepath.Base(item.StorageName))
		ctx.Header("Content-Disposition", "attachment; filename*=UTF-8''"+urlEncodeFilename(item.Filename))
		if item.ContentType != "" {
			ctx.Header("Content-Type", item.ContentType)
		}
		ctx.File(path)
	})
}

func urlEncodeFilename(value string) string {
	replacer := strings.NewReplacer("%", "%25", " ", "%20", "\"", "%22", "'", "%27", ";", "%3B", "\r", "", "\n", "")
	return replacer.Replace(value)
}

func (a *CollaborationAPI) DeleteAttachment(ctx *gin.Context) {
	withID(ctx, "id", func(messageID uint) {
		message, app, membership, user, ok := a.messageAccess(ctx, messageID)
		if !ok {
			return
		}
		if !canManageMessage(app, membership, user, message) {
			ctx.AbortWithError(http.StatusForbidden, errors.New("you cannot remove attachments from this message"))
			return
		}
		attachmentID64, err := strconv.ParseUint(ctx.Param("attachmentId"), 10, 64)
		if err != nil {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		item, err := a.DB.GetMessageAttachment(uint(attachmentID64))
		if !successOrAbort(ctx, http.StatusInternalServerError, err) {
			return
		}
		if item == nil || item.MessageID != messageID {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		if !successOrAbort(ctx, http.StatusInternalServerError, a.DB.DeleteMessageAttachment(item.ID)) {
			return
		}
		_ = os.Remove(filepath.Join(a.AttachmentDir, filepath.Base(item.StorageName)))
		ctx.Status(http.StatusNoContent)
	})
}

type templateParams struct {
	Name string `json:"name" binding:"required"`
	ApplicationID uint `json:"applicationId"`
	Title string `json:"title"`
	Message string `json:"message" binding:"required"`
	Priority int `json:"priority"`
	Extras map[string]any `json:"extras"`
}

func templateView(item *model.MessageTemplate) model.MessageTemplateView {
	view := model.MessageTemplateView{
		ID:item.ID, Name:item.Name, ApplicationID:item.ApplicationID, Title:item.Title,
		Message:item.Message, Priority:item.Priority, CreatedAt:item.CreatedAt, UpdatedAt:item.UpdatedAt,
	}
	if len(item.Extras) > 0 {
		_ = json.Unmarshal(item.Extras, &view.Extras)
	}
	return view
}

func (a *CollaborationAPI) GetTemplates(ctx *gin.Context) {
	items, err := a.DB.GetMessageTemplates(auth.GetUserID(ctx))
	if !successOrAbort(ctx, http.StatusInternalServerError, err) { return }
	out := make([]model.MessageTemplateView,0,len(items))
	for _, item := range items { out=append(out,templateView(item)) }
	ctx.JSON(http.StatusOK,out)
}

func (a *CollaborationAPI) SaveTemplate(ctx *gin.Context) {
	var params templateParams
	if err := ctx.ShouldBindJSON(&params); err != nil { return }
	extras,_:=json.Marshal(params.Extras)
	item:=&model.MessageTemplate{UserID:auth.GetUserID(ctx),Name:strings.TrimSpace(params.Name),ApplicationID:params.ApplicationID,Title:params.Title,Message:params.Message,Priority:params.Priority,Extras:extras}
	if !successOrAbort(ctx,http.StatusInternalServerError,a.DB.SaveMessageTemplate(item)){return}
	ctx.JSON(http.StatusCreated,templateView(item))
}

func (a *CollaborationAPI) UpdateTemplate(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){
		item,err:=a.DB.GetMessageTemplateByID(auth.GetUserID(ctx),id)
		if !successOrAbort(ctx,500,err){return}
		if item==nil{ctx.AbortWithStatus(404);return}
		var params templateParams
		if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
		extras,_:=json.Marshal(params.Extras)
		item.Name=strings.TrimSpace(params.Name);item.ApplicationID=params.ApplicationID;item.Title=params.Title;item.Message=params.Message;item.Priority=params.Priority;item.Extras=extras
		if !successOrAbort(ctx,500,a.DB.SaveMessageTemplate(item)){return}
		ctx.JSON(200,templateView(item))
	})
}

func (a *CollaborationAPI) DeleteTemplate(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){
		if !successOrAbort(ctx,500,a.DB.DeleteMessageTemplate(auth.GetUserID(ctx),id)){return}
		ctx.Status(http.StatusNoContent)
	})
}

func parseOptionalInt(raw string) *int {
	if strings.TrimSpace(raw)=="" { return nil }
	value,err:=strconv.Atoi(raw);if err!=nil{return nil}
	return &value
}
func parseOptionalTime(raw string) *time.Time {
	if strings.TrimSpace(raw)=="" {return nil}
	value,err:=time.Parse(time.RFC3339,raw);if err!=nil{return nil}
	return &value
}

func (a *CollaborationAPI) Search(ctx *gin.Context) {
	appID,_:=strconv.ParseUint(ctx.Query("applicationId"),10,64)
	filter:=model.MessageSearchFilter{
		Query:ctx.Query("q"), ApplicationID:uint(appID), MinPriority:parseOptionalInt(ctx.Query("minPriority")),
		MaxPriority:parseOptionalInt(ctx.Query("maxPriority")), Sender:ctx.Query("sender"), Status:ctx.Query("status"),
		Acknowledged:ctx.Query("acknowledged"), From:parseOptionalTime(ctx.Query("from")), To:parseOptionalTime(ctx.Query("to")),
	}
	filter.Limit,_=strconv.Atoi(ctx.Query("limit"))
	items,err:=a.DB.SearchMessages(auth.GetUserID(ctx),filter)
	if !successOrAbort(ctx,500,err){return}
	ctx.JSON(200,toExternalMessages(items))
}

func (a *CollaborationAPI) GetSavedSearches(ctx *gin.Context) {
	items,err:=a.DB.GetSavedMessageSearches(auth.GetUserID(ctx));if !successOrAbort(ctx,500,err){return};ctx.JSON(200,items)
}
func (a *CollaborationAPI) SaveSearch(ctx *gin.Context) {
	var item model.SavedMessageSearch;if err:=ctx.ShouldBindJSON(&item);err!=nil{return};item.ID=0;item.UserID=auth.GetUserID(ctx)
	if strings.TrimSpace(item.Name)==""{ctx.AbortWithError(400,errors.New("name is required"));return}
	if !successOrAbort(ctx,500,a.DB.SaveSavedMessageSearch(&item)){return};ctx.JSON(201,item)
}
func (a *CollaborationAPI) UpdateSearch(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){item,err:=a.DB.GetSavedMessageSearchByID(auth.GetUserID(ctx),id);if !successOrAbort(ctx,500,err){return};if item==nil{ctx.AbortWithStatus(404);return}
		var params model.SavedMessageSearch;if err:=ctx.ShouldBindJSON(&params);err!=nil{return}
		item.Name=params.Name;item.Query=params.Query;item.ApplicationID=params.ApplicationID;item.MinPriority=params.MinPriority;item.MaxPriority=params.MaxPriority;item.Sender=params.Sender;item.Status=params.Status;item.Acknowledged=params.Acknowledged
		if !successOrAbort(ctx,500,a.DB.SaveSavedMessageSearch(item)){return};ctx.JSON(200,item)
	})
}
func (a *CollaborationAPI) DeleteSearch(ctx *gin.Context) {
	withID(ctx,"id",func(id uint){if !successOrAbort(ctx,500,a.DB.DeleteSavedMessageSearch(auth.GetUserID(ctx),id)){return};ctx.Status(http.StatusNoContent)})
}
