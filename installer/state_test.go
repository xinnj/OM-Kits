package main

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// resetStateGlobals puts every persisted global back to a known sentinel so a
// test can prove which values were restored from cluster state.
func resetStateGlobals() {
	basicInfo = BasicInfo{
		host:         "sentinel.invalid",
		httpsEnabled: true,
		timezone:     "Sentinel/Timezone",
		tlsCert: TlsCert{
			certMethod:       certMethod.defaultTlsSecret,
			forceSslRedirect: true,
			acmeEmail:        "sentinel@example.com",
		},
	}
	installLocalPathProvisioner = true
	localPathProvisionerPath = "/sentinel/local-path"
	installNfsProvisioner = true
	nfsProvisionerConfig = NfsProvisionerConfig{
		server:       "sentinel-nfs",
		path:         "/sentinel-nfs-path",
		mountOptions: "vers=4.1",
	}
	installPrometheus = true
	prometheusConfig = PrometheusConfig{
		alertmanagerStorageSizeGi: 31,
		grafanaStorageSizeGi:      32,
		prometheusStorageSizeGi:   33,
		storageClass:              "sentinel-prom-sc",
		grafanaRootUrl:            "https://sentinel.example.com/grafana",
		adapterLang:               "en",
		adapterSignature:          "sentinel-signature",
		dingTalkEnabled:           true,
		dingTalkToken:             "sentinel-dingtalk-token",
		dingTalkSecret:            "sentinel-dingtalk-secret",
		feishuEnabled:             true,
		feishuToken:               "sentinel-feishu-token",
		feishuSecret:              "sentinel-feishu-secret",
		weChatEnabled:             true,
		weChatToken:               "sentinel-wechat-token",
	}
	installLogging = true
	loggingConfig = LoggingConfig{
		collectNamespaces: "sentinel-ns-a,sentinel-ns-b",
		storageClass:      "sentinel-log-sc",
		esStorageSizeGi:   41,
		esDeleteAgeDay:    42,
		esColdAgeDay:      43,
		esWarmAgeDay:      44,
		nodeAffinity:      false,
		errorLogAlert:     true,
		enableMonitoring:  true,
		resourcesPreset:   "sentinel-preset",
	}
	installPermissionManager = true
	permissionManagerConfig = PermissionManagerConfig{
		clusterName:    "sentinel-cluster",
		clusterAddress: "https://sentinel-api:6443",
	}
	enableMirror = true
	useOneMirror = true
	oneMirror = "sentinel.mirror.example.com"
	mirrors = map[string]string{
		"DOCKER_CONTAINER_MIRROR": "sentinel-docker",
		"QUAY_CONTAINER_MIRROR":   "sentinel-quay",
		"K8S_CONTAINER_MIRROR":    "sentinel-k8s",
		"GCR_CONTAINER_MIRROR":    "sentinel-gcr",
		"GHCR_CONTAINER_MIRROR":   "sentinel-ghcr",
	}
	persistedStateLoaded = false
}

type stateCommandCall struct {
	input   []byte
	command string
	timeout int
}

func stubStateCommand(t *testing.T, output []byte, err error) *stateCommandCall {
	t.Helper()

	call := &stateCommandCall{}
	previous := stateCommand
	stateCommand = func(input []byte, command string, timeout int, envs ...string) ([]byte, error) {
		call.input = input
		call.command = command
		call.timeout = timeout
		return output, err
	}
	t.Cleanup(func() { stateCommand = previous })

	return call
}

func marshalStateForTest(t *testing.T) []byte {
	t.Helper()

	data, err := marshalState(captureState())
	if err != nil {
		t.Fatalf("marshalState() returned error: %v", err)
	}
	return data
}

func configMapOutput(t *testing.T, stateJSON []byte) []byte {
	t.Helper()

	output, err := json.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"data":       map[string]string{stateConfigMapDataKey: string(stateJSON)},
	})
	if err != nil {
		t.Fatalf("json.Marshal() returned error: %v", err)
	}
	return output
}

func collectJSONKeys(value any, keys *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			*keys = append(*keys, key)
			collectJSONKeys(child, keys)
		}
	case []any:
		for _, child := range typed {
			collectJSONKeys(child, keys)
		}
	}
}

