package main

import (
	"reflect"
	"testing"
)

func TestParseAdapterCredentials(t *testing.T) {
	output := []byte("DINGTALK_TOKEN=ding-token\nDINGTALK_SECRET=ding-secret\nWEIXIN_TOKEN=wechat-key\n")

	got := parseAdapterCredentials(output)
	want := map[string]string{
		"DINGTALK_TOKEN":  "ding-token",
		"DINGTALK_SECRET": "ding-secret",
		"WEIXIN_TOKEN":    "wechat-key",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseAdapterCredentials() = %#v, want %#v", got, want)
	}
}

func TestRestoreAdapterCredentialsAlwaysFillTokens(t *testing.T) {
	creds := map[string]string{
		"DINGTALK_TOKEN":  "cluster-ding-token",
		"DINGTALK_SECRET": "cluster-ding-secret",
		"FEISHU_TOKEN":    "cluster-feishu-token",
		"FEISHU_SECRET":   "cluster-feishu-secret",
		"WEIXIN_TOKEN":    "cluster-wechat-key",
	}

	for _, loaded := range []bool{false, true} {
		loaded := loaded
		t.Run(map[bool]string{false: "no saved state", true: "saved state"}[loaded], func(t *testing.T) {
			resetStateGlobals()
			t.Cleanup(resetStateGlobals)
			persistedStateLoaded = loaded

			prometheusConfig.dingTalkToken = "old"
			prometheusConfig.dingTalkSecret = "old"
			prometheusConfig.feishuToken = "old"
			prometheusConfig.feishuSecret = "old"
			prometheusConfig.weChatToken = "old"

			applyRestoredAdapterCredentials(creds)

			if prometheusConfig.dingTalkToken != "cluster-ding-token" ||
				prometheusConfig.dingTalkSecret != "cluster-ding-secret" ||
				prometheusConfig.feishuToken != "cluster-feishu-token" ||
				prometheusConfig.feishuSecret != "cluster-feishu-secret" ||
				prometheusConfig.weChatToken != "cluster-wechat-key" {
				t.Errorf("credentials not filled from Secret: %+v", prometheusConfig)
			}
		})
	}
}

func TestRestoreAdapterEnablementDerivedOnlyWithoutSavedState(t *testing.T) {
	creds := map[string]string{
		"DINGTALK_TOKEN":  "token",
		"DINGTALK_SECRET": "secret",
		"FEISHU_TOKEN":    "token",
		"FEISHU_SECRET":   "secret",
		"WEIXIN_TOKEN":    "key",
	}

	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	persistedStateLoaded = false
	prometheusConfig.dingTalkEnabled = false
	prometheusConfig.feishuEnabled = false
	prometheusConfig.weChatEnabled = false
	applyRestoredAdapterCredentials(creds)

	if !prometheusConfig.dingTalkEnabled || !prometheusConfig.feishuEnabled || !prometheusConfig.weChatEnabled {
		t.Error("enablement was not derived from Secret keys without saved state")
	}

	persistedStateLoaded = true
	prometheusConfig.dingTalkEnabled = false
	prometheusConfig.feishuEnabled = true
	prometheusConfig.weChatEnabled = false
	applyRestoredAdapterCredentials(creds)

	if prometheusConfig.dingTalkEnabled || !prometheusConfig.feishuEnabled || prometheusConfig.weChatEnabled {
		t.Error("saved enablement was overwritten by the Secret")
	}
}

func TestRestoreAdapterArgsHonorSavedState(t *testing.T) {
	lines := []string{"--signature=cluster-signature", "--tmpl-lang=en"}

	resetStateGlobals()
	t.Cleanup(resetStateGlobals)

	persistedStateLoaded = false
	prometheusConfig.adapterSignature = "saved-signature"
	prometheusConfig.adapterLang = "zh"
	applyRestoredAdapterArgs(lines)

	if prometheusConfig.adapterSignature != "cluster-signature" || prometheusConfig.adapterLang != "en" {
		t.Errorf("deployment args not applied without saved state: %+v", prometheusConfig)
	}

	persistedStateLoaded = true
	prometheusConfig.adapterSignature = "saved-signature"
	prometheusConfig.adapterLang = "zh"
	applyRestoredAdapterArgs(lines)

	if prometheusConfig.adapterSignature != "saved-signature" || prometheusConfig.adapterLang != "zh" {
		t.Errorf("saved alerting settings were overwritten: %+v", prometheusConfig)
	}
}
