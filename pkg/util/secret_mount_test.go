/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainguard-dev/kaniko/pkg/config"
)

func TestMountSecrets(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "kaniko-secret-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test secret file
	secretContent := "my-secret-password"
	secretFile := filepath.Join(tmpDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte(secretContent), 0600); err != nil {
		t.Fatalf("Failed to create secret file: %v", err)
	}

	// Create target directory
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	tests := []struct {
		name             string
		mounts           []SecretMount
		availableSecrets map[string]config.SecretSource
		expectedPaths    int
		shouldError      bool
		setupEnv         func()
		cleanupEnv       func()
	}{
		{
			name: "mount secret from file",
			mounts: []SecretMount{
				{
					ID:       "mysecret",
					Target:   filepath.Join(targetDir, "secret1"),
					UID:      0,
					GID:      0,
					Mode:     0400,
					Required: true,
				},
			},
			availableSecrets: map[string]config.SecretSource{
				"mysecret": {
					ID:     "mysecret",
					Source: secretFile,
				},
			},
			expectedPaths: 1,
			shouldError:   false,
		},
		{
			name: "mount secret from env",
			mounts: []SecretMount{
				{
					ID:       "envsecret",
					Target:   filepath.Join(targetDir, "secret2"),
					UID:      0,
					GID:      0,
					Mode:     0400,
					Required: true,
				},
			},
			availableSecrets: map[string]config.SecretSource{
				"envsecret": {
					ID:  "envsecret",
					Env: "TEST_SECRET_VAR",
				},
			},
			expectedPaths: 1,
			shouldError:   false,
			setupEnv: func() {
				os.Setenv("TEST_SECRET_VAR", "env-secret-value")
			},
			cleanupEnv: func() {
				os.Unsetenv("TEST_SECRET_VAR")
			},
		},
		{
			name: "required secret not found",
			mounts: []SecretMount{
				{
					ID:       "missing",
					Target:   filepath.Join(targetDir, "secret3"),
					UID:      0,
					GID:      0,
					Mode:     0400,
					Required: true,
				},
			},
			availableSecrets: map[string]config.SecretSource{},
			expectedPaths:    0,
			shouldError:      true,
		},
		{
			name: "optional secret not found",
			mounts: []SecretMount{
				{
					ID:       "optional",
					Target:   filepath.Join(targetDir, "secret4"),
					UID:      0,
					GID:      0,
					Mode:     0400,
					Required: false,
				},
			},
			availableSecrets: map[string]config.SecretSource{},
			expectedPaths:    0,
			shouldError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			paths, err := MountSecrets(tt.mounts, tt.availableSecrets)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(paths) != tt.expectedPaths {
				t.Errorf("Expected %d paths, got %d", tt.expectedPaths, len(paths))
			}

			// Verify secret content if mount was successful
			if !tt.shouldError && len(paths) > 0 {
				for _, path := range paths {
					content, err := os.ReadFile(path)
					if err != nil {
						t.Errorf("Failed to read mounted secret: %v", err)
					}
					if len(content) == 0 {
						t.Error("Mounted secret is empty")
					}
				}
			}

			// Cleanup mounted secrets
			if len(paths) > 0 {
				UnmountSecrets(paths)
			}
		})
	}
}

func TestUnmountSecrets(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "kaniko-secret-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test secret files
	secretPath := filepath.Join(tmpDir, "test-secret")
	if err := os.WriteFile(secretPath, []byte("secret-content"), 0600); err != nil {
		t.Fatalf("Failed to create secret file: %v", err)
	}

	// Unmount the secret
	err = UnmountSecrets([]string{secretPath})
	if err != nil {
		t.Errorf("UnmountSecrets failed: %v", err)
	}

	// Verify the file is removed
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Error("Secret file was not removed")
	}
}

func TestStripSecretMounts(t *testing.T) {
	tests := []struct {
		name     string
		cmdLine  []string
		expected []string
	}{
		{
			name: "strip secret mount with equals",
			cmdLine: []string{
				"run",
				"--mount=type=secret,id=mysecret,target=/run/secrets/mysecret",
				"cat",
				"/run/secrets/mysecret",
			},
			expected: []string{"run", "cat", "/run/secrets/mysecret"},
		},
		{
			name: "strip secret mount without equals",
			cmdLine: []string{
				"run",
				"--mount",
				"type=secret,id=mysecret",
				"cat",
				"/run/secrets/mysecret",
			},
			expected: []string{"run", "cat", "/run/secrets/mysecret"},
		},
		{
			name: "keep non-secret mounts",
			cmdLine: []string{
				"run",
				"--mount=type=bind,source=/src,target=/dst",
				"ls",
				"/dst",
			},
			expected: []string{
				"run",
				"--mount=type=bind,source=/src,target=/dst",
				"ls",
				"/dst",
			},
		},
		{
			name: "mixed mounts",
			cmdLine: []string{
				"run",
				"--mount=type=secret,id=mysecret",
				"--mount=type=cache,target=/cache",
				"cat",
				"/run/secrets/mysecret",
			},
			expected: []string{
				"run",
				"--mount=type=cache,target=/cache",
				"cat",
				"/run/secrets/mysecret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripSecretMounts(tt.cmdLine)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d args, got %d", len(tt.expected), len(result))
				t.Logf("Expected: %v", tt.expected)
				t.Logf("Got: %v", result)
				return
			}

			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Arg %d: expected %q, got %q", i, tt.expected[i], arg)
				}
			}
		})
	}
}
