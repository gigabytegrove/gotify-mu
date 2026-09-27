package api

import "github.com/gin-gonic/gin"

// MUCapabilitiesAPI exposes a stable discovery contract for Gotify MU-aware
// clients. A stock Gotify server does not expose this route, allowing clients
// to fall back cleanly to the upstream feature set on HTTP 404.
type MUCapabilitiesAPI struct {
	Version string
}

type MUCapabilities struct {
	Product    string            `json:"product"`
	Version    string            `json:"version"`
	APIVersion int               `json:"apiVersion"`
	Features   MUCapabilityFlags `json:"features"`
}

type MUCapabilityFlags struct {
	SharedChannels       bool `json:"sharedChannels"`
	GlobalChannels       bool `json:"globalChannels"`
	ChannelTypes         bool `json:"channelTypes"`
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
}

func (a *MUCapabilitiesAPI) Get(ctx *gin.Context) {
	ctx.JSON(200, MUCapabilities{
		Product:    "gotify-mu",
		Version:    a.Version,
		APIVersion: 1,
		Features: MUCapabilityFlags{
			SharedChannels:       true,
			GlobalChannels:       true,
			ChannelTypes:         true,
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
		},
	})
}
