//go:generate /home/flonblnx/go/bin/fyne bundle -o bundled.go Icon.png

package main

import (
	"fyne.io/fyne/v2/app"
)

const winTitle = "solltIchLueften"

func main() {
	a := app.NewWithID(winTitle)
	a.Preferences().SetString("outsideAddr", outsideAddr)
	a.Preferences().SetString("insideAddr", insideAddr)

	w := a.NewWindow(winTitle)

	u := &ui{mainWin: w}

	w.SetContent(u.makeUI(a))
	updateParams(u, a) // set initial values

	w.ShowAndRun()

	// fyne package -os android -appID Lueft.en -icon Icon.png

}
