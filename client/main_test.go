package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func makeUI() (*widget.Label, *widget.Entry) {
	out := widget.NewLabel("Hello, World!")
	in := widget.NewEntry()
	in.OnChanged = func(s string) {
		out.SetText("Hello, " + s)
	}
	return out, in
}
func TestMain(t *testing.T) {
	out, in := makeUI()
	if out.Text != "Hello, World!" {
		t.Errorf("out.Text: %s", out.Text)
	}
	test.Type(in, "Andy")
	if out.Text != "Hello, Andy" {
		t.Errorf("in.Text: %s", in.Text)
	}

}
