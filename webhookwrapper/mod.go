package webhookwrapper

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"

	"github.com/go-logr/logr"

	ctrl "sigs.k8s.io/controller-runtime"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type WebhookRegistrator struct {
	Registrator         func(ctrl.Manager) error
	Name                string
	RulesWithOperations []admissionregistrationv1.RuleWithOperations
	Mutating            bool
	WebhookPath         string
}
type WebhookRegistrators = []WebhookRegistrator

func NewManager(options ctrl.Options, logger logr.Logger, developmentMode bool, webhookAdvertiseHost string, webhooks WebhookRegistrators) (ctrl.Manager, error) {
	var config *restclient.Config = nil
	var clientset *kubernetes.Clientset = nil
	caCertificate := ""

	if developmentMode {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err.Error())
		}

		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", path.Join(userHomeDir, ".kube", "config"))
		if err != nil {
			panic(err.Error())
		}

		// create the clientset
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		// create certificates
		certDir := ""
		caCertificate, certDir, err = createCertificates(logger, []string{webhookAdvertiseHost})
		if err != nil {
			logger.Error(err, "unable to create certificates")
			os.Exit(1)
		}

		options.CertDir = certDir
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		logger.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := registerWebhooks(logger, mgr, webhookAdvertiseHost, developmentMode, clientset, caCertificate, webhooks, options); err != nil {
		logger.Error(err, "unable to register webhooks")
		os.Exit(1)
	}

	return mgr, nil
}

func createCertificates(logger logr.Logger, subjectAlternativeNames []string) (string, string, error) {
	dir, err := ioutil.TempDir(os.TempDir(), "webhookwrapper-certs-*")
	if err != nil {
		logger.Error(err, "unable to create temporary directory")
		os.Exit(1)
	}
	certificateAuthority, err := newCertificateAuthority("webhookwrapper")
	if err != nil {
		logger.Error(err, "unable to create certificate authority")
		os.Exit(1)
	}
	certificate, certificateKey, err := certificateAuthority.createCertificate(
		"webhookwrapper",
		[]string{},
		subjectAlternativeNames,
	)
	if err != nil {
		logger.Error(err, "unable to create certificate")
		os.Exit(1)
	}
	if err := ioutil.WriteFile(path.Join(dir, "tls.key"), []byte(certificateKey), 0600); err != nil {
		logger.Error(err, "unable to write certificate key")
		os.Exit(1)
	}
	if err := ioutil.WriteFile(path.Join(dir, "tls.crt"), []byte(certificate), 0600); err != nil {
		logger.Error(err, "unable to write certificate")
		os.Exit(1)
	}
	return certificate, dir, nil
}

func registerWebhooks(logger logr.Logger, mgr ctrl.Manager, webhookAdvertiseHost string, developmentMode bool, clientset *kubernetes.Clientset, caCertificate string, webhookRegistrators WebhookRegistrators, managerOptions ctrl.Options) error {
	ctx := context.TODO()
	for _, webhookRegistrator := range webhookRegistrators {
		if err := webhookRegistrator.Registrator(mgr); err != nil {
			logger.Error(err, "unable to create webhook")
		}
		if !developmentMode {
			continue
		}
		failurePolicy := admissionregistrationv1.Fail
		sideEffectsNone := admissionregistrationv1.SideEffectClassNone
		webhookEndpoint := url.URL{
			Scheme: "https",
			Host:   fmt.Sprintf("%s:%d", webhookAdvertiseHost, managerOptions.Port),
			Path:   webhookRegistrator.WebhookPath,
		}
		webhookEndpointString := webhookEndpoint.String()
		if err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, webhookRegistrator.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to cleanup existing webhook")
			os.Exit(1)
		}
		if err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, webhookRegistrator.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to cleanup existing webhook")
			os.Exit(1)
		}
		if webhookRegistrator.Mutating {
			_, err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(
				ctx,
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: webhookRegistrator.Name,
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{
						{
							Name: webhookRegistrator.Name,
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								URL:      &webhookEndpointString,
								CABundle: []byte(caCertificate),
							},
							Rules:                   webhookRegistrator.RulesWithOperations,
							FailurePolicy:           &failurePolicy,
							SideEffects:             &sideEffectsNone,
							AdmissionReviewVersions: []string{"v1"},
						},
					},
				},
				metav1.CreateOptions{},
			)
			if err != nil {
				logger.Error(err, "unable to register webhook")
				os.Exit(1)
			}
		} else {
			_, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(
				ctx,
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: webhookRegistrator.Name,
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{
						{
							Name: webhookRegistrator.Name,
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								URL:      &webhookEndpointString,
								CABundle: []byte(caCertificate),
							},
							Rules:                   webhookRegistrator.RulesWithOperations,
							FailurePolicy:           &failurePolicy,
							SideEffects:             &sideEffectsNone,
							AdmissionReviewVersions: []string{"v1"},
						},
					},
				},
				metav1.CreateOptions{},
			)
			if err != nil {
				logger.Error(err, "unable to register webhook")
				os.Exit(1)
			}
		}
	}
	return nil
}
