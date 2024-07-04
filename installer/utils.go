package main

import (
	"context"
	"github.com/rivo/tview"
	"os"
	"os/exec"
	"time"
)

func check(e error) {
	if e != nil {
		app.Stop()
		panic(e)
	}
}

func execCommand(cmdString string, timeout int, envs ...string) ([]byte, error) {
	var cmd *exec.Cmd

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", cmdString)
	} else {
		cmd = exec.Command("/bin/sh", "-c", cmdString)
	}

	cmd.Env = os.Environ()
	for _, env := range envs {
		cmd.Env = append(cmd.Env, env)
	}

	output, err := cmd.CombinedOutput()

	return output, err
}

func showErrorModal(text string) {
	modalError := tview.NewModal()
	currentPage, _ := pages.GetFrontPage()
	modalError.SetText(text).AddButtons([]string{"OK"}).SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		pages.SwitchToPage(currentPage)
	})
	pages.AddPage("Error", modalError, true, true)
}

func showQuitModal() {
	currentPage, _ := pages.GetFrontPage()

	modalQuit.ClearButtons()
	modalQuit.SetText("Do you want to quit the application?").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Cancel" {
				pages.SwitchToPage(currentPage)
			}
			if buttonLabel == "Quit" {
				app.Stop()
			}
		})
	pages.SwitchToPage("Quit")
}

//func initLog(prefix string) {
//	now := time.Now()
//	suffix := fmt.Sprintf("%d%02d%02dT%02d%02d%02d",
//		now.Year(), now.Month(), now.Day(),
//		now.Hour(), now.Minute(), now.Second())
//	logFilePath = filepath.Join(projectPath, prefix+suffix+".log")
//}
