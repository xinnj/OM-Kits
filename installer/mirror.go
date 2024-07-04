package main

import (
	"github.com/rivo/tview"
	"golang.org/x/exp/slices"
)

var enableMirror = false
var mirrors map[string]string

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
		if mirrors == nil {
			mirrors = map[string]string{
				"DOCKER_CONTAINER_MIRROR": "docker.m.daocloud.io",
				"QUAY_CONTAINER_MIRROR":   "quay.m.daocloud.io",
				"K8S_CONTAINER_MIRROR":    "k8s.m.daocloud.io",
				"GCR_CONTAINER_MIRROR":    "k8s-gcr.m.daocloud.io",
			}
		}

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

	formDown := tview.NewForm()

	formDown.AddButton("Install", func() {
		if enableMirror {
			for k, v := range mirrors {
				if v == "" {
					showErrorModal(k + " is empty.")
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
}
