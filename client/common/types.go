package common

type ChatMessage struct {
	SessionUuid string // 会话UUID
	Content   string // 消息内容
	IsSent    bool   // true 表示发送的消息，false 表示接收的消息
	AvatarURI string // 头像资源路径，支持本地文件或后续的远程 URL
}

// Session 会话数据结构
type Session struct {
	UUID        string // 会话UUID
	Name        string // 联系人名称
	AvatarURI   string // 头像路径
	LastMessage string // 最后一条消息
	UnreadCount int    // 未读消息数
	LastTime    string // 最后消息时间
}