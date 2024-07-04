package main

import (
	"errors"
	"github.com/rivo/tview"
	"strconv"
)

type JenkinsConfig struct {
	controllerStorageSizeGi int
	agentStorageSizeGi      int
	pipelineLibVersion      string
}

func (config *JenkinsConfig) validate() error {
	if config.controllerStorageSizeGi == 0 {
		return errors.New("Jenkins controller storage size is 0.")
	}
	if config.agentStorageSizeGi == 0 {
		return errors.New("Jenkins agent storage size is 0.")
	}
	if config.pipelineLibVersion == "" {
		return errors.New("Jenkins pipeline lib version is emtpy.")
	}
	return nil
}

type NexusConfig struct {
	storageSizeGi  int
	dockerNodePort string
}

func (config *NexusConfig) validate() error {
	if config.storageSizeGi == 0 {
		return errors.New("Nexus storage size is 0.")
	}
	if config.dockerNodePort == "" {
		return errors.New("Nexus docker node port is emtpy.")
	} else {
		portNum, err := strconv.Atoi(config.dockerNodePort)
		if err != nil || portNum < 30001 || portNum > 32767 {
			return errors.New("Nexus docker node port can only be in [30001 - 32767].")
		}
	}
	return nil
}

type SonarConfig struct {
	storageSizeGi   int
	dbStorageSizeGi int
}

func (config *SonarConfig) validate() error {
	if config.storageSizeGi == 0 {
		return errors.New("Sonarqube storage size is 0.")
	}
	if config.dbStorageSizeGi == 0 {
		return errors.New("Sonarqube DB storage size is 0.")
	}
	return nil
}

type FileServerConfig struct {
	storageSizeGi int
}

func (config *FileServerConfig) validate() error {
	if config.storageSizeGi == 0 {
		return errors.New("File server storage size is 0.")
	}
	return nil
}

type SmbConfig struct {
	nodePort string
}

func (config *SmbConfig) validate() error {
	if config.nodePort == "" {
		return errors.New("Smb node port is empty.")
	}
	return nil
}