func TestStateConfigMapManifestShape(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	manifest, err := stateConfigMapManifest(captureState())
	if err != nil {
		t.Fatalf("stateConfigMapManifest() returned error: %v", err)
	}

	var configMap configMapManifest
	if err := json.Unmarshal(manifest, &configMap); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	if configMap.APIVersion != "v1" {
		t.Errorf("apiVersion = %q, want %q", configMap.APIVersion, "v1")
	}
	if configMap.Kind != "ConfigMap" {
		t.Errorf("kind = %q, want %q", configMap.Kind, "ConfigMap")
	}
	if configMap.Metadata.Name != stateConfigMapName {
		t.Errorf("name = %q, want %q", configMap.Metadata.Name, stateConfigMapName)
	}
	if configMap.Metadata.Namespace != stateConfigMapNamespace {
		t.Errorf("namespace = %q, want %q", configMap.Metadata.Namespace, stateConfigMapNamespace)
	}
	if got := configMap.Metadata.Labels["app.kubernetes.io/managed-by"]; got != "om-kits-installer" {
		t.Errorf("managed-by label = %q, want %q", got, "om-kits-installer")
	}
	if len(configMap.Data) != 1 {
		t.Fatalf("data has %d keys, want only %q", len(configMap.Data), stateConfigMapDataKey)
	}

	var state wizardState
	if err := json.Unmarshal([]byte(configMap.Data[stateConfigMapDataKey]), &state); err != nil {
		t.Fatalf("embedded state is not valid JSON: %v", err)
	}
	if state.SchemaVersion != stateSchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", state.SchemaVersion, stateSchemaVersion)
	}
	if state.BasicInfo.Host != "sentinel.invalid" {
		t.Errorf("host = %q, want %q", state.BasicInfo.Host, "sentinel.invalid")
	}
	if state.BasicInfo.TLSCert.CertMethod != tlsMethodKeyDefaultSecret {
		t.Errorf("certMethod = %q, want stable key %q", state.BasicInfo.TLSCert.CertMethod, tlsMethodKeyDefaultSecret)
	}
}

func TestStateConfigMapManifestExcludesCredentials(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	manifest, err := stateConfigMapManifest(captureState())
	if err != nil {
		t.Fatalf("stateConfigMapManifest() returned error: %v", err)
	}

	for _, credential := range []string{
		"sentinel-dingtalk-token",
		"sentinel-dingtalk-secret",
		"sentinel-feishu-token",
		"sentinel-feishu-secret",
		"sentinel-wechat-token",
	} {
		if strings.Contains(string(manifest), credential) {
			t.Errorf("manifest contains credential %q", credential)
		}
	}

	var configMap configMapManifest
	if err := json.Unmarshal(manifest, &configMap); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}

	var embedded any
	if err := json.Unmarshal([]byte(configMap.Data[stateConfigMapDataKey]), &embedded); err != nil {
		t.Fatalf("embedded state is not valid JSON: %v", err)
	}
	var keys []string
	collectJSONKeys(embedded, &keys)
	for _, key := range keys {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
			t.Errorf("embedded state contains credential-like key %q", key)
		}
	}
}

func TestPrometheusStateHasNoCredentialFields(t *testing.T) {
	typ := reflect.TypeOf(prometheusState{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.ToLower(typ.Field(i).Name)
		if strings.Contains(name, "token") || strings.Contains(name, "secret") {
			t.Errorf("prometheusState has credential field %q", typ.Field(i).Name)
		}
	}
}

func TestApplyStateJSONRestoresBasicInfo(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	data := marshalStateForTest(t)
	basicInfo = BasicInfo{}

	if err := applyStateJSON(data); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}

	if basicInfo.host != "sentinel.invalid" {
		t.Errorf("host = %q, want %q", basicInfo.host, "sentinel.invalid")
	}
	if !basicInfo.httpsEnabled {
		t.Error("httpsEnabled = false, want true")
	}
	if basicInfo.timezone != "Sentinel/Timezone" {
		t.Errorf("timezone = %q, want %q", basicInfo.timezone, "Sentinel/Timezone")
	}
	if basicInfo.tlsCert.certMethod != certMethod.defaultTlsSecret {
		t.Errorf("certMethod = %q, want %q", basicInfo.tlsCert.certMethod, certMethod.defaultTlsSecret)
	}
	if !basicInfo.tlsCert.forceSslRedirect {
		t.Error("forceSslRedirect = false, want true")
	}
	if basicInfo.tlsCert.acmeEmail != "sentinel@example.com" {
		t.Errorf("acmeEmail = %q, want %q", basicInfo.tlsCert.acmeEmail, "sentinel@example.com")
	}
	if !persistedStateLoaded {
		t.Error("persistedStateLoaded = false, want true")
	}
}

