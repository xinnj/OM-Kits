package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/slices"
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
	grafanaRootUrl            string
	adapterLang               string
	adapterSignature          string
	dingTalkEnabled           bool
	dingTalkToken             string
	dingTalkSecret            string
	feishuEnabled             bool
	feishuToken               string
	feishuSecret              string
	weChatEnabled             bool
	weChatToken               string
}

func (config *PrometheusConfig) validate() error {
	if config.alertmanagerStorageSizeGi == 0 {
		return errors.New("Alert manager storage size is 0.")
	}
	if config.grafanaStorageSizeGi == 0 {
		return errors.New("Grafana storage size is 0.")
	}
	if config.prometheusStorageSizeGi == 0 {
		return errors.New("Prometheus storage size is 0.")
	}
	if config.storageClass == "" {
		return errors.New("Prometheus storage class is empty.")
	}
	if config.grafanaRootUrl == "" {
		return errors.New("Grafana root URL is empty.")
	}

	switch config.adapterLang {
	case "zh", "en":
	default:
		config.adapterLang = "zh"
	}

	config.adapterSignature = strings.TrimSpace(config.adapterSignature)
	if config.adapterSignature == "" {
		config.adapterSignature = "monitoring"
	}

	config.dingTalkToken = strings.TrimSpace(config.dingTalkToken)
	config.dingTalkSecret = strings.TrimSpace(config.dingTalkSecret)
	config.feishuToken = strings.TrimSpace(config.feishuToken)
	config.feishuSecret = strings.TrimSpace(config.feishuSecret)
	config.weChatToken = strings.TrimSpace(config.weChatToken)

	if config.dingTalkEnabled {
		if config.dingTalkToken == "" {
			return errors.New("DingTalk bot token is required when DingTalk notifications are enabled.")
		}
		if config.dingTalkSecret == "" {
			return errors.New("DingTalk bot secret is required when DingTalk notifications are enabled.")
		}
		if strings.Contains(config.dingTalkToken, "://") {
			return errors.New("DingTalk bot token must be the access_token value, not the full webhook URL.")
		}
	}
	if config.feishuEnabled {
		if config.feishuToken == "" {
			return errors.New("Feishu bot token is required when Feishu notifications are enabled.")
		}
		if config.feishuSecret == "" {
			return errors.New("Feishu bot secret is required when Feishu notifications are enabled.")
		}
		if strings.Contains(config.feishuToken, "://") {
			return errors.New("Feishu bot token must be the webhook path token, not the full webhook URL.")
		}
	}
	if config.weChatEnabled {
		if config.weChatToken == "" {
			return errors.New("WeChat bot key is required when WeChat notifications are enabled.")
		}
		if strings.Contains(config.weChatToken, "://") {
			return errors.New("WeChat bot key must be the webhook key value, not the full webhook URL.")
		}
	}

	return nil
}

type LoggingConfig struct {
	collectNamespaces string
	storageClass      string
	esStorageSizeGi   int
	esDeleteAgeDay    int
	esColdAgeDay      int
	esWarmAgeDay      int
	nodeAffinity      bool
	errorLogAlert     bool
	enableMonitoring  bool
	resourcesPreset   string
}

func (config *LoggingConfig) validate() error {
	if config.esStorageSizeGi == 0 {
		return errors.New("Elasticsearch storage size is 0.")
	}
	if config.esDeleteAgeDay == 0 {
		return errors.New("Elasticsearch delete age is 0.")
	}
	if config.esColdAgeDay == 0 {
		return errors.New("Elasticsearch cold age is 0.")
	}
	if config.esWarmAgeDay == 0 {
		return errors.New("Elasticsearch warm age is 0.")
	}
	if config.storageClass == "" {
		return errors.New("Elasticsearch storage class is empty.")
	}
	if config.resourcesPreset == "" {
		return errors.New("Elasticsearch resources preset is empty.")
	}
	return nil
}

type PermissionManagerConfig struct {
	clusterName    string
	clusterAddress string
}

func (config *PermissionManagerConfig) validate() error {
	if config.clusterName == "" {
		return errors.New("Cluster name is empty.")
	}
	if config.clusterAddress == "" {
		return errors.New("Cluster address is empty.")
	}
	return nil
}

var installLocalPathProvisioner = false
var installNfsProvisioner = false
var installPrometheus = false
var installLogging = false
var installPermissionManager = false

var localPathProvisionerPath = "/data/local-path-provisioner"

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
	grafanaRootUrl:            "",
	adapterLang:               "zh",
	adapterSignature:          "monitoring",
	dingTalkEnabled:           false,
	feishuEnabled:             false,
	weChatEnabled:             false,
}

var loggingConfig = LoggingConfig{
	collectNamespaces: "",
	storageClass:      "",
	esStorageSizeGi:   20,
	esDeleteAgeDay:    365,
	esColdAgeDay:      30,
	esWarmAgeDay:      7,
	nodeAffinity:      true,
	errorLogAlert:     false,
	enableMonitoring:  false,
	resourcesPreset:   "",
}

