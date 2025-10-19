package page

import (
	"fmt"
	"im/client/common"
	apigatewayService "im/server/apigateway/rpc/service"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// CustomEntry 自定义输入框，支持 Enter 发送，Cmd+Enter 换行
type CustomEntry struct {
	widget.Entry
	onEnter func()
}

// NewCustomEntry 创建自定义输入框
func NewCustomEntry(onEnter func()) *CustomEntry {
	entry := &CustomEntry{onEnter: onEnter}
	entry.ExtendBaseWidget(entry)
	entry.MultiLine = true
	entry.Wrapping = fyne.TextWrapBreak // 按字符换行，适配中文长句
	return entry
}

// TypedKey 处理键盘事件
func (e *CustomEntry) TypedKey(key *fyne.KeyEvent) {
	// 处理 Enter 键
	if key.Name == fyne.KeyReturn || key.Name == fyne.KeyEnter {
		// 检查是否同时按下修饰键（通过桌面驱动接口）
		if d, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
			mods := d.CurrentKeyModifiers()
			// Cmd+Enter 或 Shift+Enter 换行
			if mods&fyne.KeyModifierSuper != 0 || mods&fyne.KeyModifierShift != 0 {
				// 插入换行符
				e.Entry.TypedKey(key)
				return
			}
		}
		// 单独按 Enter 发送消息
		if e.onEnter != nil {
			e.onEnter()
		}
		return
	}
	// 其他按键正常处理
	e.Entry.TypedKey(key)
}

// maxWidthLayout 自定义布局，限制内容的最大宽度
type maxWidthLayout struct {
	maxWidth float32
}

func newMaxWidthLayout(maxWidth float32) fyne.Layout {
	return &maxWidthLayout{maxWidth: maxWidth}
}

func (l *maxWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}

	// 计算实际宽度：不超过最大宽度和可用宽度
	width := fyne.Min(l.maxWidth, size.Width)

	// 对每个对象进行布局
	for _, obj := range objects {
		// 给对象设置宽度约束，让高度自适应
		objMinSize := obj.MinSize()
		height := fyne.Max(objMinSize.Height, size.Height)
		obj.Resize(fyne.Size{Width: width, Height: height})
		obj.Move(fyne.NewPos(0, 0))
	}
}

func (l *maxWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}

	// 返回固定的最大宽度，让高度由内容决定
	// 这样可以避免在 MinSize 中调用 Resize
	minWidth := l.maxWidth
	minHeight := float32(0)

	for _, obj := range objects {
		objSize := obj.MinSize()

		// 高度取最大值
		if objSize.Height > minHeight {
			minHeight = objSize.Height
		}
	}

	return fyne.Size{Width: minWidth, Height: minHeight}
}

// compactVBoxLayout 紧凑的垂直布局，减少间距
type compactVBoxLayout struct {
	padding float32
}

func (c *compactVBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minWidth := float32(0)
	minHeight := float32(0)
	for i, obj := range objects {
		if !obj.Visible() {
			continue
		}
		objSize := obj.MinSize()
		if objSize.Width > minWidth {
			minWidth = objSize.Width
		}
		minHeight += objSize.Height
		if i < len(objects)-1 {
			minHeight += c.padding
		}
	}
	return fyne.NewSize(minWidth, minHeight)
}

func (c *compactVBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for i, obj := range objects {
		if !obj.Visible() {
			continue
		}
		objHeight := obj.MinSize().Height
		obj.Resize(fyne.NewSize(size.Width, objHeight))
		obj.Move(fyne.NewPos(0, y))
		y += objHeight
		if i < len(objects)-1 {
			y += c.padding
		}
	}
}

