package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/slices"
)

var enableMirror = false
var mirrors = map[string]string{
	"DOCKER_CONTAINER_MIRROR": "docker.m.daocloud.io",
	"QUAY_CONTAINER_MIRROR":   "quay.m.daocloud.io",
	"K8S_CONTAINER_MIRROR":    "k8s.m.daocloud.io",
	"GCR_CONTAINER_MIRROR":    "gcr.m.daocloud.io",
}
var useOneMirror = false
var oneMirror = ""

func initFlexMirror() {
	flexMirror.Clear()
	formMirror := tview.NewForm()
	formMirror.SetTitle("Public Download Mirror").SetBorder(true)

	formMirror.AddCheckbox("Enable public download mirror: ", enableMirror, func(checked bool) {
		enableMirror = checked
		flexMirror.Clear()
		initFlexMirror()
	})

	if enableMirror {
		formMirror.AddCheckbox("Use one mirror for all: ", useOneMirror, func(checked bool) {
			useOneMirror = checked
			flexMirror.Clear()
			initFlexMirror()
		})

		if useOneMirror {
			formMirror.AddInputField("Mirror for all: ", oneMirror, 0, nil, func(text string) {
				oneMirror = text
			})
		} else {
			var keyOrdered []string
			for k, _ := range mirrors {
				keyOrdered = append(keyOrdered, k)
			}
			slices.Sort(keyOrdered)

			for _, item := range keyOrdered {
				key := item
				formMirror.AddInputField(item+": ", mirrors[key], 0, nil, func(text string) {
					mirrors[key] = text
				})
			}
		}
	}

	formDown := tview.NewForm()

	formDown.AddButton("Install", func() {
		if enableMirror {
			if useOneMirror {
				if oneMirror == "" {
					showErrorModal("Mirror is empty.")
					return
				}
			} else {
				for k, v := range mirrors {
					if v == "" {
						showErrorModal(k + " is empty.")
						return
					}
				}
			}
		}

		initFlexInstall()
		pages.SwitchToPage("Install")
	})

	formDown.AddButton("Back", func() {
		pages.SwitchToPage("Packages")
	})

	formDown.AddButton("Quit", func() {
		showQuitModal()
	})

	flexMirror.SetDirection(tview.FlexRow).
		AddItem(formMirror, 0, 1, true).
		AddItem(formDown, 3, 1, false)

	app.SetFocus(formMirror)

	formMirror.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlN || event.Key() == tcell.KeyCtrlP {
			app.SetFocus(formDown)
		}
		return event
	})

	formDown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlN || event.Key() == tcell.KeyCtrlP {
			app.SetFocus(formMirror)
		}
		return event
	})
}
