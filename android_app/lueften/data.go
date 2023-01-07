package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

var outsideAddr = "192.168.0.56:16083" // 10:52:1c:5f:43:b0
var insideAddr = "192.168.0.6:16083"   // 4c:11:ae:8d:00:c0

const waitReplyTimeout = time.Second * 3

type sensorReadings struct {
	Temperature float64 `json:"T"`
	RelHumidity float64 `json:"rH"`
	AbsHumidity float64 `json:"aH"`
	Pressure    float64 `json:"p"`
}

// getReadings obtains sensor values from given address
func getReadings(addr string) (s sensorReadings, err error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return s, err
	}
	con, err := net.DialUDP("udp", nil, a)
	if err != nil {
		fmt.Println(err)
		return s, err
	}
	defer con.Close()

	con.SetDeadline(time.Now().Add(waitReplyTimeout))
	if _, err := con.Write([]byte("ping")); err != nil {
		return s, err
	}

	buf := make([]byte, 128)
	n, err := bufio.NewReader(con).Read(buf)
	if err != nil {
		return s, err
	}

	err = json.Unmarshal(buf[:n], &s)
	return s, err
}

func updateParams(u *ui, a fyne.App) {
	addrOut := a.Preferences().String("outsideAddr")
	addrIn := a.Preferences().String("insideAddr")

	res0, err0 := getReadings(addrOut)
	if err0 != nil {
		e := strings.Join(strings.SplitAfter(err0.Error(), "->"), "\n")
		d := dialog.NewError(errors.New(e), u.mainWin)
		sz := u.mainWin.Canvas().Size()
		d.Resize(fyne.NewSize(sz.Width, sz.Height*0.4))
		d.Show()
	}

	if err0 == nil {
		u.outside.rH.SetValue(res0.RelHumidity / 100)
		u.outside.aH.SetText(fmt.Sprintf("%.2f g/m\u00b3", calcAbsHum(res0.RelHumidity, res0.Temperature)))
		u.outside.temp.SetText(fmt.Sprintf("%.1f \u2103", res0.Temperature))
		u.outside.pres.SetText(fmt.Sprintf("%.1f hPa", res0.Pressure))
	}

	res1, err1 := getReadings(addrIn)
	if err1 != nil {
		e := strings.Join(strings.SplitAfter(err1.Error(), "->"), "\n")
		d := dialog.NewError(errors.New(e), u.mainWin)
		sz := u.mainWin.Canvas().Size()
		d.Resize(fyne.NewSize(sz.Width, sz.Height*0.4))
		d.Show()
	}
	if err1 == nil {
		u.inside.rH.SetValue(res1.RelHumidity / 100)
		u.inside.aH.SetText(fmt.Sprintf("%.2f g/m\u00b3", calcAbsHum(res1.RelHumidity, res1.Temperature)))
		u.inside.temp.SetText(fmt.Sprintf("%.1f \u2103", res1.Temperature))
	}

	if err0 == nil && err1 == nil {
		u.ventingPossible.SetText(" Lüften? Könnte man... ")
		if res0.AbsHumidity >= res1.AbsHumidity {
			u.ventingPossible.SetText(" Lüften? Besser nicht... ")
		}

		u.lastContact.Text = "last update: " + time.Now().Format("15:04:05 h")
		u.lastContact.Refresh()

	}
}
