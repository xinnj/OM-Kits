package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

type task struct {
	name    string
	command string
}

var flexTop = tview.NewFlex()
var listTask *tview.List
var process *os.Process
var processState *os.ProcessState
var abortButton *tview.Button
var backButton *tview.Button
var quitButton *tview.Button
var logContent *tview.TextView
var stopTimer = make(chan bool)

func initFlexInstall() {
	flexInstall.Clear()
	flexTop.Clear()
	flexTop.SetTitle("Install").SetBorder(true)

	tasks, envs := buildTasks()

	listTask = tview.NewList()
	for index, task := range tasks {
		listTask.AddItem(task.name, "pending", rune(97+index), nil)
	}
	// Disable mouse
	listTask.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		return action, nil
	})

	process = nil
	processState = nil

	logContent = tview.NewTextView()
	logContent.SetBackgroundColor(tcell.ColorDarkBlue)
	logContent.SetMaxLines(0).
		SetWrap(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			logContent.ScrollToEnd()
			app.Draw()
		})

	flexTop.
		AddItem(listTask, 0, 1, false).
		AddItem(logContent, 0, 3, false)

	formDown := tview.NewForm()
	formDown.AddButton("Abort", func() {
		if process != nil && processState == nil {
			confirmAbort := tview.NewModal().
				SetText("Do you want to abort the execution?").
				AddButtons([]string{"Abort", "Cancel"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Cancel" {
						pages.SwitchToPage("Install")
					}
					if buttonLabel == "Abort" {
						pgid, err := syscall.Getpgid(process.Pid)
						check(err)
						syscall.Kill(-pgid, 15)

						abortButton.SetDisabled(true)
						backButton.SetDisabled(false)
						quitButton.SetDisabled(false)

						pages.SwitchToPage("Install")
					}
				})
			pages.AddPage("Confirm Abort", confirmAbort, true, true)
		}
	})
	abortButton = formDown.GetButton(formDown.GetButtonIndex("Abort"))
	abortButton.SetDisabled(true)

	formDown.AddButton("Back", func() {
		pages.SwitchToPage("Mirror")
	})
	backButton = formDown.GetButton(formDown.GetButtonIndex("Back"))
	backButton.SetDisabled(false)

	formDown.AddButton("Quit", func() {
		showQuitModal()
	})
	quitButton = formDown.GetButton(formDown.GetButtonIndex("Quit"))
	quitButton.SetDisabled(false)

	flexInstall.SetDirection(tview.FlexRow).
		AddItem(flexTop, 0, 1, true).
		AddItem(formDown, 3, 1, false)

	go startTimer(stopTimer)
	go execTasks(tasks, envs, logContent)
}