func TestApplyStateJSONRestoresPackages(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	data := marshalStateForTest(t)

	installLocalPathProvisioner = false
	localPathProvisionerPath = ""
	installNfsProvisioner = false
	nfsProvisionerConfig = NfsProvisionerConfig{}
	installPrometheus = false
	prometheusConfig = PrometheusConfig{
		dingTalkToken:  "from-secret",
		dingTalkSecret: "from-secret",
		feishuToken:    "from-secret",
		feishuSecret:   "from-secret",
		weChatToken:    "from-secret",
	}
	installLogging = false
	loggingConfig = LoggingConfig{}
	installPermissionManager = false
	permissionManagerConfig = PermissionManagerConfig{}

	if err := applyStateJSON(data); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}

	if installLocalPathProvisioner {
		t.Error("installLocalPathProvisioner = true, want unchecked on a new run")
	}
	if localPathProvisionerPath != "/sentinel/local-path" {
		t.Errorf("local path = %q, want %q", localPathProvisionerPath, "/sentinel/local-path")
	}
	if installNfsProvisioner {
		t.Error("installNfsProvisioner = true, want unchecked on a new run")
	}
	if nfsProvisionerConfig != (NfsProvisionerConfig{
		server:       "sentinel-nfs",
		path:         "/sentinel-nfs-path",
		mountOptions: "vers=4.1",
	}) {
		t.Errorf("nfs provisioner = %+v, want sentinel values", nfsProvisionerConfig)
	}
	if installPrometheus {
		t.Error("installPrometheus = true, want unchecked on a new run")
	}
	if prometheusConfig != (PrometheusConfig{
		alertmanagerStorageSizeGi: 31,
		grafanaStorageSizeGi:      32,
		prometheusStorageSizeGi:   33,
		storageClass:              "sentinel-prom-sc",
		grafanaRootUrl:            "https://sentinel.example.com/grafana",
		adapterLang:               "en",
		adapterSignature:          "sentinel-signature",
		dingTalkEnabled:           true,
		dingTalkToken:             "from-secret",
		dingTalkSecret:            "from-secret",
		feishuEnabled:             true,
		feishuToken:               "from-secret",
		feishuSecret:              "from-secret",
		weChatEnabled:             true,
		weChatToken:               "from-secret",
	}) {
		t.Fatalf("prometheus config was not restored without touching credentials: %+v", prometheusConfig)
	}
	if installLogging {
		t.Error("installLogging = true, want unchecked on a new run")
	}
	if loggingConfig != (LoggingConfig{
		collectNamespaces: "sentinel-ns-a,sentinel-ns-b",
		storageClass:      "sentinel-log-sc",
		esStorageSizeGi:   41,
		esDeleteAgeDay:    42,
		esColdAgeDay:      43,
		esWarmAgeDay:      44,
		nodeAffinity:      false,
		errorLogAlert:     true,
		enableMonitoring:  true,
		resourcesPreset:   "sentinel-preset",
	}) {
		t.Errorf("logging config = %+v, want sentinel values", loggingConfig)
	}
	if installPermissionManager {
		t.Error("installPermissionManager = true, want unchecked on a new run")
	}
	if permissionManagerConfig != (PermissionManagerConfig{
		clusterName:    "sentinel-cluster",
		clusterAddress: "https://sentinel-api:6443",
	}) {
		t.Errorf("permission manager = %+v, want sentinel values", permissionManagerConfig)
	}
}

func TestApplyStateJSONIgnoresInstallSelection(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	// Older payloads stored the install checkboxes. A new run must start with
	// every package unchecked while still restoring the field values.
	payload := []byte(`{"packages":{` +
		`"localPathProvisioner":{"install":true,"path":"/legacy-path"},` +
		`"nfsProvisioner":{"install":true,"server":"legacy-nfs"},` +
		`"prometheus":{"install":true,"storageClass":"legacy-sc"},` +
		`"logging":{"install":true,"resourcesPreset":"large"},` +
		`"permissionManager":{"install":true,"clusterName":"legacy-cluster"}}}`)

	installLocalPathProvisioner = false
	installNfsProvisioner = false
	installPrometheus = false
	installLogging = false
	installPermissionManager = false

	if err := applyStateJSON(payload); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}

	if installLocalPathProvisioner || installNfsProvisioner || installPrometheus ||
		installLogging || installPermissionManager {
		t.Error("install checkboxes were restored; want all unchecked")
	}
	if localPathProvisionerPath != "/legacy-path" ||
		nfsProvisionerConfig.server != "legacy-nfs" ||
		prometheusConfig.storageClass != "legacy-sc" ||
		loggingConfig.resourcesPreset != "large" ||
		permissionManagerConfig.clusterName != "legacy-cluster" {
		t.Errorf("package fields were not restored: path=%q nfs=%q promSC=%q preset=%q cluster=%q",
			localPathProvisionerPath, nfsProvisionerConfig.server, prometheusConfig.storageClass,
			loggingConfig.resourcesPreset, permissionManagerConfig.clusterName)
	}
}

