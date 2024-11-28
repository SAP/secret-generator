/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and secret-generator contributors
SPDX-License-Identifier: Apache-2.0
*/

package webhook_test

import (
	"context"
	"crypto/tls"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"maps"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/sap/admission-webhook-runtime/pkg/admission"

	"github.com/sap/secret-generator/internal/webhook"
)

func TestWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}

var testEnv *envtest.Environment
var cfg *rest.Config
var ctx context.Context
var cancel context.CancelFunc
var threads sync.WaitGroup
var clientset kubernetes.Interface

const testingNamespace = "testing"

var _ = BeforeSuite(func() {
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())
	var err error

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			MutatingWebhooks: []*admissionv1.MutatingWebhookConfiguration{
				buildMutatingWebhookConfiguration(),
			},
		},
	}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	webhookInstallOptions := &testEnv.WebhookInstallOptions

	By("initializing kubernetes clientset")
	clientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	By("initializing webhook scheme")
	scheme := runtime.NewScheme()
	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	By("registering webhook")
	err = admission.RegisterMutatingWebhook[*corev1.Secret](webhook.NewSecretWebhook(), scheme, log.Log)
	Expect(err).NotTo(HaveOccurred())

	By("starting webhook server")
	threads.Add(1)
	go func() {
		defer threads.Done()
		defer GinkgoRecover()
		options := &admission.ServeOptions{
			BindAddress: fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort),
			CertFile:    webhookInstallOptions.LocalServingCertDir + "/tls.crt",
			KeyFile:     webhookInstallOptions.LocalServingCertDir + "/tls.key",
		}
		err := admission.Serve(ctx, options)
		Expect(err).NotTo(HaveOccurred())
	}()

	By("waiting for webhook server to become ready")
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())

	By("creating testing namespace")
	_, err = clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testingNamespace,
		},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	threads.Wait()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Create secrets", func() {
	var err error

	It("should generate correct secret values", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
			StringData: map[string]string{
				"regularKey":              "regularValue",
				"uuidKey":                 "%generate:uuid",
				"base32UuidKey":           "%generate:uuid:encoding=base32",
				"base64UuidKey":           "%generate:uuid:encoding=base64",
				"base64UrlUuidKey":        "%generate:uuid:encoding=base64_url",
				"base64RawUuidKey":        "%generate:uuid:encoding=base64_raw",
				"base64RawUrlUuidKey":     "%generate:uuid:encoding=base64_raw_url",
				"simplePasswordKey":       "%generate:password",
				"complexPasswordKey":      "%generate:password:length=10;num_digits=1;num_symbols=1;symbols=_",
				"base32PasswordKey":       "%generate:password:length=100;encoding=base32",
				"base64PasswordKey":       "%generate:password:length=100;encoding=base64",
				"base64UrlPasswordKey":    "%generate:password:length=100;encoding=base64_url",
				"base64RawPasswordKey":    "%generate:password:length=100;encoding=base64_raw",
				"base64RawUrlPasswordKey": "%generate:password:length=100;encoding=base64_raw_url",
			},
		}

		secret, err = clientset.CoreV1().Secrets(testingNamespace).Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKeyWithValue("regularKey", []byte("regularValue")))

		Expect(secret.Data).To(HaveKey("uuidKey"))
		_, err = uuid.ParseBytes(secret.Data["uuidKey"])
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base32UuidKey"))
		Expect(secret.Data["base32UuidKey"]).NotTo(BeEmpty())
		_, err = base32.StdEncoding.DecodeString(string(secret.Data["base32UuidKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base64UuidKey"))
		Expect(secret.Data["base64UuidKey"]).NotTo(BeEmpty())
		_, err = base64.StdEncoding.DecodeString(string(secret.Data["base64UuidKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base64UrlUuidKey"))
		Expect(secret.Data["base64UrlUuidKey"]).NotTo(BeEmpty())
		_, err = base64.URLEncoding.DecodeString(string(secret.Data["base64UrlUuidKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base64RawUuidKey"))
		Expect(secret.Data["base64RawUuidKey"]).NotTo(BeEmpty())
		_, err = base64.RawStdEncoding.DecodeString(string(secret.Data["base64RawUuidKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base64RawUrlUuidKey"))
		Expect(secret.Data["base64RawUrlUuidKey"]).NotTo(BeEmpty())
		_, err = base64.RawURLEncoding.DecodeString(string(secret.Data["base64RawUrlUuidKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("simplePasswordKey"))
		Expect(secret.Data["simplePasswordKey"]).To(HaveLen(32))

		Expect(secret.Data).To(HaveKey("complexPasswordKey"))
		Expect(secret.Data["complexPasswordKey"]).To(HaveLen(10))

		Expect(secret.Data).To(HaveKey("base32PasswordKey"))
		Expect(secret.Data["base32PasswordKey"]).NotTo(BeEmpty())
		_, err = base32.StdEncoding.DecodeString(string(secret.Data["base32PasswordKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base64PasswordKey"))
		Expect(secret.Data["base64PasswordKey"]).NotTo(BeEmpty())
		_, err = base64.StdEncoding.DecodeString(string(secret.Data["base64PasswordKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base64UrlPasswordKey"))
		Expect(secret.Data["base64UrlPasswordKey"]).NotTo(BeEmpty())
		_, err = base64.URLEncoding.DecodeString(string(secret.Data["base64UrlPasswordKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base64RawPasswordKey"))
		Expect(secret.Data["base64RawPasswordKey"]).NotTo(BeEmpty())
		_, err = base64.RawStdEncoding.DecodeString(string(secret.Data["xbase64RawPasswordKey"]))
		Expect(err).NotTo(HaveOccurred())

		Expect(secret.Data).To(HaveKey("base64RawUrlPasswordKey"))
		Expect(secret.Data["base64RawUrlPasswordKey"]).NotTo(BeEmpty())
		_, err = base64.RawURLEncoding.DecodeString(string(secret.Data["base64RawUrlPasswordKey"]))
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("Update secrets", func() {
	var specifiedSecret, createdSecret *corev1.Secret
	var err error

	BeforeEach(func() {
		specifiedSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
			Data: map[string][]byte{
				"regularKey":  []byte("regularValue"),
				"uuidKey":     []byte("%generate:uuid"),
				"passwordKey": []byte("%generate:password"),
			},
		}

		createdSecret, err = clientset.CoreV1().Secrets(testingNamespace).Create(ctx, specifiedSecret, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		specifiedSecret.GenerateName = ""
		specifiedSecret.Name = createdSecret.Name
		specifiedSecret.ResourceVersion = createdSecret.ResourceVersion
	})

	It("should update secrets correctly", func() {
		specifiedSecret.Data["regularKey"] = []byte("anotherValue")
		expectedData := maps.Clone(createdSecret.Data)
		expectedData["regularKey"] = specifiedSecret.Data["regularKey"]

		updatedSecret, err := clientset.CoreV1().Secrets(testingNamespace).Update(ctx, specifiedSecret, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedSecret.Data).To(Equal(expectedData))
	})
})

// assemble mutatingwebhookconfiguration descriptor
func buildMutatingWebhookConfiguration() *admissionv1.MutatingWebhookConfiguration {
	return &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mutate",
		},
		Webhooks: []admissionv1.MutatingWebhook{{
			Name:                    "mutate-secrets.test.local",
			AdmissionReviewVersions: []string{"v1"},
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Path: &[]string{"/core/v1/secret/mutate"}[0],
				},
			},
			Rules: []admissionv1.RuleWithOperations{{
				Operations: []admissionv1.OperationType{
					admissionv1.Create,
					admissionv1.Update,
				},
				Rule: admissionv1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"secrets"},
				},
			}},
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": testingNamespace,
				},
			},
			SideEffects: &[]admissionv1.SideEffectClass{admissionv1.SideEffectClassNone}[0],
		}},
	}
}
