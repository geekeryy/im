package xstrings

import (
	"fmt"
	"math/rand"
	"time"
)

// 生成随机昵称
func NewRandomUserName() string {
	adjectives := []string{"快乐的", "聪明的", "勇敢的", "温柔的", "活泼的", "可爱的", "优雅的", "神秘的", "阳光的", "梦幻的"}
	nouns := []string{"小猫", "小狗", "小鸟", "小熊", "小兔", "小鱼", "小鹿", "小狐", "小象", "小龙"}

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	adjective := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	number := rand.Intn(99)
	return fmt.Sprintf("%s%s%02d", adjective, noun, number)
}

// 生成随机头像
func NewRandomAvatar() string {
	avatars := []string{
		"girl1.png",
		"girl2.png",
		"boy1.png",
		"boy2.png",
	}
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	avatar := avatars[rand.Intn(len(avatars))]
	return avatar
}
