/*
SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and secret-generator contributors
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2"

	"github.com/sap/admission-webhook-runtime/pkg/admission"

	"github.com/sap/secret-generator/internal/webhook"
)

func main() {
	pflag.CommandLine.AddGoFlagSet(admission.FlagSet())
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.SortFlags = false
	pflag.Parse()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		klog.Fatal(errors.Wrap(err, "error populating corev1 scheme"))
	}
	webhook := webhook.NewSecretWebhook()
	if err := admission.RegisterMutatingWebhook[*corev1.Secret](webhook, scheme, klog.NewKlogr()); err != nil {
		klog.Fatal(errors.Wrapf(err, "error registering webhook for corev1.Secret"))
	}
	if err := admission.Serve(context.Background(), nil); err != nil {
		klog.Fatal(errors.Wrap(err, "error running http server"))
	}
}
