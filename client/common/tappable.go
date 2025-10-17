package common

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// 1. 定义我们自己的可点击容器结构
type tappableContainer struct {
	widget.BaseWidget // 嵌入 BaseWidget 以获得基础功能
	container         *fyne.Container
	onTapped          func()
}

// 2. 创建一个新的构造函数
func NewTappableContainer(content fyne.CanvasObject, tapped func()) *tappableContainer {
	c := &tappableContainer{
		container: container.NewStack(content), // 使用 Stack 布局让内容填满
		onTapped:  tapped,
	}
	c.ExtendBaseWidget(c) // 非常重要的一步，完成自身初始化
	return c
}

// 3. 实现 fyne.Widget 接口中的 CreateRenderer
func (c *tappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return &tappableContainerRenderer{
		container: c.container,
	}
}

// 4. 实现 fyne.Tappable 接口
func (c *tappableContainer) Tapped(_ *fyne.PointEvent) {
	if c.onTapped != nil {
		c.onTapped()
	}
}

// --- 自定义组件的渲染器 ---

type tappableContainerRenderer struct {
	container *fyne.Container
}

func (r *tappableContainerRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

func (r *tappableContainerRenderer) MinSize() fyne.Size {
	return r.container.MinSize()
}

func (r *tappableContainerRenderer) Refresh() {
	r.container.Refresh()
}

func (r *tappableContainerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.container}
}

func (r *tappableContainerRenderer) Destroy() {}
