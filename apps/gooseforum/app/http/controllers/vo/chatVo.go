package vo

import "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"

// ChatItemVo summarizes one conversation in the messages list.
type ChatItemVo struct {
	Id           uint64 `json:"id"` // user_chat_config id
	PeerId       uint64 `json:"peerId"`
	PeerUsername string `json:"peerUsername"`
	// PeerNickname 当前昵称；备注名显示 note(display name) 需要，无昵称时省略。
	PeerNickname string `json:"peerNickname,omitempty"`
	PeerAvatar   string `json:"peerAvatar"`
	LastMsg      string `json:"lastMsg"`
	LastMsgTime  string `json:"lastMsgTime"`
	UnreadCount  uint   `json:"unreadCount"`
	ConvId       uint64 `json:"convId"`
	PeerUrl      string `json:"peerUrl"`
}

// MessageVo represents one chat message decorated for the current viewer.
type MessageVo struct {
	Forwarded *messages.ForwardedBundle `json:"forwarded,omitempty"`
	Id        uint64                    `json:"id"`
	SenderId  uint64                    `json:"senderId"`
	Content   string                    `json:"content"`
	MsgType   int8                      `json:"msgType"`
	IsRead    int                       `json:"isRead"`
	CreatedAt string                    `json:"createdAt"`
	IsSelf    bool                      `json:"isSelf"`
}
