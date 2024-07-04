package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"path/filepath"
)

var appPath string
var app = tview.NewApplication()
var pages = tview.NewPages()
var modalQuit = tview.NewModal()
var flexBasicInfo = tview.NewFlex()
var flexStorage = tview.NewFlex()
var flexPackages = tview.NewFlex()
var flexMirror = tview.NewFlex()
var flexInstall = tview.NewFlex()

func main() {
	ex, err := os.Executable()
	check(err)
	appPath = filepath.Dir(ex)

	_, err = execCommand("which kubectl", 0)
	if err != nil {
		panic("kubectl is not found!")
	}
	_, err = execCommand("which helm", 0)
	if err != nil {
		panic("helm is not found!")
	}
	_, err = execCommand("kubectl cluster-info", 0)
	if err != nil {
		panic("Can't connect to k8s cluster!")
	}

	initFlexBasicInfo()

	pages.AddPage("Quit", modalQuit, true, false)

	pages.AddPage("Basic Info", flexBasicInfo, true, true)
	pages.AddPage("Storage", flexStorage, true, false)
	pages.AddPage("Packages", flexPackages, true, false)
	pages.AddPage("Mirror", flexMirror, true, false)
	pages.AddPage("Install", flexInstall, true, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			showQuitModal()
			return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
		}

		return event
	})

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