// createSessionItem 创建仿微信风格的会话列表项（精致小巧版）
func (homeCtx *HomePageContext) createSessionItem(session common.Session) fyne.CanvasObject {
	// 创建头像（40x40）
	avatar := canvas.NewImageFromFile(session.AvatarURI)
	avatar.FillMode = canvas.ImageFillContain
	avatar.SetMinSize(fyne.Size{Width: 40, Height: 40})
	avatar.Resize(fyne.Size{Width: 40, Height: 40})
	avatar.Move(fyne.NewPos(5, 0))

	// 创建头像容器（可能包含红点）
	var avatarContainer *fyne.Container
	if session.UnreadCount > 0 {
		// 创建红色圆点
		redDot := canvas.NewRectangle(color.RGBA{R: 255, G: 59, B: 48, A: 255})
		redDot.CornerRadius = 4
		redDot.Resize(fyne.Size{Width: 8, Height: 8})
		redDot.Move(fyne.NewPos(37, 0))

		// 使用 NewWithoutLayout 将头像和红点放在一起
		avatarContainer = container.NewWithoutLayout(avatar, redDot)
	} else {
		avatarContainer = container.NewWithoutLayout(avatar)
	}
	avatarContainer.Resize(fyne.Size{Width: 40, Height: 40})

	// 创建用户名标签
	nameLabel := widget.NewLabel(session.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// 创建最后一条消息预览标签
	lastMsgLabel := canvas.NewText("   "+session.LastMessage, color.Gray{Y: 128})
	lastMsgLabel.TextSize = 12

	// 创建时间标签
	timeLabel := widget.NewLabel(session.LastTime)
	timeLabel.Alignment = fyne.TextAlignTrailing

	// 顶部行：用户名和时间
	topRow := container.NewBorder(nil, nil, nil, timeLabel, nameLabel)

	// 中间信息布局：使用紧凑布局，间距设为0px（非常紧凑）
	middleInfo := container.New(&compactVBoxLayout{padding: 0}, topRow, lastMsgLabel)

	// 创建带左右边距的头像容器（左边距12px，右边距8px）
	avatarWithMargin := container.NewBorder(
		nil, nil, nil, nil,
		container.NewPadded(avatarContainer),
	)

	// 整体布局：头像 + 中间信息
	// 使用 HBox 让头像自然居中，并控制间距
	itemContent := container.NewBorder(
		nil, nil,
		avatarWithMargin,
		nil,
		middleInfo,
	)

	// 添加分隔线
	separator := canvas.NewRectangle(color.RGBA{R: 240, G: 240, B: 240, A: 255})
	separator.SetMinSize(fyne.Size{Height: 2})

	// 使用紧凑布局返回会话项
	sessionItem := common.NewTappableContainer(container.New(
		&compactVBoxLayout{padding: 4}, // 会话内容和分隔线之间4px间距
		itemContent,
		separator,
	), func() {

		homeCtx.MessageBox.RemoveAll()
		homeCtx.UsernName.Text = session.Name
		homeCtx.CurrentSessionUUID = session.UUID
		homeCtx.UsernName.Refresh()

		fyne.Do(func() {
			response, err := homeCtx.AppCtx.ApiGatewayClient.HistoryMessage(homeCtx.AppCtx.Ctx, &apigatewayService.HistoryMessageRequest{
				SessionUuid: session.UUID,
				StartSeqid:  0,
			})
			if err != nil {
				homeCtx.AppCtx.Logger.Error("Failed to get history message", "error", err)
				return
			}
			for _, msg := range response.Messages {
				homeCtx.AppCtx.Logger.Debug("message", "message", msg, "user_uuid", homeCtx.AppCtx.User.UUID)
				if msg.SenderUuid == homeCtx.AppCtx.User.UUID {
					homeCtx.MessageBox.Add(createSentMessage(common.ChatMessage{
						Content:   msg.Content,
						IsSent:    true,
						AvatarURI: fmt.Sprintf("assets/%s", msg.SenderAvatar),
					}))
				} else {
					homeCtx.MessageBox.Add(createReceivedMessage(common.ChatMessage{
						Content:   msg.Content,
						IsSent:    false,
						AvatarURI: fmt.Sprintf("assets/%s", msg.SenderAvatar),
					}))
				}
			}
			homeCtx.MessageBox.Refresh()

			sessionUserList, err := homeCtx.AppCtx.ApiGatewayClient.GetSessionUserList(homeCtx.AppCtx.Ctx, &apigatewayService.GetSessionUserListRequest{
				SessionUuid: session.UUID,
			})
			if err != nil {
				homeCtx.AppCtx.Logger.Error("Failed to get session user list", "error", err)
				return
			}
			users := make(map[string]common.User, 0)
			for _, user := range sessionUserList.Users {
				users[user.UserUuid] = common.User{
					UUID:   user.UserUuid,
					Name:   user.UserName,
					Avatar: fmt.Sprintf("assets/%s", user.UserAvatar),
				}
			}
			homeCtx.AppCtx.SessionUserTable[session.UUID] = users
		})

	})

	return sessionItem
}

// createReceivedMessage 创建接收到的消息组件（左侧布局）
func createReceivedMessage(msg common.ChatMessage) fyne.CanvasObject {
	// 创建头像
	avatar := canvas.NewImageFromFile(msg.AvatarURI)
	avatar.FillMode = canvas.ImageFillContain
	avatar.SetMinSize(fyne.Size{Width: 40, Height: 40})

	// 限制最大宽度为 400px
	maxWidth := float32(400)

	// 创建消息文本标签（统一使用 Label 以支持换行）
	messageLabel := widget.NewLabel(msg.Content)
	messageLabel.Wrapping = fyne.TextWrapWord

	// 创建气泡背景
	bg := canvas.NewRectangle(color.RGBA{R: 240, G: 240, B: 240, A: 255})
	bg.CornerRadius = 8

	// 创建气泡内容
	bubbleContent := container.NewPadded(messageLabel)
	messageBubble := container.NewStack(bg, bubbleContent)

	// 使用 VBox 包装气泡，然后用自定义布局限制最大宽度
	bubbleWrapper := container.NewVBox(messageBubble)
	bubbleWithMaxWidth := container.New(newMaxWidthLayout(maxWidth), bubbleWrapper)

	// 布局：[头像] [气泡] [spacer]
	msgRow := container.NewHBox(
		avatar,
		bubbleWithMaxWidth,
		layout.NewSpacer(),
	)

	return container.NewPadded(msgRow)
}

// createSentMessage 创建发送的消息组件（右侧布局，仿照微信）
func createSentMessage(msg common.ChatMessage) fyne.CanvasObject {
	// 创建头像
	avatar := canvas.NewImageFromFile(msg.AvatarURI)
	avatar.FillMode = canvas.ImageFillContain
	avatar.SetMinSize(fyne.Size{Width: 40, Height: 40})

	// 限制最大宽度为 400px
	maxWidth := float32(400)

	// 创建消息文本标签（统一使用 Label 以支持换行）
	messageLabel := widget.NewLabel(msg.Content)
	messageLabel.Wrapping = fyne.TextWrapWord

	// 创建消息气泡背景（微信绿色）
	bg := canvas.NewRectangle(color.RGBA{R: 149, G: 236, B: 105, A: 255})
	bg.CornerRadius = 8

	// 创建气泡内容
	bubbleContent := container.NewPadded(messageLabel)
	messageBubble := container.NewStack(bg, bubbleContent)

	// 使用 VBox 包装气泡，然后用自定义布局限制最大宽度
	bubbleWrapper := container.NewVBox(messageBubble)
	bubbleWithMaxWidth := container.New(newMaxWidthLayout(maxWidth), bubbleWrapper)

	// 布局：[spacer] [气泡] [头像]（右侧布局，仿照微信）
	msgRow := container.NewHBox(
		layout.NewSpacer(),
		bubbleWithMaxWidth,
		avatar,
	)

	// 添加上下边距
	return container.NewPadded(msgRow)
}

type HomePageContext struct {
	AppCtx             *common.Context
	Messages           []common.ChatMessage
	MessageBox         *fyne.Container
	UsernName          *widget.Label
	CurrentSessionUUID string // 当前会话UUID

}

func HomePage(ctx *common.Context) fyne.Window {
	homeCtx := &HomePageContext{
		AppCtx: ctx,
	}
	w := ctx.App.NewWindow("Home")
	w.Resize(fyne.Size{Width: 900, Height: 550})

	// 聊天消息列表
	messages := []common.ChatMessage{}

	// 使用VBox创建会话列表（更灵活）
	sessionBox := container.NewVBox()

	// 创建消息容器（VBox）
	messageBox := container.NewVBox()
	homeCtx.MessageBox = messageBox

	// 顶部用户名
	usernName := widget.NewLabel(ctx.User.Name)
	homeCtx.UsernName = usernName

	go func() {
		// for range time.NewTicker(5 * time.Second).C {
		fyne.Do(func() {
			sessionBox.RemoveAll()
			response, err := ctx.ApiGatewayClient.SessionList(ctx.Ctx, &apigatewayService.SessionListRequest{})
			if err != nil {
				ctx.Logger.Error("Failed to get session list", "error", err)
				return
			}
			for _, v := range response.Sessions {
				ctx.Logger.Debug("session", "session", v)
				session := common.Session{
					UUID:        v.Uuid,
					Name:        v.Name,
					AvatarURI:   fmt.Sprintf("assets/%s", v.Avatar),
					LastMessage: v.LastMessage,
					UnreadCount: int(v.UnreadCount),
					LastTime:    v.LastTime,
				}
				item := homeCtx.createSessionItem(session)
				sessionBox.Add(item)
			}
			sessionBox.Refresh()
		})
		// }
	}()

	go func() {
		for msg := range homeCtx.AppCtx.MessageReadChan {
			fyne.Do(func() {
				messageBox.Add(createReceivedMessage(msg))
				// messageBox.Refresh()
			})
		}
	}()

	// 左侧内容：会话列表滚动容器（隐藏滚动条样式）
	sessionScroll := container.NewVScroll(sessionBox)
	sessionScroll.ScrollToTop()
	leftContent := sessionScroll

	// 将现有消息添加到容器中
	for _, msg := range messages {
		if msg.IsSent {
			messageBox.Add(createSentMessage(msg))
		} else {
			messageBox.Add(createReceivedMessage(msg))
		}
	}

	// 创建滚动容器
	chatScroll := container.NewVScroll(messageBox)

	// 发送消息函数（需要提前定义，供 CustomEntry 使用）
	var inputEntry *CustomEntry
	sendMessage := func() {
		if inputEntry != nil && strings.TrimSpace(inputEntry.Text) != "" {
			// 创建新消息
			newMsg := common.ChatMessage{
				Content:     inputEntry.Text,
				IsSent:      true,
				AvatarURI:   fmt.Sprintf("assets/%s", ctx.User.Avatar),
				SessionUuid: homeCtx.CurrentSessionUUID,
			}
			messages = append(messages, newMsg)
			homeCtx.AppCtx.MessageWriteChan <- newMsg
			// 添加消息到界面
			messageBox.Add(createSentMessage(newMsg))

			// 清空输入框
			inputEntry.SetText("")

			// 刷新并滚动到底部
			messageBox.Refresh()
			chatScroll.ScrollToBottom()
		}
	}

	// 创建自定义输入框（支持 Enter 发送，Cmd+Enter 或 Shift+Enter 换行）
	inputEntry = NewCustomEntry(sendMessage)
	inputEntry.SetPlaceHolder("输入消息... (Enter发送，Cmd+Enter换行)")
	inputEntry.SetMinRowsVisible(3) // 设置最小可见行数

	// 发送按钮
	sendButton := widget.NewButton("发送", sendMessage)

	// 输入区域：输入框 + 发送按钮（水平布局）
	inputArea := container.NewBorder(
		nil, nil, nil, sendButton,
		inputEntry,
	)

	// 使用 Padded 容器为输入区域添加一些边距
	inputAreaWithPadding := container.NewPadded(inputArea)

	// 右侧内容：使用 Border 布局，标题在顶部，输入区域在底部，聊天区域占据中间
	rightContent := container.NewBorder(
		container.NewCenter(usernName), // 顶部：标题
		inputAreaWithPadding,           // 底部：输入区域
		nil, nil,
		chatScroll, // 中间：聊天滚动区域（自动填充剩余空间）
	)
	split := container.NewHSplit(
		leftContent,
		rightContent,
	)
	split.SetOffset(1.0 / 3.5)
	w.SetContent(split)
	return w
}
