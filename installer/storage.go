package main

import (
	"github.com/rivo/tview"
	"golang.org/x/exp/slices"
	"strings"
)

type StorageClassType struct {
	ceph string
	nfs  string
}
type NfsConfig struct {
	server string
	path   string
}

var storageClassType = StorageClassType{
	ceph: "ceph-filesystem",
	nfs:  "nfs-client",
}
var storageClass = ""
var useExistingSC = true
var existingSCs []string
var nfsConfig NfsConfig

func initFlexStorage() {
	flexStorage.Clear()
	formStorage := tview.NewForm()
	formStorage.SetTitle("Storage").SetBorder(true)

	formStorage.AddCheckbox("Use existing StorageClass: ", useExistingSC, func(checked bool) {
		useExistingSC = true
		initFlexStorage()
	})
	if useExistingSC {
		if len(existingSCs) == 0 {
			result, err := execCommand("kubectl get sc --no-headers -o custom-columns=\":metadata.name\"", 0)
			check(err)
			existingSCs = strings.Split(strings.TrimSpace(string(result)), "\n")
		}

		initialOption := slices.Index(existingSCs, storageClass)
		formStorage.AddDropDown("  Select a StorageClass: ", existingSCs, initialOption, func(option string, optionIndex int) {
			storageClass = option
		})
	}

	formStorage.AddCheckbox("Use CEPH: ", !useExistingSC && storageClass == storageClassType.ceph, func(checked bool) {
		useExistingSC = false
		storageClass = storageClassType.ceph
		initFlexStorage()
	})

	formStorage.AddCheckbox("Use NFS: ", !useExistingSC && storageClass == storageClassType.nfs, func(checked bool) {
		useExistingSC = false
		storageClass = storageClassType.nfs
		initFlexStorage()
	})
	if !useExistingSC && storageClass == storageClassType.nfs {
		formStorage.AddInputField("  NFS server: ", nfsConfig.server, 0, nil, func(text string) {
			nfsConfig.server = strings.Trim(text, " ")
		})
		formStorage.AddInputField("  NFS path: ", nfsConfig.path, 0, nil, func(text string) {
			nfsConfig.path = strings.Trim(text, " ")
		})
	}

	formDown := tview.NewForm()

	formDown.AddButton("Next", func() {
		if storageClass == "" {
			showErrorModal("Please select a StorageClass.")
			return
		}

		if storageClass == storageClassType.nfs && !useExistingSC {
			if nfsConfig.server == "" {
				showErrorModal("NFS server is empty.")
				return
			}
			if nfsConfig.path == "" {
				showErrorModal("NFS path is empty.")
				return
			}
		}

		initFlexPackages()
		pages.SwitchToPage("Packages")
	})

	formDown.AddButton("Back", func() {
		pages.SwitchToPage("Basic Info")
	})

	formDown.AddButton("Quit", func() {
		showQuitModal()
	})

	flexStorage.SetDirection(tview.FlexRow).
		AddItem(formStorage, 0, 1, true).
		AddItem(formDown, 3, 1, false)
}
