/*
SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and secret-generator contributors
SPDX-License-Identifier: Apache-2.0
*/

package webhook

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type SecretWebhook struct{}

func NewSecretWebhook() *SecretWebhook {
	return &SecretWebhook{}
}

func (w *SecretWebhook) MutateCreate(ctx context.Context, secret *corev1.Secret) error {
	return handleCreateSecret(secret)
}

func (w *SecretWebhook) MutateUpdate(ctx context.Context, oldSecret *corev1.Secret, newSecret *corev1.Secret) error {
	return handleUpdateSecret(newSecret, oldSecret)
}