func TestApplyStateJSONRestoresMirror(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	data := marshalStateForTest(t)
	enableMirror = false
	useOneMirror = false
	oneMirror = ""
	mirrors = map[string]string{"DOCKER_CONTAINER_MIRROR": "docker.io"}

	if err := applyStateJSON(data); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}

	if !enableMirror || !useOneMirror || oneMirror != "sentinel.mirror.example.com" {
		t.Errorf("mirror settings = (%v, %v, %q), want (true, true, %q)",
			enableMirror, useOneMirror, oneMirror, "sentinel.mirror.example.com")
	}
	if !reflect.DeepEqual(mirrors, map[string]string{
		"DOCKER_CONTAINER_MIRROR": "sentinel-docker",
		"QUAY_CONTAINER_MIRROR":   "sentinel-quay",
		"K8S_CONTAINER_MIRROR":    "sentinel-k8s",
		"GCR_CONTAINER_MIRROR":    "sentinel-gcr",
		"GHCR_CONTAINER_MIRROR":   "sentinel-ghcr",
	}) {
		t.Errorf("mirrors = %#v, want sentinel values", mirrors)
	}
}

func TestStateRetainsUnselectedPackageSettings(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	// Simulate a run where all package checkboxes are off but their fields
	// still hold values, like installing package 2 after package 1.
	installLocalPathProvisioner = false
	installNfsProvisioner = false
	installPrometheus = false
	installLogging = false
	installPermissionManager = false
	data := marshalStateForTest(t)

	localPathProvisionerPath = ""
	nfsProvisionerConfig = NfsProvisionerConfig{}
	prometheusConfig = PrometheusConfig{}
	loggingConfig = LoggingConfig{}
	permissionManagerConfig = PermissionManagerConfig{}

	if err := applyStateJSON(data); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}

	if localPathProvisionerPath != "/sentinel/local-path" {
		t.Errorf("local path = %q, want sentinel value", localPathProvisionerPath)
	}
	if nfsProvisionerConfig.server != "sentinel-nfs" {
		t.Errorf("NFS server = %q, want sentinel value", nfsProvisionerConfig.server)
	}
	if prometheusConfig.storageClass != "sentinel-prom-sc" {
		t.Errorf("Prometheus storage class = %q, want sentinel value", prometheusConfig.storageClass)
	}
	if loggingConfig.resourcesPreset != "sentinel-preset" {
		t.Errorf("Logging resources preset = %q, want sentinel value", loggingConfig.resourcesPreset)
	}
	if permissionManagerConfig.clusterName != "sentinel-cluster" {
		t.Errorf("Permission Manager cluster = %q, want sentinel value", permissionManagerConfig.clusterName)
	}
	if installLocalPathProvisioner || installNfsProvisioner || installPrometheus ||
		installLogging || installPermissionManager {
		t.Error("install checkboxes should stay off")
	}
}

func TestApplyStateJSONPartialKeepsDefaults(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	if err := applyStateJSON([]byte(`{"basicInfo":{"host":"remembered.example.com"}}`)); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}

	if basicInfo.host != "remembered.example.com" {
		t.Errorf("host = %q, want %q", basicInfo.host, "remembered.example.com")
	}
	if basicInfo.timezone != "Sentinel/Timezone" {
		t.Errorf("timezone = %q, want existing value %q", basicInfo.timezone, "Sentinel/Timezone")
	}
	if loggingConfig.esStorageSizeGi != 41 {
		t.Errorf("esStorageSizeGi = %d, want existing value 41", loggingConfig.esStorageSizeGi)
	}
	if oneMirror != "sentinel.mirror.example.com" {
		t.Errorf("oneMirror = %q, want existing value %q", oneMirror, "sentinel.mirror.example.com")
	}
}

