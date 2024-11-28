/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and secret-generator contributors
SPDX-License-Identifier: Apache-2.0
*/

package webhook

import (
	"encoding/base32"
	"encoding/base64"
	"regexp"
	"testing"

	"github.com/google/uuid"

	corev1 "k8s.io/api/core/v1"
)

func TestHandleCreateSecret(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"key1": []byte("%generate"),
			"key2": []byte("%generate:password:length=8"),
			"key3": []byte("%generate:uuid"),
			"key4": []byte("value"),
		},
	}
	if err := handleCreateSecret(secret); err != nil {
		t.Fatalf("handleCreateSecret: got errror: %s", err)
	}
	if s := string(secret.Data["key1"]); len(s) != 32 {
		t.Errorf("handleCreateSecret: got invalid password: %s", s)
	}
	if s := string(secret.Data["key2"]); len(s) != 8 {
		t.Errorf("handleCreateSecret: got invalid password: %s", s)
	}
	if _, err := uuid.Parse(string(secret.Data["key3"])); err != nil {
		t.Errorf("handleCreateSecret: got invalid uuid; error: %s", err)
	}
	if s := string(secret.Data["key4"]); s != "value" {
		t.Errorf("handleCreateSecret: got invalid unmanaged value: %s", s)
	}
}

func TestHandleCreateSecretWithError(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"key1": []byte("%generate:foobar"),
		},
	}
	if err := handleCreateSecret(secret); err == nil {
		t.Error("handleCreateSecret: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}
}

func TestHandleUpdateSecret(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"key1":         []byte("%generate"),
			"key2":         []byte("%generate:password:length=8"),
			"key3":         []byte("%generate:uuid"),
			"key4":         []byte("value"),
			"existingKey1": []byte("%generate"),
			"existingKey2": []byte("%generate:password:length=8"),
			"existingKey3": []byte("%generate:uuid"),
			"existingKey4": []byte("value"),
		},
	}
	oldSecret := &corev1.Secret{
		Data: map[string][]byte{
			"existingKey1": []byte("ABCDEFGHIJKLMabcdefghijklm012345"),
			"existingKey2": []byte("ABCabc12"),
			"existingKey3": []byte("eb89b65f-cd54-40b4-8122-e8d42ce3c324"),
			"existingKey4": []byte("value"),
		},
	}
	if err := handleUpdateSecret(secret, oldSecret); err != nil {
		t.Fatalf("handleUpdateSecret: got errror: %s", err)
	}
	if s := string(secret.Data["key1"]); len(s) != 32 {
		t.Errorf("handleUpdateSecret: got invalid password: %s", s)
	}
	if s := string(secret.Data["key2"]); len(s) != 8 {
		t.Errorf("handleUpdateSecret: got invalid password: %s", s)
	}
	if _, err := uuid.Parse(string(secret.Data["key3"])); err != nil {
		t.Errorf("handleUpdateSecret: got invalid uuid; error: %s", err)
	}
	if s := string(secret.Data["key4"]); s != "value" {
		t.Errorf("handleUpdateSecret: got invalid unmanaged value: %s", s)
	}
	if string(secret.Data["existingKey1"]) != string(oldSecret.Data["existingKey1"]) {
		t.Error("handleUpdateSecret: existing value got changed")
	}
	if string(secret.Data["existingKey2"]) != string(oldSecret.Data["existingKey2"]) {
		t.Error("handleUpdateSecret: existing value got changed")
	}
	if string(secret.Data["existingKey3"]) != string(oldSecret.Data["existingKey3"]) {
		t.Error("handleUpdateSecret: existing value got changed")
	}
	if string(secret.Data["existingKey4"]) != string(oldSecret.Data["existingKey4"]) {
		t.Error("handleUpdateSecret: existing value got changed")
	}
}

