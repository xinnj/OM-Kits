package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	stateConfigMapNamespace = "kube-system"
	stateConfigMapName      = "om-kits-state"
	stateConfigMapDataKey   = "state"
	stateSchemaVersion      = 1
	stateCommandTimeoutSec  = 30

	stateLoadCommand = "kubectl -n " + stateConfigMapNamespace + " get configmap " + stateConfigMapName + " --ignore-not-found -o json"
	stateSaveCommand = "kubectl apply -f -"
)

var (
	// stateCommand is swappable in tests so state handling can be verified
	// without a cluster.
	stateCommand = execCommandWithInput

	// persistedStateLoaded records that answers were restored from the cluster.
	// Credentials still come from the adapter Secret, but saved non-secret
	// answers must not be overwritten by a stale adapter Deployment.
	persistedStateLoaded bool
)

const (
	tlsMethodKeyDefaultSecret = "default-tls-secret"
	tlsMethodKeyCertManager   = "cert-manager"
)

// wizardState is the cluster-side representation of the wizard's answers. It
// is a separate shape from the in-memory config structs so the JSON keys stay
// stable even when internal field names change. Credentials are deliberately
// absent: they are stored only in the adapter Secret.
type wizardState struct {
	SchemaVersion int            `json:"schemaVersion"`
	BasicInfo     basicInfoState `json:"basicInfo"`
	Packages      packagesState  `json:"packages"`
	Mirror        mirrorState    `json:"mirror"`
}

type basicInfoState struct {
	Host         string       `json:"host"`
	HTTPSEnabled bool         `json:"httpsEnabled"`
	Timezone     string       `json:"timezone"`
	TLSCert      tlsCertState `json:"tlsCert"`
}

type tlsCertState struct {
	CertMethod       string `json:"certMethod"`
	ForceSslRedirect bool   `json:"forceSslRedirect"`
	ACMEEmail        string `json:"acmeEmail"`
}

type packagesState struct {
	LocalPathProvisioner localPathProvisionerState `json:"localPathProvisioner"`
	NFSProvisioner       nfsProvisionerState       `json:"nfsProvisioner"`
	Prometheus           prometheusState           `json:"prometheus"`
	Logging              loggingState              `json:"logging"`
	PermissionManager    permissionManagerState    `json:"permissionManager"`
}

type localPathProvisionerState struct {
	Path string `json:"path"`
}

type nfsProvisionerState struct {
	Server       string `json:"server"`
	Path         string `json:"path"`
	MountOptions string `json:"mountOptions"`
}

type prometheusState struct {
	AlertmanagerStorageSizeGi int    `json:"alertmanagerStorageSizeGi"`
	GrafanaStorageSizeGi      int    `json:"grafanaStorageSizeGi"`
	PrometheusStorageSizeGi   int    `json:"prometheusStorageSizeGi"`
	StorageClass              string `json:"storageClass"`
	GrafanaRootURL            string `json:"grafanaRootUrl"`
	AdapterLang               string `json:"adapterLang"`
	AdapterSignature          string `json:"adapterSignature"`
	DingTalkEnabled           bool   `json:"dingTalkEnabled"`
	FeishuEnabled             bool   `json:"feishuEnabled"`
	WeChatEnabled             bool   `json:"weChatEnabled"`
}

type loggingState struct {
	CollectNamespaces string `json:"collectNamespaces"`
	StorageClass      string `json:"storageClass"`
	ESStorageSizeGi   int    `json:"esStorageSizeGi"`
	ESDeleteAgeDay    int    `json:"esDeleteAgeDay"`
	ESColdAgeDay      int    `json:"esColdAgeDay"`
	ESWarmAgeDay      int    `json:"esWarmAgeDay"`
	NodeAffinity      bool   `json:"nodeAffinity"`
	ErrorLogAlert     bool   `json:"errorLogAlert"`
	EnableMonitoring  bool   `json:"enableMonitoring"`
	ResourcesPreset   string `json:"resourcesPreset"`
}

type permissionManagerState struct {
	ClusterName    string `json:"clusterName"`
	ClusterAddress string `json:"clusterAddress"`
}

type mirrorState struct {
	Enabled      bool              `json:"enabled"`
	UseOneMirror bool              `json:"useOneMirror"`
	OneMirror    string            `json:"oneMirror"`
	PerRegistry  map[string]string `json:"perRegistry"`
}

type configMapManifest struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   configMapMetadata `json:"metadata"`
	Data       map[string]string `json:"data"`
}

type configMapMetadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// captureState snapshots the current answers into the shape written to the
// cluster. Credentials are never part of this shape.
func captureState() wizardState {
	return wizardState{
		SchemaVersion: stateSchemaVersion,
		BasicInfo: basicInfoState{
			Host:         basicInfo.host,
			HTTPSEnabled: basicInfo.httpsEnabled,
			Timezone:     basicInfo.timezone,
			TLSCert: tlsCertState{
				CertMethod:       tlsMethodKey(basicInfo.tlsCert.certMethod),
				ForceSslRedirect: basicInfo.tlsCert.forceSslRedirect,
				ACMEEmail:        basicInfo.tlsCert.acmeEmail,
			},
		},
		Packages: packagesState{
			LocalPathProvisioner: localPathProvisionerState{
				Path: localPathProvisionerPath,
			},
			NFSProvisioner: nfsProvisionerState{
				Server:       nfsProvisionerConfig.server,
				Path:         nfsProvisionerConfig.path,
				MountOptions: nfsProvisionerConfig.mountOptions,
			},
			Prometheus: prometheusState{
				AlertmanagerStorageSizeGi: prometheusConfig.alertmanagerStorageSizeGi,
				GrafanaStorageSizeGi:      prometheusConfig.grafanaStorageSizeGi,
				PrometheusStorageSizeGi:   prometheusConfig.prometheusStorageSizeGi,
				StorageClass:              prometheusConfig.storageClass,
				GrafanaRootURL:            prometheusConfig.grafanaRootUrl,
				AdapterLang:               prometheusConfig.adapterLang,
				AdapterSignature:          prometheusConfig.adapterSignature,
				DingTalkEnabled:           prometheusConfig.dingTalkEnabled,
				FeishuEnabled:             prometheusConfig.feishuEnabled,
				WeChatEnabled:             prometheusConfig.weChatEnabled,
			},
			Logging: loggingState{
				CollectNamespaces: loggingConfig.collectNamespaces,
				StorageClass:      loggingConfig.storageClass,
				ESStorageSizeGi:   loggingConfig.esStorageSizeGi,
				ESDeleteAgeDay:    loggingConfig.esDeleteAgeDay,
				ESColdAgeDay:      loggingConfig.esColdAgeDay,
				ESWarmAgeDay:      loggingConfig.esWarmAgeDay,
				NodeAffinity:      loggingConfig.nodeAffinity,
				ErrorLogAlert:     loggingConfig.errorLogAlert,
				EnableMonitoring:  loggingConfig.enableMonitoring,
				ResourcesPreset:   loggingConfig.resourcesPreset,
			},
			PermissionManager: permissionManagerState{
				ClusterName:    permissionManagerConfig.clusterName,
				ClusterAddress: permissionManagerConfig.clusterAddress,
			},
		},
		Mirror: mirrorState{
			Enabled:      enableMirror,
			UseOneMirror: useOneMirror,
			OneMirror:    oneMirror,
			PerRegistry:  cloneStringMap(mirrors),
		},
	}
}

// applyState writes loaded answers back into the globals the pages read. It
// never touches credentials.
func applyState(state wizardState) {
	basicInfo.host = state.BasicInfo.Host
	basicInfo.httpsEnabled = state.BasicInfo.HTTPSEnabled
	basicInfo.timezone = state.BasicInfo.Timezone
	basicInfo.tlsCert.certMethod = tlsMethodFromKey(state.BasicInfo.TLSCert.CertMethod)
	basicInfo.tlsCert.forceSslRedirect = state.BasicInfo.TLSCert.ForceSslRedirect
	basicInfo.tlsCert.acmeEmail = state.BasicInfo.TLSCert.ACMEEmail

	localPathProvisionerPath = state.Packages.LocalPathProvisioner.Path

	nfsProvisionerConfig.server = state.Packages.NFSProvisioner.Server
	nfsProvisionerConfig.path = state.Packages.NFSProvisioner.Path
	nfsProvisionerConfig.mountOptions = state.Packages.NFSProvisioner.MountOptions

	prometheusConfig.alertmanagerStorageSizeGi = state.Packages.Prometheus.AlertmanagerStorageSizeGi
	prometheusConfig.grafanaStorageSizeGi = state.Packages.Prometheus.GrafanaStorageSizeGi
	prometheusConfig.prometheusStorageSizeGi = state.Packages.Prometheus.PrometheusStorageSizeGi
	prometheusConfig.storageClass = state.Packages.Prometheus.StorageClass
	prometheusConfig.grafanaRootUrl = state.Packages.Prometheus.GrafanaRootURL
	prometheusConfig.adapterLang = state.Packages.Prometheus.AdapterLang
	prometheusConfig.adapterSignature = state.Packages.Prometheus.AdapterSignature
	prometheusConfig.dingTalkEnabled = state.Packages.Prometheus.DingTalkEnabled
	prometheusConfig.feishuEnabled = state.Packages.Prometheus.FeishuEnabled
	prometheusConfig.weChatEnabled = state.Packages.Prometheus.WeChatEnabled

	loggingConfig.collectNamespaces = state.Packages.Logging.CollectNamespaces
	loggingConfig.storageClass = state.Packages.Logging.StorageClass
	loggingConfig.esStorageSizeGi = state.Packages.Logging.ESStorageSizeGi
	loggingConfig.esDeleteAgeDay = state.Packages.Logging.ESDeleteAgeDay
	loggingConfig.esColdAgeDay = state.Packages.Logging.ESColdAgeDay
	loggingConfig.esWarmAgeDay = state.Packages.Logging.ESWarmAgeDay
	loggingConfig.nodeAffinity = state.Packages.Logging.NodeAffinity
	loggingConfig.errorLogAlert = state.Packages.Logging.ErrorLogAlert
	loggingConfig.enableMonitoring = state.Packages.Logging.EnableMonitoring
	loggingConfig.resourcesPreset = state.Packages.Logging.ResourcesPreset

	permissionManagerConfig.clusterName = state.Packages.PermissionManager.ClusterName
	permissionManagerConfig.clusterAddress = state.Packages.PermissionManager.ClusterAddress

	enableMirror = state.Mirror.Enabled
	useOneMirror = state.Mirror.UseOneMirror
	oneMirror = state.Mirror.OneMirror
	if state.Mirror.PerRegistry != nil {
		mirrors = cloneStringMap(state.Mirror.PerRegistry)
	}
}

