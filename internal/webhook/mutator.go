/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and secret-generator contributors
SPDX-License-Identifier: Apache-2.0
*/

package webhook

import (
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"

	corev1 "k8s.io/api/core/v1"
)

const (
	AnnotationKeyPrefix = "secret-generator.cs.sap.com/prefix"
	DefaultPrefix       = "%generate"
	Symbols             = `-~!@#$%^&*()_+={}|:<>?,./` // caveat: important to have - at first place (to work in regexp character sets)
)

func handleCreateSecret(secret *corev1.Secret) error {
	prefix := DefaultPrefix
	if v, ok := secret.Annotations[AnnotationKeyPrefix]; ok {
		prefix = v
	}
	for k := range secret.Data {
		if format, ok := parseValue(string(secret.Data[k]), prefix); ok {
			generatedValue, err := generateValue(format)
			if err != nil {
				return errors.Wrapf(err, "error generating value for key '%s'", k)
			}
			secret.Data[k] = []byte(generatedValue)
		}
	}
	return nil
}

func handleUpdateSecret(secret *corev1.Secret, oldSecret *corev1.Secret) error {
	prefix := DefaultPrefix
	if v, ok := secret.Annotations[AnnotationKeyPrefix]; ok {
		prefix = v
	}
	for k := range secret.Data {
		if format, ok := parseValue(string(secret.Data[k]), prefix); ok {
			if v, ok := oldSecret.Data[k]; ok {
				secret.Data[k] = v
			} else {
				generatedValue, err := generateValue(format)
				if err != nil {
					return errors.Wrapf(err, "error generating value for key '%s'", k)
				}
				secret.Data[k] = []byte(generatedValue)
			}
		}
	}
	return nil
}

func parseValue(value string, prefix string) (string, bool) {
	if value == prefix {
		return "", true
	} else if strings.HasPrefix(value, prefix+":") {
		return strings.TrimPrefix(value, prefix+":"), true
	} else {
		return "", false
	}
}

func generateValue(format string) (string, error) {
	if format == "" || format == ":" {
		format = "password"
	}
	m := regexp.MustCompile(`^([^:]+)(?::(.*))?$`).FindStringSubmatch(format)
	generatorType := m[1]
	generatorArgs := m[2]
	var generatedValue string
	var generationError error
	switch generatorType {
	case "password":
		length := 32
		symbols := Symbols
		numDigits := -1
		numSymbols := -1
		encoding := ""
		if generatorArgs != "" {
			for _, arg := range strings.Split(generatorArgs, ";") {
				if m := regexp.MustCompile(`^length=(\d+)$`).FindStringSubmatch(arg); m != nil {
					length, _ = strconv.Atoi(m[1])
				} else if m := regexp.MustCompile(`^symbols=([` + Symbols + `]+)$`).FindStringSubmatch(arg); m != nil {
					symbols = normalizeSymbols(m[1])
				} else if m := regexp.MustCompile(`^num_digits=(\d{1,2})$`).FindStringSubmatch(arg); m != nil {
					numDigits, _ = strconv.Atoi(m[1])
				} else if m := regexp.MustCompile(`^num_symbols=(\d{1,2})$`).FindStringSubmatch(arg); m != nil {
					numSymbols, _ = strconv.Atoi(m[1])
				} else if m := regexp.MustCompile(`^encoding=(.+)$`).FindStringSubmatch(arg); m != nil {
					encoding = m[1]
				} else {
					return "", fmt.Errorf("invalid password generator argument: %s", arg)
				}
			}
		}
		if numDigits < 0 {
			numDigits = length / 4
		}
		if numSymbols < 0 {
			numSymbols = length / 4
		}
		value, err := generatePassword(length, numDigits, numSymbols, symbols)
		if err != nil {
			return "", err
		}
		if encoding == "" {
			generatedValue = value
		} else {
			generatedValue, generationError = encode(encoding, []byte(value))
		}
	case "uuid":
		encoding := ""
		if generatorArgs != "" {
			for _, arg := range strings.Split(generatorArgs, ";") {
				if m := regexp.MustCompile(`^encoding=(.+)$`).FindStringSubmatch(arg); m != nil {
					encoding = m[1]
				} else {
					return "", fmt.Errorf("invalid uuid generator argument: %s", arg)
				}
			}
		}
		generatedUuid := uuid.New()
		if encoding == "" {
			generatedValue = generatedUuid.String()
		} else {
			generatedValue, generationError = encode(encoding, generatedUuid[:])
		}

	default:
		return "", fmt.Errorf("unsupported generator type: %s", generatorType)
	}
	return generatedValue, generationError
}

func encode(encoding string, value []byte) (string, error) {
	var encodedValue string
	var err error

	switch encoding {
	case "base32":
		encodedValue = base32.StdEncoding.EncodeToString(value)
	case "base64":
		encodedValue = base64.StdEncoding.EncodeToString(value)
	case "base64_url":
		encodedValue = base64.URLEncoding.EncodeToString(value)
	case "base64_raw":
		encodedValue = base64.RawStdEncoding.EncodeToString(value)
	case "base64_raw_url":
		encodedValue = base64.RawURLEncoding.EncodeToString(value)
	default:
		err = fmt.Errorf("unsupported encoding %s", encoding)
	}

	return encodedValue, err
}

func generatePassword(length int, numDigits int, numSymbols int, symbols string) (string, error) {
	input := &password.GeneratorInput{Symbols: symbols}
	generator, err := password.NewGenerator(input)
	if err != nil {
		return "", err
	}
	return generator.Generate(length, numDigits, numSymbols, false, true)
}
