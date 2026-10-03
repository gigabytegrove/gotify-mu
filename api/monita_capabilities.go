package api

import "github.com/gin-gonic/gin"

// MonitaCapabilitiesAPI exposes a stable discovery contract for Monita-aware
// clients. A stock Monita server does not expose this route, allowing clients
// to fall back cleanly to the upstream feature set on HTTP 404.
type MonitaCapabilitiesAPI struct {
	Version string
}

type MonitaCapabilities struct {
	Product    string                `json:"product"`
	Version    string                `json:"version"`
	APIVersion int                   `json:"apiVersion"`
	Features   MonitaCapabilityFlags `json:"features"`
}

type MonitaCapabilityFlags struct {
	SharedChannels       bool `json:"sharedChannels"`
	GlobalChannels       bool `json:"globalChannels"`
	ChannelTypes         bool `json:"channelTypes"`
	ChannelImages        bool `json:"channelImages"`
	ChatImages           bool `json:"chatImages"`
	NotificationImages   bool `json:"notificationImages"`
	MessageControls      bool `json:"messageControls"`
	ChatChannels         bool `json:"chatChannels"`
	MemberPosting        bool `json:"memberPosting"`
	SenderIdentity       bool `json:"senderIdentity"`
	PerUserArchive       bool `json:"perUserArchive"`
	PerChannelMute       bool `json:"perChannelMute"`
	MembershipManagement bool `json:"membershipManagement"`
	OwnershipTransfer    bool `json:"ownershipTransfer"`
	UserGroups           bool `json:"userGroups"`
	AuditLog             bool `json:"auditLog"`
	TypingPresence       bool `json:"typingPresence"`
	ChatNotifications    bool `json:"chatNotifications"`
	Mentions             bool `json:"mentions"`
}

func (a *MonitaCapabilitiesAPI) Get(ctx *gin.Context) {
	ctx.JSON(200, MonitaCapabilities{
		Product:    "monita",
		Version:    a.Version,
		APIVersion: 1,
		Features: MonitaCapabilityFlags{
			SharedChannels:       true,
			GlobalChannels:       true,
			ChannelTypes:         true,
			ChannelImages:        true,
			ChatImages:           true,
			NotificationImages:   true,
			MessageControls:      true,
			ChatChannels:         true,
			MemberPosting:        true,
			SenderIdentity:       true,
			PerUserArchive:       true,
			PerChannelMute:       true,
			MembershipManagement: true,
			OwnershipTransfer:    true,
			UserGroups:           true,
			AuditLog:             true,
			TypingPresence:       true,
			ChatNotifications:    true,
			Mentions:             true,
		},
	})
}
