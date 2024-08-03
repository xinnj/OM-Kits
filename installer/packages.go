package main

import (
	"errors"
	"github.com/rivo/tview"
	"golang.org/x/exp/slices"
	"strconv"
	"strings"
)

type NfsProvisionerConfig struct {
	server       string
	path         string
	mountOptions string
}

func (config *NfsProvisionerConfig) validate() error {
	if config.server == "" {
		return errors.New("NFS server is empty.")
	}
	if config.path == "" {
		return errors.New("NFS path is empty.")
	}
	return nil
}

type PrometheusConfig struct {
	alertmanagerStorageSizeGi int
	grafanaStorageSizeGi      int
	prometheusStorageSizeGi   int
	storageClass              string
}

func (config *PrometheusConfig) validate() error {
	if config.alertmanagerStorageSizeGi == 0 {
		return errors.New("Alert manager storage size is 0.")
	}
	if config.grafanaStorageSizeGi == 0 {
		return errors.New("Grafana storage size is 0.")
	}
	if config.prometheusStorageSizeGi == 0 {
		return errors.New(" Prometheus storage size is 0.")
	}
	return nil
}

var installLocalPathProvisioner = false
var installNfsProvisioner = false
var installPrometheus = false

var nfsProvisionerConfig = NfsProvisionerConfig{
	server:       "",
	path:         "/",
	mountOptions: "vers=3,nolock,proto=tcp,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2,noresvport",
}

var prometheusConfig = PrometheusConfig{
	alertmanagerStorageSizeGi: 10,
	grafanaStorageSizeGi:      5,
	prometheusStorageSizeGi:   10,
	storageClass:              "",
}

var storageClasses []string
var packages = []string{"Local-Path Provisioner", "NFS Provisioner", "Prometheus"}
var listPackages = tview.NewList()
var formPackage = tview.NewForm()

func initFlexPackages() {
	storageClasses = getStorageClasses()
	flexPackages.Clear()
	flexList := tview.NewFlex()
	flexList.SetTitle("Packages").SetBorder(true)

	if listPackages.GetItemCount() == 0 {
		for index, item := range packages {
			listPackages.AddItem(item, "", rune(97+index), nil)
		}

		mainText, _ := listPackages.GetItemText(0)
		selectPackage(0, mainText)

		listPackages.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
			selectPackage(index, mainText)
		})
	}

	formDown := tview.NewForm()
	formDown.AddButton("Next", func() {
		if installNfsProvisioner {
			err := nfsProvisionerConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		if installPrometheus {
			err := prometheusConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		initFlexMirror()
		pages.SwitchToPage("Mirror")
	})

	formDown.AddButton("Back", func() {
		pages.SwitchToPage("Basic Info")
	})

	formDown.AddButton("Quit", func() {
		showQuitModal()
	})

	flexList.
		AddItem(listPackages, 0, 1, true).
		AddItem(formPackage, 0, 3, false)

	flexPackages.SetDirection(tview.FlexRow).
		AddItem(flexList, 0, 1, true).
		AddItem(formDown, 3, 1, false)
}

func selectPackage(index int, mainText string) {
	formPackage.Clear(true)
	listPackages.SetItemText(index, mainText, "")
	switch mainText {
	case "Local-Path Provisioner":
		formPackage.AddCheckbox("Install Local-Path Provisioner: ", installLocalPathProvisioner, func(checked bool) {
			installLocalPathProvisioner = checked
			selectPackage(index, mainText)
		})
		if installLocalPathProvisioner {
			listPackages.SetItemText(index, mainText, "Will install")
		}
	case "NFS Provisioner":
		formPackage.AddCheckbox("Install NFS Provisioner: ", installNfsProvisioner, func(checked bool) {
			installNfsProvisioner = checked
			selectPackage(index, mainText)
		})
		if installNfsProvisioner {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Server: ", nfsProvisionerConfig.server,
				0, nil, func(text string) {
					nfsProvisionerConfig.server = text
				})
			formPackage.AddInputField("Path: ", nfsProvisionerConfig.path,
				0, nil, func(text string) {
					nfsProvisionerConfig.path = text
				})
			formPackage.AddInputField("Mount options: ", nfsProvisionerConfig.mountOptions,
				0, nil, func(text string) {
					nfsProvisionerConfig.mountOptions = text
				})
		}
	case "Prometheus":
		formPackage.AddCheckbox("Install Prometheus: ", installPrometheus, func(checked bool) {
			installPrometheus = checked
			selectPackage(index, mainText)
		})
		if installPrometheus {
			listPackages.SetItemText(index, mainText, "Will install")

			initialOption := slices.Index(storageClasses, prometheusConfig.storageClass)
			formPackage.AddDropDown("Storage Class: ", storageClasses, initialOption, func(option string, optionIndex int) {
				prometheusConfig.storageClass = option
			})
			formPackage.AddInputField("Alert manager storage size (Gi): ", strconv.Itoa(prometheusConfig.alertmanagerStorageSizeGi),
				0, nil, func(text string) {
					prometheusConfig.alertmanagerStorageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Grafana storage size (Gi): ", strconv.Itoa(prometheusConfig.grafanaStorageSizeGi),
				0, nil, func(text string) {
					prometheusConfig.grafanaStorageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Prometheus storage size (Gi): ", strconv.Itoa(prometheusConfig.prometheusStorageSizeGi),
				0, nil, func(text string) {
					prometheusConfig.prometheusStorageSizeGi, _ = strconv.Atoi(text)
				})
		}
	}
}

func getStorageClasses() []string {
	var storageClasses []string

	result, err := execCommand("kubectl get sc --no-headers -o custom-columns=\":metadata.name\"", 0)
	check(err)
	storageClasses = strings.Split(strings.TrimSpace(string(result)), "\n")

	if !slices.Contains(storageClasses, "local-path") {
		storageClasses = append(storageClasses, "local-path")
	}
	if !slices.Contains(storageClasses, "nfs-client") {
		storageClasses = append(storageClasses, "nfs-client")
	}

	return storageClasses
}