var permissionManagerConfig = PermissionManagerConfig{
	clusterName:    "cluster.local",
	clusterAddress: "https://lb-apiserver.kubernetes.local:6443",
}

var storageClasses []string
var packages = []string{"Local-Path Provisioner", "NFS Provisioner", "Prometheus", "Logging", "Permission Manager"}
var listPackages = tview.NewList()
var formPackage = tview.NewForm()
var adapterStateLoaded bool

func initFlexPackages() {
	if !adapterStateLoaded {
		adapterStateLoaded = true
		restoreAdapterState()
	}

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
		if installLocalPathProvisioner {
			if localPathProvisionerPath == "" {
				showErrorModal("Local-Path Provisioner path is empty.")
				return
			}
		}

		if installNfsProvisioner {
			err := nfsProvisionerConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
				return
			}
		}

		if installPrometheus {
			err := prometheusConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
				return
			}
		}

		if installLogging {
			err := loggingConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
				return
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

	app.SetFocus(flexList)
	app.SetFocus(listPackages)

	listPackages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlN {
			app.SetFocus(formPackage)
		}
		if event.Key() == tcell.KeyCtrlP {
			app.SetFocus(formDown)
		}
		return event
	})

	formPackage.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlN {
			app.SetFocus(formDown)
		}
		if event.Key() == tcell.KeyCtrlP {
			app.SetFocus(listPackages)
		}
		return event
	})

	formDown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlN {
			app.SetFocus(listPackages)
		}
		if event.Key() == tcell.KeyCtrlP {
			app.SetFocus(formPackage)
		}
		return event
	})
}

// restoreAdapterState repopulates the Packages form from the installed adapter.
// OM-Kits has no packages-env ConfigMap, so without this a re-run with
// Prometheus selected would silently disable alerting and uninstall the adapter.
func restoreAdapterState() {
	output, err := execCommand(
		`kubectl -n monitoring get secret alertmanager-webhook-adapter -o go-template='{{range $k, $v := .data}}{{$k}}={{$v | base64decode}}{{"\n"}}{{end}}' 2>/dev/null`, 0)
	if err == nil {
		creds := make(map[string]string)
		for _, line := range strings.Split(strings.TrimRight(string(output), "\n"), "\n") {
			if key, value, ok := strings.Cut(line, "="); ok {
				creds[key] = value
			}
		}
		prometheusConfig.dingTalkToken = creds["DINGTALK_TOKEN"]
		prometheusConfig.dingTalkSecret = creds["DINGTALK_SECRET"]
		prometheusConfig.dingTalkEnabled = creds["DINGTALK_TOKEN"] != "" && creds["DINGTALK_SECRET"] != ""
		prometheusConfig.feishuToken = creds["FEISHU_TOKEN"]
		prometheusConfig.feishuSecret = creds["FEISHU_SECRET"]
		prometheusConfig.feishuEnabled = creds["FEISHU_TOKEN"] != "" && creds["FEISHU_SECRET"] != ""
		prometheusConfig.weChatToken = creds["WEIXIN_TOKEN"]
		prometheusConfig.weChatEnabled = creds["WEIXIN_TOKEN"] != ""
	}

	output, err = execCommand(
		`kubectl -n monitoring get deployment alertmanager-webhook-adapter -o go-template='{{range .spec.template.spec.containers}}{{range .args}}{{println .}}{{end}}{{end}}' 2>/dev/null`, 0)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "--signature="):
			if signature := strings.TrimPrefix(line, "--signature="); signature != "" {
				prometheusConfig.adapterSignature = signature
			}
		case strings.HasPrefix(line, "--tmpl-lang="):
			if lang := strings.TrimPrefix(line, "--tmpl-lang="); lang == "zh" || lang == "en" {
				prometheusConfig.adapterLang = lang
			}
		}
	}
}

