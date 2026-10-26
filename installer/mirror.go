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
	// DaoCloud's GHCR mirror rejects anonymous pulls, so default to the NJU one.
	"GHCR_CONTAINER_MIRROR": "ghcr.nju.edu.cn",
}
var useOneMirror = false
var oneMirror = ""

// mirrorEnvVars returns the IDO_*_CONTAINER_MIRROR variables passed to the
// install scripts. Upstream registries are used unless mirrors are enabled.
func mirrorEnvVars(enableMirror, useOneMirror bool, oneMirror string, mirrors map[string]string) map[string]string {
	upstream := map[string]string{
		"DOCKER_CONTAINER_MIRROR": "docker.io",
		"QUAY_CONTAINER_MIRROR":   "quay.io",
		"K8S_CONTAINER_MIRROR":    "registry.k8s.io",
		"GCR_CONTAINER_MIRROR":    "k8s-gcr.io",
		"GHCR_CONTAINER_MIRROR":   "ghcr.io",
	}

	envs := make(map[string]string, len(upstream))
	for key, upstreamValue := range upstream {
		value := upstreamValue
		if enableMirror {
			if useOneMirror {
				value = oneMirror
			} else {
				value = mirrors[key]
			}
		}
		envs["IDO_"+key] = value
	}
	return envs
}

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