type PrometheusConfig struct {
	alertmanagerStorageSizeGi int
	grafanaStorageSizeGi      int
	prometheusStorageSizeGi   int
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

type GiteaConfig struct {
	sshNodePort              string
	giteaSharedStorageSizeGi int
	giteaPgStorageSizeGi     int
}

func (config *GiteaConfig) validate() error {
	if config.sshNodePort == "" {
		return errors.New("SSH node port is empty.")
	}
	if config.giteaSharedStorageSizeGi == 0 {
		return errors.New("Gitea storage size is 0.")
	}
	if config.giteaPgStorageSizeGi == 0 {
		return errors.New("Gitea DB storage size is 0.")
	}
	return nil
}

type ZentaoConfig struct {
	zentaoStorageSizeGi   int
	zentaoDbStorageSizeGi int
}

func (config *ZentaoConfig) validate() error {
	if config.zentaoStorageSizeGi == 0 {
		return errors.New("Zentao storage size is 0.")
	}
	if config.zentaoDbStorageSizeGi == 0 {
		return errors.New("Zentao DB storage size is 0.")
	}
	return nil
}

type KeycloakConfig struct {
	dbStorageSizeGi int
}

func (config *KeycloakConfig) validate() error {
	if config.dbStorageSizeGi == 0 {
		return errors.New("Keycloak DB storage size is 0.")
	}
	return nil
}

var installJenkins = false
var installNexus = false
var installSonar = false
var installFileServer = false
var installSmb = false
var installPrometheus = false
var installGitea = false
var installZentao = false
var installKeycloak = false
var jenkinsConfig = JenkinsConfig{
	controllerStorageSizeGi: 20,
	agentStorageSizeGi:      200,
	pipelineLibVersion:      "latest",
}
var nexusConfig = NexusConfig{
	storageSizeGi:  200,
	dockerNodePort: "30001",
}
var sonarConfig = SonarConfig{
	storageSizeGi:   5,
	dbStorageSizeGi: 20,
}
var fileServerConfig = FileServerConfig{
	storageSizeGi: 200,
}
var smbConfig = SmbConfig{
	nodePort: "30445",
}
var prometheusConfig = PrometheusConfig{
	alertmanagerStorageSizeGi: 10,
	grafanaStorageSizeGi:      5,
	prometheusStorageSizeGi:   10,
}
var giteaConfig = GiteaConfig{
	sshNodePort:              "32222",
	giteaSharedStorageSizeGi: 200,
	giteaPgStorageSizeGi:     20,
}
var zentaoConfig = ZentaoConfig{
	zentaoStorageSizeGi:   200,
	zentaoDbStorageSizeGi: 20,
}

var keycloakConfig = KeycloakConfig{
	dbStorageSizeGi: 8,
}
var packages = []string{"Keycloak", "Gitea", "Zentao", "Jenkins", "Nexus", "File server", "Samba server", "Sonarqube", "Prometheus"}
var listPackages = tview.NewList()
var formPackage = tview.NewForm()

func initFlexPackages() {
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
		if installJenkins {
			err := jenkinsConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		if installNexus {
			err := nexusConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		if installSonar {
			err := sonarConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		if installFileServer {
			err := fileServerConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		if installSmb {
			err := smbConfig.validate()
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

		if installGitea {
			err := giteaConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		if installZentao {
			err := zentaoConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		if installKeycloak {
			err := keycloakConfig.validate()
			if err != nil {
				showErrorModal(err.Error())
			}
		}

		initFlexMirror()
		pages.SwitchToPage("Mirror")
	})

	formDown.AddButton("Back", func() {
		pages.SwitchToPage("Storage")
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
	case "Gitea":
		formPackage.AddCheckbox("Install Gitea: ", installGitea, func(checked bool) {
			installGitea = checked
			selectPackage(index, mainText)
		})
		if installGitea {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Gitea shared storage size (Gi): ", strconv.Itoa(giteaConfig.giteaSharedStorageSizeGi),
				0, nil, func(text string) {
					giteaConfig.giteaSharedStorageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Gitea DB storage size (Gi): ", strconv.Itoa(giteaConfig.giteaPgStorageSizeGi),
				0, nil, func(text string) {
					giteaConfig.giteaPgStorageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Gitea SSH node port: ", giteaConfig.sshNodePort,
				0, nil, func(text string) {
					giteaConfig.sshNodePort = text
				})
		}
	case "Zentao":
		formPackage.AddCheckbox("Install Zentao: ", installZentao, func(checked bool) {
			installZentao = checked
			selectPackage(index, mainText)
		})
		if installZentao {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Zentao storage size (Gi): ", strconv.Itoa(zentaoConfig.zentaoStorageSizeGi),
				0, nil, func(text string) {
					zentaoConfig.zentaoStorageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Zentao DB storage size (Gi): ", strconv.Itoa(zentaoConfig.zentaoDbStorageSizeGi),
				0, nil, func(text string) {
					zentaoConfig.zentaoDbStorageSizeGi, _ = strconv.Atoi(text)
				})
		}
	case "Jenkins":
		formPackage.AddCheckbox("Install Jenkins: ", installJenkins, func(checked bool) {
			installJenkins = checked
			selectPackage(index, mainText)
		})
		if installJenkins {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Jenkins controller storage size (Gi): ", strconv.Itoa(jenkinsConfig.controllerStorageSizeGi),
				0, nil, func(text string) {
					jenkinsConfig.controllerStorageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Jenkins agent storage size (Gi): ", strconv.Itoa(jenkinsConfig.agentStorageSizeGi),
				0, nil, func(text string) {
					jenkinsConfig.agentStorageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Jenkins pipeline lib version: ", jenkinsConfig.pipelineLibVersion,
				0, nil, func(text string) {
					jenkinsConfig.pipelineLibVersion = text
				})
		}
	case "Nexus":
		formPackage.AddCheckbox("Install Nexus: ", installNexus, func(checked bool) {
			installNexus = checked
			selectPackage(index, mainText)
		})
		if installNexus {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Nexus storage size (Gi): ", strconv.Itoa(nexusConfig.storageSizeGi),
				0, nil, func(text string) {
					nexusConfig.storageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Nexus docker node port: ", nexusConfig.dockerNodePort,
				0, nil, func(text string) {
					nexusConfig.dockerNodePort = text
				})
		}
	case "Sonarqube":
		formPackage.AddCheckbox("Install Sonarqube: ", installSonar, func(checked bool) {
			installSonar = checked
			selectPackage(index, mainText)
		})
		if installSonar {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Sonarqube storage size (Gi): ", strconv.Itoa(sonarConfig.storageSizeGi),
				0, nil, func(text string) {
					sonarConfig.storageSizeGi, _ = strconv.Atoi(text)
				})
			formPackage.AddInputField("Sonarqube DB storage size (Gi): ", strconv.Itoa(sonarConfig.dbStorageSizeGi),
				0, nil, func(text string) {
					sonarConfig.dbStorageSizeGi, _ = strconv.Atoi(text)
				})
		}
	case "File server":
		formPackage.AddCheckbox("Install file server: ", installFileServer, func(checked bool) {
			installFileServer = checked
			selectPackage(index, mainText)
		})
		if installFileServer {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("File server storage size (Gi): ", strconv.Itoa(fileServerConfig.storageSizeGi),
				0, nil, func(text string) {
					fileServerConfig.storageSizeGi, _ = strconv.Atoi(text)
				})
		}
	case "Samba server":
		formPackage.AddCheckbox("Install Samba server: ", installSmb, func(checked bool) {
			installSmb = checked
			selectPackage(index, mainText)
		})
		if installSmb {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Smb node port: ", smbConfig.nodePort,
				0, nil, func(text string) {
					smbConfig.nodePort = text
				})
		}
	case "Prometheus":
		formPackage.AddCheckbox("Install Prometheus: ", installPrometheus, func(checked bool) {
			installPrometheus = checked
			selectPackage(index, mainText)
		})
		if installPrometheus {
			listPackages.SetItemText(index, mainText, "Will install")
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
	case "Keycloak":
		formPackage.AddCheckbox("Install Keycloak: ", installKeycloak, func(checked bool) {
			installKeycloak = checked
			selectPackage(index, mainText)
		})
		if installKeycloak {
			listPackages.SetItemText(index, mainText, "Will install")
			formPackage.AddInputField("Keycloak DB storage size (Gi): ", strconv.Itoa(keycloakConfig.dbStorageSizeGi),
				0, nil, func(text string) {
					keycloakConfig.dbStorageSizeGi, _ = strconv.Atoi(text)
				})
		}
	}
}
