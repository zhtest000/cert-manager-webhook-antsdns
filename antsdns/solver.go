package antsdns

import (
	"context"
	"fmt"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/jetstack/cert-manager/pkg/acme/webhook"
	v1alpha1 "github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	cmmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/jetstack/cert-manager/pkg/issuer/acme/dns/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func NewSolver() webhook.Solver {
	return &Solver{}
}

// Solver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/jetstack/cert-manager/pkg/acme/webhook.Solver`
// interface.
type Solver struct {
	client *kubernetes.Clientset
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (s *Solver) Name() string {
	return "antsdns"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (s *Solver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("Presenting txt record: %v %v", ch.ResolvedFQDN, ch.ResolvedZone)
	klog.Infof("top-value:%v,dns_name:%v",ch.Key,ch.DNSName)
	antsClient, err := s.newClientFromChallenge(ch)
	if err != nil {
		klog.Errorf("New client from challenge error: %v", err)
		return err
	}
	top, domain := s.getDomainAndEntry(ch)

	present, err := antsClient.HasTxtRecord(&domain, &top)
	if present {
		klog.Infof("update-txt-record")
		err := antsClient.UpdateTxtRecord(&domain, &top, &ch.Key, 600)
		if err != nil {
			return fmt.Errorf("unable to change TXT record: %v", err)
		}
	} else {
		klog.Infof("create-txt-record")
		err := antsClient.CreateTxtRecord(&domain, &top, &ch.Key, 600)
		if err != nil {
			return fmt.Errorf("unable to create TXT record: %v", err)
		}
	}
	klog.Infof("Presented txt record %v", ch.ResolvedFQDN)
	return nil
}

func (s *Solver) newClientFromChallenge(ch *v1alpha1.ChallengeRequest) (*antsClient, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Infof("loadConfig error ")
		return nil, err
	}
	AppId, err := s.getSecretData(cfg.AppIdSecretRef, ch.ResourceNamespace)
	if err != nil {
		klog.Infof("getSecretData AppId error ")
		return nil, err
	}

	AppKey, err := s.getSecretData(cfg.AppKeySecretRef, ch.ResourceNamespace)
	if err != nil {
		klog.Infof("getSecretData AppKey error ")
		return nil, err
	}
	klog.Infof("Decoded config: %s, %s,%s",cfg.IspAddress, string(AppId),string(AppKey))
	antsClient := newClient(cfg.IspAddress,string(AppId),string(AppKey))

	return antsClient, nil
}

func (s *Solver) getCredential(cfg *Config, ns string) (*credentials.AccessKeyCredential, error) {
	//accessKey, err := s.getSecretData(cfg.AppId, ns)
	//if err != nil {
	//	return nil, err
	//}
	//
	//secretKey, err := s.getSecretData(cfg.SecretKeySecretRef, ns)
	//if err != nil {
	//	return nil, err
	//}
	//
	//return credentials.NewAccessKeyCredential(string(accessKey), string(secretKey)), nil
	return nil,nil
}

func (s *Solver) getSecretData(selector cmmetav1.SecretKeySelector, ns string) ([]byte, error) {
	secret, err := s.client.CoreV1().Secrets(ns).Get(context.TODO(),selector.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load secret %q", ns+"/"+selector.Name)
	}

	if data, ok := secret.Data[selector.Key]; ok {
		return data, nil
	}

	return nil, errors.Errorf("no key %q in secret %q", selector.Key, ns+"/"+selector.Name)
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (s *Solver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("1 Cleaning up txt record: %v %v", ch.ResolvedFQDN, ch.ResolvedZone)
	antsClient, err := s.newClientFromChallenge(ch)
	if err != nil {
		klog.Errorf("New client from challenge error: %v", err)
		return err
	}
	entry, domain := s.getDomainAndEntry(ch)
	klog.Infof("entry,domain=: %v %v",entry, domain)
	present, err := antsClient.HasTxtRecord(&domain, &entry)

	if present {
		klog.Infof("HasTxt Record-->deleting entry=%s, domain=%s", entry, domain)
		err := antsClient.DeleteTxtRecord(&domain, &entry)
		if err != nil {
			return fmt.Errorf("unable to remove TXT record: %v", err)
		}
	}

	klog.Infof("2 Cleaned up txt record: %v %v", ch.ResolvedFQDN, ch.ResolvedZone)
	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
//
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
//
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (s *Solver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	s.client = cl
	return nil
}

func extractRR(fqdn, domain string) string {
	name := util.UnFqdn(fqdn)
	if idx := strings.Index(name, "."+domain); idx != -1 {
		return name[:idx]
	}

	return name
}

func (s *Solver) getDomainAndEntry(ch *v1alpha1.ChallengeRequest) (string, string) {
	// Both ch.ResolvedZone and ch.ResolvedFQDN end with a dot: '.'
	top := strings.TrimSuffix(ch.ResolvedFQDN, ch.ResolvedZone)
	top = strings.TrimSuffix(top, ".")
	domain := strings.TrimSuffix(ch.ResolvedZone, ".")
	return top, domain
}