func TestApplyStateJSONIgnoresLegacyCredentialFields(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	payload := []byte(`{"packages":{"prometheus":{` +
		`"dingTalkToken":"legacy","dingTalkSecret":"legacy",` +
		`"feishuToken":"legacy","feishuSecret":"legacy",` +
		`"weChatToken":"legacy"}}}`)

	if err := applyStateJSON(payload); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}

	if prometheusConfig.dingTalkToken != "sentinel-dingtalk-token" ||
		prometheusConfig.dingTalkSecret != "sentinel-dingtalk-secret" ||
		prometheusConfig.feishuToken != "sentinel-feishu-token" ||
		prometheusConfig.feishuSecret != "sentinel-feishu-secret" ||
		prometheusConfig.weChatToken != "sentinel-wechat-token" {
		t.Errorf("legacy payload changed credentials: %+v", prometheusConfig)
	}
}

func TestApplyStateJSONNullMirrorMapKeepsExisting(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	if err := applyStateJSON([]byte(`{"mirror":{"perRegistry":null}}`)); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}

	if mirrors == nil {
		t.Fatal("mirrors is nil after applying a null perRegistry map")
	}
	if mirrors["DOCKER_CONTAINER_MIRROR"] != "sentinel-docker" {
		t.Errorf("DOCKER_CONTAINER_MIRROR = %q, want existing value %q",
			mirrors["DOCKER_CONTAINER_MIRROR"], "sentinel-docker")
	}
}

func TestApplyStateJSONInvalidJSONReturnsErrorAndLeavesGlobals(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	if err := applyStateJSON([]byte(`{"basicInfo":`)); err == nil {
		t.Fatal("applyStateJSON() returned nil error for invalid JSON")
	}
	if basicInfo.host != "sentinel.invalid" {
		t.Errorf("host = %q, want existing value %q", basicInfo.host, "sentinel.invalid")
	}
	if persistedStateLoaded {
		t.Error("persistedStateLoaded = true after invalid JSON")
	}
}

func TestApplyStateJSONUnknownFieldsIgnored(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	payload := []byte(`{"futureSetting":true,"basicInfo":{"host":"future.example.com"}}`)
	if err := applyStateJSON(payload); err != nil {
		t.Fatalf("applyStateJSON() returned error: %v", err)
	}
	if basicInfo.host != "future.example.com" {
		t.Errorf("host = %q, want %q", basicInfo.host, "future.example.com")
	}
}

func TestStateStoresStableTLSMethodKeys(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	basicInfo.tlsCert.certMethod = certMethod.certManager
	if got := captureState().BasicInfo.TLSCert.CertMethod; got != tlsMethodKeyCertManager {
		t.Errorf("cert-manager key = %q, want %q", got, tlsMethodKeyCertManager)
	}

	basicInfo.tlsCert.certMethod = certMethod.defaultTlsSecret
	if got := captureState().BasicInfo.TLSCert.CertMethod; got != tlsMethodKeyDefaultSecret {
		t.Errorf("default TLS key = %q, want %q", got, tlsMethodKeyDefaultSecret)
	}
}

func TestMergeStorageClasses(t *testing.T) {
	got := mergeStorageClasses([]string{"local-path", "nfs-client"}, "fast", "", "local-path", "fast")
	want := []string{"local-path", "nfs-client", "fast"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("mergeStorageClasses() = %#v, want %#v", got, want)
	}
}

