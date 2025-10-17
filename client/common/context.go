package common

import "fyne.io/fyne/v2"

type Context struct {
	App       fyne.App
	LoginPage fyne.Window
	HomePage  fyne.Window
	Account   string
	Password  string
}