func buildTasks() (tasks []task, envs []string) {
	envs = append(envs, "IDO_TEAM="+basicInfo.team)
	envs = append(envs, "IDO_TIMEZONE="+basicInfo.timezone)
	envs = append(envs, "IDO_CLUSTER_HOSTNAME="+basicInfo.host)

	if net.ParseIP(basicInfo.host) == nil {
		envs = append(envs, "IDO_INGRESS_HOSTNAME="+basicInfo.host)
	} else {
		envs = append(envs, "IDO_INGRESS_HOSTNAME=")
	}

	var clusterUrl string
	if basicInfo.httpsEnabled {
		clusterUrl = "https://" + basicInfo.host
		envs = append(envs, "IDO_CLUSTER_URL="+clusterUrl)
		envs = append(envs, "IDO_TLS_KEY=tls")
		envs = append(envs, "IDO_TLS_HOST="+basicInfo.host)

		switch basicInfo.tlsCert.certMethod {
		case certMethod.defaultTlsSecret:
			envs = append(envs, "IDO_TLS_ACME=false")
			envs = append(envs, "IDO_TLS_SECRET=")
		case certMethod.existingTlsSecret:
			envs = append(envs, "IDO_TLS_ACME=false")
			envs = append(envs, "IDO_TLS_SECRET="+basicInfo.tlsCert.existingCertSecret)
		case certMethod.certManager:
			envs = append(envs, "IDO_TLS_ACME=true")
			envs = append(envs, "IDO_TLS_SECRET="+basicInfo.host)
		}
	} else {
		clusterUrl = "http://" + basicInfo.host
		envs = append(envs, "IDO_CLUSTER_URL="+clusterUrl)
		envs = append(envs, "IDO_TLS_KEY=tls-disabled")
		envs = append(envs, "IDO_TLS_HOST=")
		envs = append(envs, "IDO_TLS_ACME=false")
		envs = append(envs, "IDO_TLS_SECRET=")
	}
	envs = append(envs, "IDO_FORCE_SSL_REDIRECT="+strconv.FormatBool(basicInfo.tlsCert.forceSslRedirect))

	if basicInfo.team == "default" {
		envs = append(envs, "IDO_TEAM_URL="+clusterUrl)
	} else {
		envs = append(envs, "IDO_TEAM_URL="+clusterUrl+"/"+basicInfo.team)
	}

	var finalMirrors map[string]string
	if enableMirror {
		finalMirrors = map[string]string{
			"IDO_DOCKER_CONTAINER_MIRROR": mirrors["DOCKER_CONTAINER_MIRROR"],
			"IDO_QUAY_CONTAINER_MIRROR":   mirrors["QUAY_CONTAINER_MIRROR"],
			"IDO_K8S_CONTAINER_MIRROR":    mirrors["K8S_CONTAINER_MIRROR"],
			"IDO_GCR_CONTAINER_MIRROR":    mirrors["GCR_CONTAINER_MIRROR"],
		}
	} else {
		finalMirrors = map[string]string{
			"IDO_DOCKER_CONTAINER_MIRROR": "docker.io",
			"IDO_QUAY_CONTAINER_MIRROR":   "quay.io",
			"IDO_K8S_CONTAINER_MIRROR":    "k8s.io",
			"IDO_GCR_CONTAINER_MIRROR":    "k8s-gcr.io",
		}
	}
	for k, v := range finalMirrors {
		envs = append(envs, k+"="+v)
	}

	tasks = append(tasks, task{name: "Populate Configuration",
		command: "chmod +x packages/populate-configuration.sh; packages/populate-configuration.sh"})

	if basicInfo.httpsEnabled && basicInfo.tlsCert.certMethod == certMethod.certManager {
		tasks = append(tasks, task{name: "Install Cert-manager",
			command: "chmod +x packages/cert-manager/install.sh; packages/cert-manager/install.sh"})
		envs = append(envs, "IDO_ACME_EMAIL="+basicInfo.tlsCert.acmeEmail)
	}

	if installPrometheus {
		tasks = append(tasks, task{name: "Install Prometheus",
			command: "chmod +x packages/prometheus/install.sh; packages/prometheus/install.sh"})
		envs = append(envs, "IDO_ALTERMANAGER_STORAGE_SIZE="+strconv.Itoa(prometheusConfig.alertmanagerStorageSizeGi)+"Gi")
		envs = append(envs, "IDO_GRAFANA_STORAGE_SIZE="+strconv.Itoa(prometheusConfig.grafanaStorageSizeGi)+"Gi")
		envs = append(envs, "IDO_PROMETHEUS_STORAGE_SIZE="+strconv.Itoa(prometheusConfig.prometheusStorageSizeGi)+"Gi")
		envs = append(envs, "IDO_ENABLE_PROMETHEUS=true")
	} else {
		_, err := execCommand("helm -n monitoring status prometheus", 0)
		if err == nil {
			envs = append(envs, "IDO_ENABLE_PROMETHEUS=true")
		} else {
			envs = append(envs, "IDO_ENABLE_PROMETHEUS=false")
		}
	}

	if !useExistingSC {
		switch storageClass {
		case storageClassType.ceph:
			tasks = append(tasks, task{name: "Install Ceph",
				command: "chmod +x packages/storage/ceph/install.sh; packages/storage/ceph/install.sh"})
		case storageClassType.nfs:
			tasks = append(tasks, task{name: "Install nfs",
				command: "chmod +x packages/storage/nfs/install.sh; packages/storage/nfs/install.sh"})
			envs = append(envs, "IDO_NFS_SERVER="+nfsConfig.server)
			envs = append(envs, "IDO_NFS_PATH="+nfsConfig.path)
		}
	}
	envs = append(envs, "IDO_STORAGE_CLASS="+storageClass)

	if installKeycloak {
		tasks = append(tasks, task{name: "Install Keycloak",
			command: "chmod +x packages/keycloak/install.sh; packages/keycloak/install.sh"})
		envs = append(envs, "IDO_KEYCLOAK_PG_STORAGE_SIZE="+strconv.Itoa(keycloakConfig.dbStorageSizeGi)+"Gi")
	}

	if installGitea {
		tasks = append(tasks, task{name: "Install Gitea",
			command: "chmod +x packages/gitea/install.sh; packages/gitea/install.sh"})
		envs = append(envs, "IDO_GITEA_SHARED_STORAGE_SIZE="+strconv.Itoa(giteaConfig.giteaSharedStorageSizeGi)+"Gi")
		envs = append(envs, "IDO_GITEA_PG_STORAGE_SIZE="+strconv.Itoa(giteaConfig.giteaPgStorageSizeGi)+"Gi")
		envs = append(envs, "IDO_GITEA_SSH_NODE_PORT="+giteaConfig.sshNodePort)
	}

	if installZentao {
		tasks = append(tasks, task{name: "Install Zentao",
			command: "chmod +x packages/zentao/install.sh; packages/zentao/install.sh"})
		envs = append(envs, "IDO_ZENTAO_STORAGE_SIZE="+strconv.Itoa(zentaoConfig.zentaoStorageSizeGi)+"Gi")
		envs = append(envs, "IDO_ZENTAO_DB_STORAGE_SIZE="+strconv.Itoa(zentaoConfig.zentaoDbStorageSizeGi)+"Gi")
	}

	if installJenkins {
		tasks = append(tasks, task{name: "Install Jenkins",
			command: "chmod +x packages/jenkins/install.sh; packages/jenkins/install.sh"})
		envs = append(envs, "IDO_JENKINS_CONTROLLER_STORAGE_SIZE="+strconv.Itoa(jenkinsConfig.controllerStorageSizeGi)+"Gi")
		envs = append(envs, "IDO_JENKINS_AGENT_STORAGE_SIZE="+strconv.Itoa(jenkinsConfig.agentStorageSizeGi)+"Gi")
		envs = append(envs, "IDO_JENKINS_LIB_VERSION="+jenkinsConfig.pipelineLibVersion)
	}

	if installNexus {
		tasks = append(tasks, task{name: "Install Nexus",
			command: "chmod +x packages/nexus/install.sh; packages/nexus/install.sh"})
		envs = append(envs, "IDO_NEXUS_STORAGE_SIZE="+strconv.Itoa(nexusConfig.storageSizeGi)+"Gi")
		envs = append(envs, "IDO_NEXUS_DOCKER_NODE_PORT="+nexusConfig.dockerNodePort)
	}

	if installFileServer {
		tasks = append(tasks, task{name: "Install File Server",
			command: "chmod +x packages/file-server/install.sh; packages/file-server/install.sh"})
		envs = append(envs, "IDO_FILE_STORAGE_SIZE="+strconv.Itoa(fileServerConfig.storageSizeGi)+"Gi")
	}

	if installSmb {
		tasks = append(tasks, task{name: "Install Samba Server",
			command: "chmod +x packages/samba-server/install.sh; packages/samba-server/install.sh"})
		envs = append(envs, "IDO_SMB_NODE_PORT="+smbConfig.nodePort)
	}

	if installSonar {
		tasks = append(tasks, task{name: "Install Sonarqube",
			command: "chmod +x packages/sonar/install.sh; packages/sonar/install.sh"})
		envs = append(envs, "IDO_SONAR_STORAGE_SIZE="+strconv.Itoa(sonarConfig.storageSizeGi)+"Gi")
		envs = append(envs, "IDO_SONAR_PG_STORAGE_SIZE="+strconv.Itoa(sonarConfig.dbStorageSizeGi)+"Gi")
	}

	// backup
	if enableBackup {
		tasks = append(tasks, task{name: "Install Velero",
			command: "chmod +x packages/velero/install.sh; packages/velero/install.sh"})

		envs = append(envs, "IDO_BACKUP_PROVIDER="+backupInfo.provider)
		envs = append(envs, "IDO_BACKUP_BUCKET="+backupInfo.bucket)
		envs = append(envs, "IDO_BACKUP_LOCATION_CONFIG="+backupInfo.locationConfig)
		envs = append(envs, "IDO_BACKUP_REGION="+backupInfo.region)
		envs = append(envs, "IDO_BACKUP_CLOUD_SECRET="+backupInfo.cloudSecret)
		envs = append(envs, "IDO_BACKUP_SCHEDULE="+backupInfo.schedule)
		envs = append(envs, "IDO_BACKUP_TTL="+strconv.Itoa(backupInfo.ttl)+"h")

		backupSelectors := ""
		for key, value := range backupInfo.backupItems {
			if value {
				backupSelectors = backupSelectors + "        - matchLabels:\n            velero.io/backup-app: " + key + "\n"
			}
		}
		envs = append(envs, "IDO_BACKUP_SELECTORS="+backupSelectors)
	}

	tasks = append(tasks, task{name: "Final Check",
		command: "chmod +x packages/final-check.sh; packages/final-check.sh"})

	return
}