func marshalState(state wizardState) ([]byte, error) {
	return json.MarshalIndent(state, "", "  ")
}

// applyStateJSON merges saved answers over the current values, so fields
// missing from an older state payload keep their in-code defaults.
func applyStateJSON(data []byte) error {
	state := captureState()
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse state: %w", err)
	}
	applyState(state)
	persistedStateLoaded = true
	return nil
}

func stateConfigMapManifest(state wizardState) ([]byte, error) {
	stateJSON, err := marshalState(state)
	if err != nil {
		return nil, err
	}
	return json.Marshal(configMapManifest{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: configMapMetadata{
			Name:      stateConfigMapName,
			Namespace: stateConfigMapNamespace,
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "om-kits-installer"},
		},
		Data: map[string]string{stateConfigMapDataKey: string(stateJSON)},
	})
}

// extractStateFromConfigMap returns the saved state payload. found is false
// when the ConfigMap or the data key is absent, which is normal on first run.
func extractStateFromConfigMap(output []byte) ([]byte, bool, error) {
	if strings.TrimSpace(string(output)) == "" {
		return nil, false, nil
	}

	var configMap struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(output, &configMap); err != nil {
		return nil, false, fmt.Errorf("parse ConfigMap: %w", err)
	}

	stateJSON, ok := configMap.Data[stateConfigMapDataKey]
	if !ok || strings.TrimSpace(stateJSON) == "" {
		return nil, false, nil
	}
	return []byte(stateJSON), true, nil
}

func loadState() error {
	output, err := stateCommand(nil, stateLoadCommand, stateCommandTimeoutSec)
	if err != nil {
		return commandError(fmt.Sprintf("read ConfigMap %s/%s", stateConfigMapNamespace, stateConfigMapName), output, err)
	}

	stateJSON, found, err := extractStateFromConfigMap(output)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	return applyStateJSON(stateJSON)
}

func saveState() error {
	manifest, err := stateConfigMapManifest(captureState())
	if err != nil {
		return fmt.Errorf("build ConfigMap manifest: %w", err)
	}

	output, err := stateCommand(manifest, stateSaveCommand, stateCommandTimeoutSec)
	if err != nil {
		return commandError(fmt.Sprintf("apply ConfigMap %s/%s", stateConfigMapNamespace, stateConfigMapName), output, err)
	}
	return nil
}

func commandError(action string, output []byte, err error) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s: %w: %s", action, err, message)
}

func tlsMethodKey(method string) string {
	switch method {
	case certMethod.defaultTlsSecret:
		return tlsMethodKeyDefaultSecret
	case certMethod.certManager:
		return tlsMethodKeyCertManager
	default:
		return ""
	}
}

func tlsMethodFromKey(key string) string {
	switch key {
	case tlsMethodKeyDefaultSecret:
		return certMethod.defaultTlsSecret
	case tlsMethodKeyCertManager:
		return certMethod.certManager
	default:
		return ""
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
