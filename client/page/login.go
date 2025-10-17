package page

import (
	"im/client/common"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func LoginPage(ctx *common.Context) fyne.Window {
	w := ctx.App.NewWindow("欢迎登录")
	w.Resize(fyne.NewSize(450, 550))
	w.CenterOnScreen()

	// 创建标题
	title := canvas.NewText("即时通讯", theme.Color(theme.ColorNameForeground))
	title.TextSize = 28
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	subtitle := canvas.NewText("连接你我，沟通无界", theme.Color(theme.ColorNameForeground))
	subtitle.TextSize = 14
	subtitle.Alignment = fyne.TextAlignCenter


	// 创建输入框
	accountEntry := widget.NewEntry()
	accountEntry.SetPlaceHolder("请输入账号（任意用户名均可）")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("请输入密码（123456）")

	// 错误提示标签
	errorLabel := widget.NewLabel("")
	errorLabel.Alignment = fyne.TextAlignCenter
	errorLabel.TextStyle = fyne.TextStyle{Italic: true}
	errorLabel.Hide()

	// 记住密码选项
	rememberCheck := widget.NewCheck("记住密码", func(checked bool) {
		// TODO: 实现记住密码功能
	})

	// 登录按钮
	loginButton := widget.NewButton("登 录", func() {
		// 清除之前的错误提示
		errorLabel.Hide()

		// 验证输入
		account := strings.TrimSpace(accountEntry.Text)
		password := strings.TrimSpace(passwordEntry.Text)

		if account == "" {
			errorLabel.SetText("❌ 请输入账号")
			errorLabel.Show()
			return
		}

		if password == "" {
			errorLabel.SetText("❌ 请输入密码")
			errorLabel.Show()
			return
		}

		if len(account) < 3 {
			errorLabel.SetText("❌ 账号长度至少3个字符")
			errorLabel.Show()
			return
		}

		if len(password) < 6 {
			errorLabel.SetText("❌ 密码长度至少6个字符")
			errorLabel.Show()
			return
		}

		// TODO: 实际的登录验证逻辑
		ctx.Account = account
		ctx.Password = password

		// 登录成功
		ctx.LoginPage.Close()
		ctx.HomePage.Show()
	})
	loginButton.Importance = widget.HighImportance

	// 注册按钮
	registerButton := widget.NewButton("注 册", func() {
		dialog.ShowInformation("提示", "注册功能开发中...", w)
	})

	// 忘记密码链接
	forgotPasswordLabel := widget.NewLabel("忘记密码？")
	forgotPasswordLabel.Alignment = fyne.TextAlignCenter
	forgotPasswordLabel.TextStyle = fyne.TextStyle{Italic: true}

	// 创建可点击的忘记密码按钮
	forgotPasswordBtn := widget.NewButton("忘记密码？", func() {
		dialog.ShowInformation("提示", "密码找回功能开发中...", w)
	})
	forgotPasswordBtn.Importance = widget.LowImportance

	// 设置Enter键触发登录
	accountEntry.OnSubmitted = func(string) {
		passwordEntry.FocusGained()
	}
	passwordEntry.OnSubmitted = func(string) {
		loginButton.OnTapped()
	}

	// 主要内容区域
	content := container.NewVBox(
		layout.NewSpacer(),

		// 标题区域
		container.NewVBox(
			layout.NewSpacer(),
			container.NewCenter(title),
			container.NewCenter(subtitle),
		),

		layout.NewSpacer(),

		// 表单区域
		container.NewVBox(
			// 账号输入
			container.NewBorder(
				nil, nil,
				container.NewHBox(
					widget.NewIcon(theme.AccountIcon()),
					layout.NewSpacer(),
				),
				nil,
				accountEntry,
			),

			layout.NewSpacer(),

			// 密码输入
			container.NewBorder(
				nil, nil,
				container.NewHBox(
					widget.NewIcon(theme.VisibilityOffIcon()),
					layout.NewSpacer(),
				),
				nil,
				passwordEntry,
			),

			// 错误提示
			errorLabel,

			// 记住密码
			container.NewHBox(
				rememberCheck,
				layout.NewSpacer(),
				forgotPasswordBtn,
			),
		),

		layout.NewSpacer(),

		// 按钮区域
		container.NewVBox(
			loginButton,
			registerButton,
		),

		layout.NewSpacer(),

		// 底部版权信息
		container.NewCenter(
			widget.NewLabel("© 2025 即时通讯系统"),
		),

		layout.NewSpacer(),
	)

	// 添加内边距
	paddedContent := container.NewPadded(
		container.NewPadded(content),
	)

	w.SetContent(paddedContent)

	return w
}