func execTasks(tasks []task, envs []string, view *tview.TextView) {
	var logBgColor tcell.Color

	for index, task := range tasks {
		listTask.SetCurrentItem(index)
		mainText, _ := listTask.GetItemText(index)
		listTask.SetItemText(index, mainText, "in-progress...")

		processState = nil

		//cmd := exec.Command("/bin/bash", "-c", "echo '"+command+"'")
		//cmd := exec.Command("/bin/bash", "-c", "export")
		cmd := exec.Command("/bin/bash", "-c", task.command)

		cmd.Dir = appPath

		cmd.Env = os.Environ()
		for _, env := range envs {
			cmd.Env = append(cmd.Env, env)
		}

		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		stdout, err := cmd.StdoutPipe()
		check(err)
		stderr, err := cmd.StderrPipe()
		check(err)

		err = cmd.Start()
		check(err)
		process = cmd.Process

		abortButton.SetDisabled(false)
		backButton.SetDisabled(true)
		quitButton.SetDisabled(true)

		_, err = io.Copy(view, stdout)
		check(err)

		errBytes, err := io.ReadAll(stderr)
		check(err)

		err = cmd.Wait()
		processState = cmd.ProcessState
		process = nil

		if err != nil {
			view.SetText(view.GetText(false) + "\n" + string(errBytes))
			listTask.SetItemText(index, mainText, "failed!")
			logBgColor = tcell.ColorDarkRed
			break
		} else {
			listTask.SetItemText(index, mainText, "done")
			logBgColor = tcell.ColorDarkGreen
		}
	}

	stopTimer <- true

	app.QueueUpdateDraw(func() {
		logContent.SetBackgroundColor(logBgColor)
		abortButton.SetDisabled(true)
		backButton.SetDisabled(false)
		quitButton.SetDisabled(false)
	})
}

func startTimer(stop chan bool) {
	startTime := time.Now()
	for {
		select {
		case <-stop:
			return
		default:
			app.QueueUpdateDraw(func() {
				flexTop.SetTitle("Install - Time Elapsed: " + time.Since(startTime).Round(time.Second).String())
			})
			time.Sleep(time.Second)
		}
	}
}
