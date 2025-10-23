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

package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// This type is used to supported passing in multiple flags
type multiArg []string

// Now, for our new type, implement the two methods of
// the flag.Value interface...
// The first method is String() string
func (b *multiArg) String() string {
	return strings.Join(*b, ",")
}

// The second method is Set(value string) error
func (b *multiArg) Set(value string) error {
	logrus.Debugf("Appending to multi args %s", value)
	*b = append(*b, value)
	return nil
}

// The third is Type() string
func (b *multiArg) Type() string {
	return "multi-arg type"
}

func (b *multiArg) Contains(v string) bool {
	for _, s := range *b {
		if s == v {
			return true
		}
	}
	return false
}

// This type is used to supported passing in multiple key=value flags
type keyValueArg map[string]string

// Now, for our new type, implement the two methods of
// the flag.Value interface...
// The first method is String() string
func (a *keyValueArg) String() string {
	var result []string
	for key := range *a {
		result = append(result, fmt.Sprintf("%s=%s", key, (*a)[key]))
	}
	return strings.Join(result, ",")
}

// The second method is Set(value string) error
func (a *keyValueArg) Set(value string) error {
	valueSplit := strings.SplitN(value, "=", 2)
	if len(valueSplit) < 2 {
		return fmt.Errorf("invalid argument value. expect key=value, got %s", value)
	}
	(*a)[valueSplit[0]] = valueSplit[1]
	return nil
}

// The third is Type() string
func (a *keyValueArg) Type() string {
	return "key-value-arg type"
}

type multiKeyMultiValueArg map[string][]string

func (c *multiKeyMultiValueArg) parseKV(value string) error {
	valueSplit := strings.SplitN(value, "=", 2)
	if len(valueSplit) < 2 {
		return fmt.Errorf("invalid argument value. expect key=value, got %s", value)
	}
	(*c)[valueSplit[0]] = append((*c)[valueSplit[0]], valueSplit[1])
	return nil
}

func (c *multiKeyMultiValueArg) String() string {
	var result []string
	for key := range *c {
		for _, val := range (*c)[key] {
			result = append(result, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return strings.Join(result, ";")

}

func (c *multiKeyMultiValueArg) Set(value string) error {
	if value == "" {
		return nil
	}
	if strings.Contains(value, ";") {
		kvpairs := strings.Split(value, ";")
		for _, kv := range kvpairs {
			err := c.parseKV(kv)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return c.parseKV(value)
}

func (c *multiKeyMultiValueArg) Type() string {
	return "key-multi-value-arg type"
}

// SecretSource represents a secret that can be mounted during build
type SecretSource struct {
	ID     string
	Source string // file path on the host
	Env    string // optional: source from environment variable
}

// secretArg handles --secret flags in the format id=mysecret,src=/path/to/secret
type secretArg map[string]SecretSource

func (s *secretArg) String() string {
	var result []string
	for id, secret := range *s {
		if secret.Env != "" {
			result = append(result, fmt.Sprintf("id=%s,env=%s", id, secret.Env))
		} else {
			result = append(result, fmt.Sprintf("id=%s,src=%s", id, secret.Source))
		}
	}
	return strings.Join(result, " ")
}

func (s *secretArg) Set(value string) error {
	// Parse format: id=mysecret,src=/path/to/secret or id=mysecret,env=MY_ENV_VAR
	parts := strings.Split(value, ",")
	if len(parts) < 2 {
		return fmt.Errorf("invalid secret format. expect id=name,src=path or id=name,env=var, got %s", value)
	}

	var id, src, env string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid secret parameter: %s", part)
		}
		switch kv[0] {
		case "id":
			id = kv[1]
		case "src":
			src = kv[1]
		case "env":
			env = kv[1]
		default:
			return fmt.Errorf("unknown secret parameter: %s", kv[0])
		}
	}

	if id == "" {
		return fmt.Errorf("secret id is required")
	}
	if src == "" && env == "" {
		return fmt.Errorf("either src or env must be specified for secret %s", id)
	}
	if src != "" && env != "" {
		return fmt.Errorf("only one of src or env can be specified for secret %s", id)
	}

	(*s)[id] = SecretSource{
		ID:     id,
		Source: src,
		Env:    env,
	}
	return nil
}

func (s *secretArg) Type() string {
	return "secret-arg type"
}
