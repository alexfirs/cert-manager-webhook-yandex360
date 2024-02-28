package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	certmgrapiv1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	"github.com/alexfirs/cert-manager-webhook-yandex360/yandex360api"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		New(),
	)
}

// yandex360DNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/cert-manager/cert-manager/pkg/acme/webhook.Solver`
// interface.
type yandex360DNSSolver struct {
	name      string
	apiClient *yandex360api.ApiClient
	k8sClient *kubernetes.Clientset
}

// customDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type yandex360DNSProviderConfig struct {
	Endpoint          string                         `json:"endpoint"`
	OrganizationId    int                            `json:"organizationId"`
	APITokenSecretRef certmgrapiv1.SecretKeySelector `json:"apiTokenSecretRef"`
	TTL               int                            `json:"ttl"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (y *yandex360DNSSolver) Name() string {
	return y.name
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (y *yandex360DNSSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	var chString string
	if ch != nil {
		chString = fmt.Sprintf("rn: %s, rz: %s, rfqdn: %s, dnsn: %s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN, ch.DNSName)
	}

	klog.Infof("solver.present: ch.: %s", chString)

	apiSettings, err := y.getApiSettingsForChallengeRequest(ch)
	if err != nil {
		return err
	}
	klog.Infof("solver.present: after getApiSettingsForChallengeRequest: api: %s, orgId:%d, ttl:%d, token len:%d ", apiSettings.ApiUrl, apiSettings.OrganizationId, apiSettings.TTL, len(apiSettings.Token))

	name := strings.TrimSuffix(ch.ResolvedFQDN, "."+apiSettings.Domain+".")
	err = y.apiClient.AddTxtRecord(apiSettings, name, ch.Key, apiSettings.TTL)
	if err != nil {
		return err
	}
	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (y *yandex360DNSSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	apiSettings, err := y.getApiSettingsForChallengeRequest(ch)
	if err != nil {
		return err
	}

	name := strings.TrimSuffix(ch.ResolvedFQDN, "."+apiSettings.Domain+".")
	err = y.apiClient.DeleteTxtRecordByName(apiSettings, name)
	if err != nil {
		return err
	}
	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (y *yandex360DNSSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {

	if y.k8sClient != nil {
		return nil
	}

	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	y.k8sClient = cl
	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (yandex360DNSProviderConfig, error) {
	cfg := yandex360DNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func (y *yandex360DNSSolver) getApiSettingsForChallengeRequest(ch *v1alpha1.ChallengeRequest) (*yandex360api.ApiSettings, error) {
	var chString string
	if ch != nil {
		chString = fmt.Sprintf("rn: %s, rz: %s, rfqdn: %s, dnsn: %s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN, ch.DNSName)
	}

	klog.Infof("solver.getApiSettingsForChallengeRequest ch.: %s", chString)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}

	apiUrl, err := url.Parse(cfg.Endpoint)

	if err != nil {
		return nil, err
	}

	token, err := y.secret(cfg.APITokenSecretRef, ch.ResourceNamespace)
	if err != nil {
		return nil, err
	}

	domain := getDomainFromZone(ch.ResolvedZone)

	ttl := 300
	if cfg.TTL > 0 {
		ttl = cfg.TTL
	}

	apiSettings := &yandex360api.ApiSettings{ApiUrl: apiUrl, Token: token, OrganizationId: cfg.OrganizationId, Domain: domain, TTL: ttl}
	klog.Infof("solver.getApiSettingsForChallengeRequest ch.: %s, api:%s, token len:%d, orgId:%d, domain:%s, ttl:%d ", chString, apiUrl, len(token), cfg.OrganizationId, domain, ttl)
	return apiSettings, nil
}

func (s *yandex360DNSSolver) secret(ref certmgrapiv1.SecretKeySelector, namespace string) (string, error) {
	klog.Infof("solver.secret name:%s", ref.Name)
	if ref.Name == "" {
		return "", nil
	}

	secret, err := s.k8sClient.CoreV1().Secrets(namespace).Get(context.TODO(), ref.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("solver.secret: calling k8s: %v", err)
		return "", err
	}

	bytes, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key not found %q in secret '%s/%s'", ref.Key, namespace, ref.Name)
	}
	return strings.TrimSuffix(string(bytes), "\n"), nil
}

func New() webhook.Solver {
	e := &yandex360DNSSolver{
		name:      "yandex360-dns-solver",
		apiClient: yandex360api.NewApiClient(),
	}
	return e
}

func getDomainFromZone(zone string) string {
	parts := strings.Split(zone[0:len(zone)-1], ".")
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}
