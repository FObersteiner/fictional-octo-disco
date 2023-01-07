package main

import (
	"image/color"
	"net"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type params struct {
	rH   *widget.ProgressBar
	aH   *widget.Label
	temp *widget.Label
	pres *widget.Label
}

func NewParamsStruct() *params {
	return &params{widget.NewProgressBar(), widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel("")}
}

type ui struct {
	mainWin         fyne.Window
	outside         *params
	inside          *params
	ventingPossible *widget.Label
	lastContact     *canvas.Text
}

func (u *ui) makeUI(a fyne.App) fyne.CanvasObject {

	u.outside = NewParamsStruct()
	u.outside.aH.SetText("-999")
	u.outside.temp.SetText("-999")
	u.outside.pres.SetText("-999")

	u.inside = NewParamsStruct()
	u.inside.aH.SetText("-999")
	u.inside.temp.SetText("-999")

	u.ventingPossible = widget.NewLabel(" Lüften? ")
	u.lastContact = canvas.NewText("last update:", color.Gray{128})

	outside := container.NewVBox(
		canvas.NewLine(color.White),
		widget.NewLabel("# Draussen #"),
		canvas.NewLine(color.NRGBA{0x10, 0xfa, 0x07, 0xff}),

		container.New(
			layout.NewFormLayout(),
			widget.NewLabel("rH:"), u.outside.rH,
			widget.NewLabel("aH:"), u.outside.aH,
			widget.NewLabel("T:"), u.outside.temp,
			widget.NewLabel("p:"), u.outside.pres,
		),
	)

	inside := container.NewVBox(
		canvas.NewLine(color.White),
		widget.NewLabel("# In der Bude #"),
		canvas.NewLine(color.NRGBA{0x10, 0xfa, 0x07, 0xff}),

		container.New(
			layout.NewFormLayout(),
			widget.NewLabel("rH:"), u.inside.rH,
			widget.NewLabel("aH:"), u.inside.aH,
			widget.NewLabel("T:"), u.inside.temp,
		),
	)

	eval := container.NewVBox(
		canvas.NewLine(color.White),
		u.ventingPossible,
		canvas.NewLine(color.White),
	)

	rf := widget.NewButton("Refresh", func() { u.refresh(a) })
	ips := widget.NewButton("Set IPs", func() { u.setIP(a) })

	content := container.NewVBox(outside, inside, eval, rf, layout.NewSpacer(), ips)

	return container.NewBorder(nil, u.lastContact, nil, nil, u.lastContact, content)
}

func (u *ui) refresh(a fyne.App) {
	updateParams(u, a)
}

func (u *ui) setIP(a fyne.App) {
	entry0 := widget.NewEntry()
	entry0.SetPlaceHolder(a.Preferences().String("outsideAddr"))
	entry1 := widget.NewEntry()
	entry1.SetPlaceHolder(a.Preferences().String("insideAddr"))

	d := dialog.NewForm("Set IP Addresses", "OK", "Cancel",
		[]*widget.FormItem{
			{Text: "Outside", Widget: entry0},
			{Text: "Inside", Widget: entry1}},
		func(set bool) {
			if set {
				if entry0.Text != "" {
					if _, err := net.ResolveUDPAddr("udp", entry0.Text); err == nil {
						a.Preferences().SetString("outsideAddr", entry0.Text)
					}
				}
				if entry1.Text != "" {
					if _, err := net.ResolveUDPAddr("udp", entry1.Text); err == nil {
						a.Preferences().SetString("insideAddr", entry1.Text)
					}
				}
			}
		},
		u.mainWin)

	sz := u.mainWin.Canvas().Size()
	d.Resize(fyne.NewSize(sz.Width, sz.Height*0.5))
	d.Show()
}