func selectPackage(index int, mainText string) {
	formPackage.Clear(true)
	app.SetFocus(listPackages)
	listPackages.SetItemText(index, mainText, "")
	switch mainText {
	case "Local-Path Provisioner":
		formPackage.AddCheckbox("Install Local-Path Provisioner: ", installLocalPathProvisioner, func(checked bool) {
			installLocalPathProvisioner = checked
			selectPackage(index, mainText)
		})
		if installLocalPathProvisioner {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Path: ", localPathProvisionerPath,
				0, nil, func(text string) {
					localPathProvisionerPath = strings.Trim(text, " ")
				})
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

			if prometheusConfig.grafanaRootUrl == "" {
				prometheusConfig.grafanaRootUrl = "https://" + basicInfo.host + "/grafana"
			}
			formPackage.AddInputField("Grafana root URL: ", prometheusConfig.grafanaRootUrl,
				0, nil, func(text string) {
					prometheusConfig.grafanaRootUrl = text
				})

			adapterLangOptions := []string{"zh", "en"}
			adapterLangIndex := slices.Index(adapterLangOptions, prometheusConfig.adapterLang)
			if adapterLangIndex < 0 {
				adapterLangIndex = 0
			}
			formPackage.AddDropDown("Alert message language: ", adapterLangOptions, adapterLangIndex,
				func(option string, optionIndex int) {
					prometheusConfig.adapterLang = option
				})
			formPackage.AddInputField("Alert signature: ", prometheusConfig.adapterSignature, 0, nil,
				func(text string) {
					prometheusConfig.adapterSignature = text
				})
			formPackage.AddCheckbox("Enable DingTalk notifications: ", prometheusConfig.dingTalkEnabled, func(checked bool) {
				prometheusConfig.dingTalkEnabled = checked
				selectPackage(index, mainText)
			})
			if prometheusConfig.dingTalkEnabled {
				formPackage.AddInputField("  DingTalk bot token: ", prometheusConfig.dingTalkToken, 0, nil,
					func(text string) {
						prometheusConfig.dingTalkToken = text
					})
				formPackage.AddInputField("  DingTalk bot secret: ", prometheusConfig.dingTalkSecret, 0, nil,
					func(text string) {
						prometheusConfig.dingTalkSecret = text
					})
			}
			formPackage.AddCheckbox("Enable Feishu notifications: ", prometheusConfig.feishuEnabled, func(checked bool) {
				prometheusConfig.feishuEnabled = checked
				selectPackage(index, mainText)
			})
			if prometheusConfig.feishuEnabled {
				formPackage.AddInputField("  Feishu bot token: ", prometheusConfig.feishuToken, 0, nil,
					func(text string) {
						prometheusConfig.feishuToken = text
					})
				formPackage.AddInputField("  Feishu bot secret: ", prometheusConfig.feishuSecret, 0, nil,
					func(text string) {
						prometheusConfig.feishuSecret = text
					})
			}
			formPackage.AddCheckbox("Enable WeChat (WeCom) notifications: ", prometheusConfig.weChatEnabled, func(checked bool) {
				prometheusConfig.weChatEnabled = checked
				selectPackage(index, mainText)
			})
			if prometheusConfig.weChatEnabled {
				formPackage.AddInputField("  WeChat bot key: ", prometheusConfig.weChatToken, 0, nil,
					func(text string) {
						prometheusConfig.weChatToken = text
					})
			}
		}
	case "Logging":
		formPackage.AddCheckbox("Install Logging: ", installLogging, func(checked bool) {
			installLogging = checked
			selectPackage(index, mainText)
		})
		if installLogging {
			listPackages.SetItemText(index, mainText, "Will install")

			formPackage.AddInputField("Collect logs from namespaces\n (comma separated, empty means all): ", loggingConfig.collectNamespaces,
				0, nil, func(text string) {
					loggingConfig.collectNamespaces = text
				})

			initialOption := slices.Index(storageClasses, loggingConfig.storageClass)
			formPackage.AddDropDown("Storage Class: ", storageClasses, initialOption, func(option string, optionIndex int) {
				loggingConfig.storageClass = option
			})
			formPackage.AddInputField("Elasticsearch storage size (Gi): ", strconv.Itoa(loggingConfig.esStorageSizeGi),
				0, nil, func(text string) {
					loggingConfig.esStorageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Lifecycle warm age (day): ", strconv.Itoa(loggingConfig.esWarmAgeDay),
				0, nil, func(text string) {
					loggingConfig.esWarmAgeDay, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Lifecycle cold age (day): ", strconv.Itoa(loggingConfig.esColdAgeDay),
				0, nil, func(text string) {
					loggingConfig.esColdAgeDay, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Lifecycle delete age (day): ", strconv.Itoa(loggingConfig.esDeleteAgeDay),
				0, nil, func(text string) {
					loggingConfig.esDeleteAgeDay, _ = strconv.Atoi(text)
				})
			formPackage.AddCheckbox("Node affinity: ", loggingConfig.nodeAffinity, func(checked bool) {
				loggingConfig.nodeAffinity = checked
			})
			formPackage.AddCheckbox("Send alert when ERROR level log detected: ", loggingConfig.errorLogAlert, func(checked bool) {
				loggingConfig.errorLogAlert = checked
			})
			formPackage.AddCheckbox("Enable monitoring: ", loggingConfig.enableMonitoring, func(checked bool) {
				loggingConfig.enableMonitoring = checked
			})

			resourcesPresets := []string{"nano", "micro", "small", "medium", "large", "xlarge", "2xlarge"}
			initialOption = slices.Index(resourcesPresets, "medium")
			formPackage.AddDropDown("Resources preset: ", resourcesPresets, initialOption, func(option string, optionIndex int) {
				loggingConfig.resourcesPreset = option
			})
		}
	case "Permission Manager":
		formPackage.AddCheckbox("Install Permission Manager: ", installPermissionManager, func(checked bool) {
			installPermissionManager = checked
			selectPackage(index, mainText)
		})
		if installPermissionManager {
			listPackages.SetItemText(index, mainText, "Will install")

			formPackage.AddInputField("Cluster name: ", permissionManagerConfig.clusterName,
				0, nil, func(text string) {
					permissionManagerConfig.clusterName = text
				})
			formPackage.AddInputField("Cluster address: ", permissionManagerConfig.clusterAddress,
				0, nil, func(text string) {
					permissionManagerConfig.clusterAddress = text
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
