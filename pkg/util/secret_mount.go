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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainguard-dev/kaniko/pkg/config"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SecretMount represents a parsed secret mount directive from Dockerfile
type SecretMount struct {
	ID       string
	Target   string
	UID      uint32
	GID      uint32
	Mode     uint32
	Required bool
}

// ParseSecretMounts parses RUN --mount=type=secret directives
func ParseSecretMounts(cmd *instructions.RunCommand) ([]SecretMount, error) {
	var mounts []SecretMount

	buildkitMounts := instructions.GetMounts(cmd)
	if len(buildkitMounts) == 0 {
		return mounts, nil
	}

	for _, mount := range buildkitMounts {
		if mount.Type != instructions.MountTypeSecret {
			continue
		}

		sm := SecretMount{
			UID:      0,
			GID:      0,
			Mode:     0400, // default mode: read-only for owner
			Required: true, // default: required
		}

		// Get the secret ID from CacheID field (used for id= in mount)
		if mount.CacheID != "" {
			sm.ID = mount.CacheID
		} else if mount.Source != "" {
			sm.ID = mount.Source
		}

		// Set target path
		if mount.Target != "" {
			sm.Target = mount.Target
		} else if sm.ID != "" {
			sm.Target = "/run/secrets/" + sm.ID // default target path
		}

		if mount.UID != nil {
			sm.UID = uint32(*mount.UID)
		}

		if mount.GID != nil {
			sm.GID = uint32(*mount.GID)
		}

		if mount.Mode != nil {
			sm.Mode = uint32(*mount.Mode)
		}

		sm.Required = mount.Required

		if sm.ID == "" {
			// If no ID is specified, use the basename of the target
			sm.ID = filepath.Base(sm.Target)
		}

		mounts = append(mounts, sm)
	}

	return mounts, nil
}

// MountSecrets mounts secrets from host to container filesystem
func MountSecrets(mounts []SecretMount, availableSecrets map[string]config.SecretSource) ([]string, error) {
	var mountedPaths []string

	for _, mount := range mounts {
		secret, ok := availableSecrets[mount.ID]
		if !ok {
			if mount.Required {
				return nil, fmt.Errorf("secret with id %s not found", mount.ID)
			}
			logrus.Infof("Optional secret %s not found, skipping", mount.ID)
			continue
		}

		var secretContent []byte
		var err error

		if secret.Env != "" {
			// Read from environment variable
			envValue := os.Getenv(secret.Env)
			if envValue == "" {
				if mount.Required {
					return nil, fmt.Errorf("environment variable %s for secret %s is not set", secret.Env, mount.ID)
				}
				logrus.Infof("Optional environment variable %s not set, skipping secret %s", secret.Env, mount.ID)
				continue
			}
			secretContent = []byte(envValue)
		} else {
			// Read from file
			secretContent, err = os.ReadFile(secret.Source)
			if err != nil {
				if mount.Required {
					return nil, errors.Wrapf(err, "failed to read secret file %s", secret.Source)
				}
				logrus.Infof("Failed to read optional secret file %s, skipping: %v", secret.Source, err)
				continue
			}
		}

		// Create target directory if it doesn't exist
		targetDir := filepath.Dir(mount.Target)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return nil, errors.Wrapf(err, "failed to create directory %s for secret", targetDir)
		}

		// Write secret to target location
		if err := os.WriteFile(mount.Target, secretContent, os.FileMode(mount.Mode)); err != nil {
			return nil, errors.Wrapf(err, "failed to write secret to %s", mount.Target)
		}

		// Set ownership
		if err := os.Chown(mount.Target, int(mount.UID), int(mount.GID)); err != nil {
			logrus.Warnf("Failed to set ownership on secret %s: %v", mount.Target, err)
		}

		logrus.Infof("Mounted secret %s to %s", mount.ID, mount.Target)
		mountedPaths = append(mountedPaths, mount.Target)
	}

	return mountedPaths, nil
}

// UnmountSecrets removes secrets from filesystem
func UnmountSecrets(mountedPaths []string) error {
	for _, path := range mountedPaths {
		// First, overwrite the file with zeros for security
		if info, err := os.Stat(path); err == nil {
			zeroContent := make([]byte, info.Size())
			if err := os.WriteFile(path, zeroContent, info.Mode()); err != nil {
				logrus.Warnf("Failed to overwrite secret file %s: %v", path, err)
			}
		}

		// Then remove the file
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			logrus.Warnf("Failed to remove secret file %s: %v", path, err)
		} else {
			logrus.Debugf("Removed secret from %s", path)
		}
	}
	return nil
}

// StripSecretMounts removes secret mounts from command line for logging/history
// This prevents secrets from appearing in the image history
func StripSecretMounts(cmdLine []string) []string {
	var result []string
	skipNext := false

	for i, arg := range cmdLine {
		if skipNext {
			skipNext = false
			continue
		}

		// Check if this is a --mount flag
		if strings.HasPrefix(arg, "--mount=") {
			// Parse the mount to see if it's a secret
			mountStr := strings.TrimPrefix(arg, "--mount=")
			if strings.Contains(mountStr, "type=secret") {
				// Skip this secret mount
				continue
			}
		} else if arg == "--mount" {
			// Check next argument
			if i+1 < len(cmdLine) && strings.Contains(cmdLine[i+1], "type=secret") {
				skipNext = true
				continue
			}
		}

		result = append(result, arg)
	}

	return result
}