func TestHandleUpdateSecretWithError(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"key1": []byte("%generate:foobar"),
		},
	}
	oldSecret := &corev1.Secret{}
	if err := handleUpdateSecret(secret, oldSecret); err == nil {
		t.Error("handleUpdateSecret: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}
}

func TestGenerateValue(t *testing.T) {
	var v string
	var err error

	// short form; will be interpreted as password without arguments
	v, err = generateValue("")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9` + Symbols + `]{32}$`).MatchString(v) {
		t.Errorf("generateValue: got invalid password (wrong length): %s", v)
	}
	if len(regexp.MustCompile(`[A-Za-z]`).FindAllString(v, -1)) != 16 {
		t.Errorf("generateValue: got invalid password (wrong letter count): %s", v)
	}
	if len(regexp.MustCompile(`[0-9]`).FindAllString(v, -1)) != 8 {
		t.Errorf("generateValue: got invalid password (wrong digit count): %s", v)
	}
	if len(regexp.MustCompile(`[`+Symbols+`]`).FindAllString(v, -1)) != 8 {
		t.Errorf("generateValue: got invalid password (wrong symbol count): %s", v)
	}

	// short form; will be interpreted as password without arguments
	v, err = generateValue(":")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9` + Symbols + `]{32}$`).MatchString(v) {
		t.Errorf("generateValue: got invalid password (wrong length): %s", v)
	}
	if len(regexp.MustCompile(`[A-Za-z]`).FindAllString(v, -1)) != 16 {
		t.Errorf("generateValue: got invalid password (wrong letter count): %s", v)
	}
	if len(regexp.MustCompile(`[0-9]`).FindAllString(v, -1)) != 8 {
		t.Errorf("generateValue: got invalid password (wrong digit count): %s", v)
	}
	if len(regexp.MustCompile(`[`+Symbols+`]`).FindAllString(v, -1)) != 8 {
		t.Errorf("generateValue: got invalid password (wrong symbol count): %s", v)
	}

	// password without arguments
	v, err = generateValue("password")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9` + Symbols + `]{32}$`).MatchString(v) {
		t.Errorf("generateValue: got invalid password (wrong length): %s", v)
	}
	if len(regexp.MustCompile(`[A-Za-z]`).FindAllString(v, -1)) != 16 {
		t.Errorf("generateValue: got invalid password (wrong letter count): %s", v)
	}
	if len(regexp.MustCompile(`[0-9]`).FindAllString(v, -1)) != 8 {
		t.Errorf("generateValue: got invalid password (wrong digit count): %s", v)
	}
	if len(regexp.MustCompile(`[`+Symbols+`]`).FindAllString(v, -1)) != 8 {
		t.Errorf("generateValue: got invalid password (wrong symbol count): %s", v)
	}

	// password with arguments
	symbols := "_-"
	v, err = generateValue("password:length=20;num_digits=3;num_symbols=4;symbols=" + symbols)
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9` + symbols + `]{20}$`).MatchString(v) {
		t.Errorf("generateValue: got invalid password (wrong length): %s", v)
	}
	if len(regexp.MustCompile(`[A-Za-z]`).FindAllString(v, -1)) != 13 {
		t.Errorf("generateValue: got invalid password (wrong letter count): %s", v)
	}
	if len(regexp.MustCompile(`[0-9]`).FindAllString(v, -1)) != 3 {
		t.Errorf("generateValue: got invalid password (wrong digit count): %s", v)
	}
	if len(regexp.MustCompile(`[`+symbols+`]`).FindAllString(v, -1)) != 4 {
		t.Errorf("generateValue: got invalid password (wrong symbol count): %s", v)
	}

	// password with base32 encoding
	v, err = generateValue("password:length=5;num_digits=0;num_symbols=5;symbols=_;encoding=base32")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	if v != "L5PV6X27" {
		// L5PV6X27 is base32 of _____
		t.Errorf("generateValue: got invalid password (invalid base64 encoding): %s", v)
	}

	// password with base64 encoding
	v, err = generateValue("password:length=5;num_digits=0;num_symbols=5;symbols=_;encoding=base64")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	if v != "X19fX18=" {
		// X19fX18= is base64 of _____
		t.Errorf("generateValue: got invalid password (invalid base64 encoding): %s", v)
	}

	// password with base64_raw (without padding) encoding
	v, err = generateValue("password:length=5;num_digits=0;num_symbols=5;symbols=_;encoding=base64_raw")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	if v != "X19fX18" {
		// X19fX18 is base64_raw of _____
		t.Errorf("generateValue: got invalid password (invalid base64 encoding): %s", v)
	}

	// uuid
	v, err = generateValue("uuid")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	if _, err := uuid.Parse(string(v)); err != nil {
		t.Errorf("generateValue: got invalid uuid; error: %s", err)
	}

	// uuid encoding base32
	v, err = generateValue("uuid:encoding=base32")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	var decodedUuidBytes []byte
	decodedUuidBytes, _ = base32.StdEncoding.DecodeString(v)
	if _, err := uuid.FromBytes(decodedUuidBytes); err != nil {
		t.Errorf("generateValue: got invalid uuid; error: %s", err)
	}

	// uuid encoding base64
	v, err = generateValue("uuid:encoding=base64")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	decodedUuidBytes, _ = base64.StdEncoding.DecodeString(v)
	if _, err := uuid.FromBytes(decodedUuidBytes); err != nil {
		t.Errorf("generateValue: got invalid uuid; error: %s", err)
	}

	// uuid encoding base64 url
	v, err = generateValue("uuid:encoding=base64_url")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	decodedUuidBytes, _ = base64.URLEncoding.DecodeString(v)
	if _, err := uuid.FromBytes(decodedUuidBytes); err != nil {
		t.Errorf("generateValue: got invalid uuid; error: %s", err)
	}

	// uuid encoding base64_raw (without padding)
	v, err = generateValue("uuid:encoding=base64_raw")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	decodedUuidBytes, _ = base64.RawStdEncoding.DecodeString(v)
	if _, err := uuid.FromBytes(decodedUuidBytes); err != nil {
		t.Errorf("generateValue: got invalid uuid; error: %s", err)
	}

	// uuid encoding base64_raw url (without padding)
	v, err = generateValue("uuid:encoding=base64_raw_url")
	if err != nil {
		t.Fatalf("generateValue: got errror: %s", err)
	}
	decodedUuidBytes, _ = base64.RawURLEncoding.DecodeString(v)
	if _, err := uuid.FromBytes(decodedUuidBytes); err != nil {
		t.Errorf("generateValue: got invalid uuid; error: %s", err)
	}

}

func TestGenerateValueWithError(t *testing.T) {
	var err error

	// invalid generator
	_, err = generateValue("foobar")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}

	// invalid password argument
	_, err = generateValue("password:foo=bar")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}

	// invalid password argument: length
	_, err = generateValue("password:length=foo")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}

	// invalid password argument: number of digits
	_, err = generateValue("password:num_digits=foo")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}

	// invalid password argument: number of symbols
	_, err = generateValue("password:num_symbols=foo")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}

	// invalid password argument: number of symbols
	_, err = generateValue("password:symbols=foo")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}

	// invalid password argument: encoding
	_, err = generateValue("password:encoding=foo")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}

	// error during password generation: too many digits/symbols (symbols will default to 4/4 = 1 here)
	_, err = generateValue("password:length=4;num_digits=4")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}

	// invalid uuid argument
	_, err = generateValue("uuid:foo=bar")
	if err == nil {
		t.Error("generateValue: expected error, but got none")
	} else {
		t.Logf("ok; got error: %s", err)
	}
}
