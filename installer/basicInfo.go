package main

import (
	"github.com/rivo/tview"
	"github.com/thlib/go-timezone-local/tzlocal"
	"golang.org/x/exp/slices"
	"net"
	"net/mail"
	"strings"
)

type BasicInfo struct {
	host         string
	httpsEnabled bool
	timezone     string
	tlsCert      TlsCert
}
type TlsCert struct {
	certMethod       string
	forceSslRedirect bool
	acmeEmail        string
}
type CertMethod struct {
	defaultTlsSecret string
	certManager      string
}

var certMethod = CertMethod{
	defaultTlsSecret: "Default TLS Secret (Secret name: default-tls, Namespace: default)",
	certManager:      "Cert Manager",
}

var basicInfo = BasicInfo{host: "", httpsEnabled: false, timezone: "", tlsCert: TlsCert{
	certMethod:       "",
	forceSslRedirect: false,
	acmeEmail:        "",
}}

func initFlexBasicInfo() {
	flexBasicInfo.Clear()
	formBasicInfo := tview.NewForm()
	formBasicInfo.SetTitle("Basic Info").SetBorder(true)

	if basicInfo.timezone == "" {
		var err error
		basicInfo.timezone, err = tzlocal.RuntimeTZ()
		check(err)
	}

	formBasicInfo.AddInputField("Timezone: ", basicInfo.timezone, 0, nil, func(text string) {
		basicInfo.timezone = text
	})

	formBasicInfo.AddInputField("Cluster DNS or IP: ", basicInfo.host, 0, nil,
		func(text string) {
			basicInfo.host = strings.Trim(text, " ")
		})

	formBasicInfo.AddCheckbox("Enable https: ", basicInfo.httpsEnabled, func(checked bool) {
		basicInfo.httpsEnabled = checked
		initFlexBasicInfo()
	})

	if basicInfo.httpsEnabled {
		formBasicInfo.AddCheckbox("  Force SSL redirect: ", basicInfo.tlsCert.forceSslRedirect, func(checked bool) {
			basicInfo.tlsCert.forceSslRedirect = checked
		})

		arrCertMethods := []string{certMethod.defaultTlsSecret, certMethod.certManager}
		initialOption := slices.Index(arrCertMethods, basicInfo.tlsCert.certMethod)
		formBasicInfo.AddDropDown("  Select a method to generate SSL certificate: ", arrCertMethods, initialOption,
			func(option string, optionIndex int) {
				if basicInfo.tlsCert.certMethod != option {
					basicInfo.tlsCert.certMethod = option
					initFlexBasicInfo()
				}
			})

		if basicInfo.tlsCert.certMethod == certMethod.certManager {
			formBasicInfo.AddInputField("    Email: ", basicInfo.tlsCert.acmeEmail, 0, nil,
				func(text string) {
					basicInfo.tlsCert.acmeEmail = strings.Trim(text, " ")
				})
		}
	}

	formDown := tview.NewForm()

	formDown.AddButton("Next", func() {
		if basicInfo.host == "" {
			showErrorModal("Custer domain name or IP is empty.")
			return
		}

		if basicInfo.timezone == "" {
			showErrorModal("Timezone is empty.")
			return
		}

		if basicInfo.httpsEnabled {
			if net.ParseIP(basicInfo.host) != nil {
				showErrorModal(basicInfo.host + " must be a DNS, not an IP address, when https is enabled.")
				return
			}

			if basicInfo.tlsCert.certMethod == "" {
				showErrorModal("Please select a method to generate SSL certificate.")
				return
			}

			if basicInfo.tlsCert.certMethod == certMethod.defaultTlsSecret {
				_, err := execCommand("kubectl get secret default-tls", 0)
				if err != nil {
					showErrorModal("Secret 'default-tls' not existing.")
					return
				}
			}

			if basicInfo.tlsCert.certMethod == certMethod.certManager {
				email, err := mail.ParseAddress(basicInfo.tlsCert.acmeEmail)
				if err != nil {
					showErrorModal("Email is empty or format is wrong.")
					return
				} else {
					basicInfo.tlsCert.acmeEmail = email.Address
				}
			}
		}

		initFlexPackages()
		pages.SwitchToPage("Packages")
	})

	formDown.AddButton("Quit", func() {
		showQuitModal()
	})

	flexBasicInfo.SetDirection(tview.FlexRow).
		AddItem(formBasicInfo, 0, 1, true).
		AddItem(formDown, 3, 1, false)
}
