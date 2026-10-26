package main

import "testing"

func TestMirrorEnvVarsGHCR(t *testing.T) {
	mirrors := map[string]string{"GHCR_CONTAINER_MIRROR": "ghcr.nju.edu.cn"}

	tests := []struct {
		name     string
		enable   bool
		useOne   bool
		one      string
		expected string
	}{
		{name: "mirrors disabled", enable: false, expected: "ghcr.io"},
		{name: "single mirror for all", enable: true, useOne: true, one: "harbor.example.com", expected: "harbor.example.com"},
		{name: "per-registry mirror", enable: true, expected: "ghcr.nju.edu.cn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envs := mirrorEnvVars(tt.enable, tt.useOne, tt.one, mirrors)
			if got := envs["IDO_GHCR_CONTAINER_MIRROR"]; got != tt.expected {
				t.Fatalf("IDO_GHCR_CONTAINER_MIRROR = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMirrorEnvVarsIncludesEveryRegistry(t *testing.T) {
	envs := mirrorEnvVars(false, false, "", map[string]string{})
	for _, key := range []string{
		"IDO_DOCKER_CONTAINER_MIRROR",
		"IDO_QUAY_CONTAINER_MIRROR",
		"IDO_K8S_CONTAINER_MIRROR",
		"IDO_GCR_CONTAINER_MIRROR",
		"IDO_GHCR_CONTAINER_MIRROR",
	} {
		if _, ok := envs[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
}