func TestDropdownIndex(t *testing.T) {
	options := []string{"zh", "en"}

	tests := []struct {
		name     string
		value    string
		fallback string
		want     int
	}{
		{name: "present value", value: "en", fallback: "zh", want: 1},
		{name: "empty uses fallback", value: "", fallback: "zh", want: 0},
		{name: "unknown uses fallback", value: "fr", fallback: "zh", want: 0},
		{name: "missing fallback", value: "fr", fallback: "de", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dropdownIndex(options, tt.value, tt.fallback); got != tt.want {
				t.Errorf("dropdownIndex() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLoadStateReadsConfigMap(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	stateJSON := marshalStateForTest(t)
	call := stubStateCommand(t, configMapOutput(t, stateJSON), nil)

	basicInfo = BasicInfo{}
	if err := loadState(); err != nil {
		t.Fatalf("loadState() returned error: %v", err)
	}

	if call.command != stateLoadCommand {
		t.Errorf("command = %q, want %q", call.command, stateLoadCommand)
	}
	if call.timeout != stateCommandTimeoutSec {
		t.Errorf("timeout = %d, want %d", call.timeout, stateCommandTimeoutSec)
	}
	if call.input != nil {
		t.Errorf("load passed stdin %q, want nil", call.input)
	}
	if basicInfo.host != "sentinel.invalid" {
		t.Errorf("host = %q, want %q", basicInfo.host, "sentinel.invalid")
	}
	if !persistedStateLoaded {
		t.Error("persistedStateLoaded = false, want true")
	}
}

func TestLoadStateMissingOrEmptyConfigMapIsNoOp(t *testing.T) {
	tests := []struct {
		name   string
		output []byte
	}{
		{name: "missing ConfigMap", output: nil},
		{name: "no data", output: []byte(`{"data":{}}`)},
		{name: "empty state", output: []byte(`{"data":{"state":""}}`)},
		{name: "whitespace state", output: []byte(`{"data":{"state":"   "}}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetStateGlobals()
			t.Cleanup(resetStateGlobals)
			stubStateCommand(t, tt.output, nil)

			if err := loadState(); err != nil {
				t.Fatalf("loadState() returned error: %v", err)
			}
			if basicInfo.host != "sentinel.invalid" {
				t.Errorf("host = %q, want existing value %q", basicInfo.host, "sentinel.invalid")
			}
			if persistedStateLoaded {
				t.Error("persistedStateLoaded = true for a no-op load")
			}
		})
	}
}

func TestLoadStateMalformedConfigMapOrStateReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		output []byte
	}{
		{name: "malformed ConfigMap", output: []byte(`not-json`)},
		{name: "malformed state", output: []byte(`{"data":{"state":"not-json"}}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetStateGlobals()
			t.Cleanup(resetStateGlobals)
			stubStateCommand(t, tt.output, nil)

			if err := loadState(); err == nil {
				t.Fatal("loadState() returned nil error")
			}
			if basicInfo.host != "sentinel.invalid" {
				t.Errorf("host = %q, want existing value %q", basicInfo.host, "sentinel.invalid")
			}
			if persistedStateLoaded {
				t.Error("persistedStateLoaded = true after a malformed load")
			}
		})
	}
}

func TestLoadStateKubectlFailureReturnsError(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	stubStateCommand(t, []byte("forbidden: cannot get configmaps"), errors.New("exit status 1"))

	err := loadState()
	if err == nil {
		t.Fatal("loadState() returned nil error")
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("error %q does not include kubectl output", err)
	}
	if basicInfo.host != "sentinel.invalid" {
		t.Errorf("host = %q, want existing value %q", basicInfo.host, "sentinel.invalid")
	}
}

func TestSaveStateAppliesManifestOnStdin(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	call := stubStateCommand(t, []byte("configmap/om-kits-state configured"), nil)

	if err := saveState(); err != nil {
		t.Fatalf("saveState() returned error: %v", err)
	}

	if call.command != "kubectl apply -f -" {
		t.Errorf("command = %q, want %q", call.command, "kubectl apply -f -")
	}
	if call.timeout != stateCommandTimeoutSec {
		t.Errorf("timeout = %d, want %d", call.timeout, stateCommandTimeoutSec)
	}
	if call.input == nil {
		t.Fatal("save did not pass a manifest on stdin")
	}

	var configMap configMapManifest
	if err := json.Unmarshal(call.input, &configMap); err != nil {
		t.Fatalf("stdin is not a valid ConfigMap manifest: %v", err)
	}
	if configMap.Metadata.Name != stateConfigMapName || configMap.Metadata.Namespace != stateConfigMapNamespace {
		t.Errorf("manifest metadata = %s/%s, want %s/%s",
			configMap.Metadata.Namespace, configMap.Metadata.Name,
			stateConfigMapNamespace, stateConfigMapName)
	}

	var state wizardState
	if err := json.Unmarshal([]byte(configMap.Data[stateConfigMapDataKey]), &state); err != nil {
		t.Fatalf("embedded state is not valid JSON: %v", err)
	}
	if state.BasicInfo.Host != "sentinel.invalid" {
		t.Errorf("host = %q, want %q", state.BasicInfo.Host, "sentinel.invalid")
	}
	if strings.Contains(string(call.input), "sentinel-dingtalk-token") {
		t.Error("manifest contains a credential")
	}
}

func TestSaveStateReturnsKubectlFailure(t *testing.T) {
	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	stubStateCommand(t, []byte("forbidden: cannot create configmaps"), errors.New("exit status 1"))

	err := saveState()
	if err == nil {
		t.Fatal("saveState() returned nil error")
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("error %q does not include kubectl output", err)
	}
}
