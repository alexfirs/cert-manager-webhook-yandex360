package main

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/alexfirs/cert-manager-webhook-yandex360/yandex360api"
	acmetest "github.com/cert-manager/cert-manager/test/acme"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
)

/*
func SetupTest(t *testing.T) {
	if os.Getenv("TEST_ASSET_ETCD") == "" {
		os.Setenv("TEST_ASSET_ETCD", "_test/kubebuilder/bin/etcd")
		defer os.Unsetenv("TEST_ASSET_ETCD")
	}
	if os.Getenv("TEST_ASSET_KUBE_APISERVER") == "" {
		os.Setenv("TEST_ASSET_KUBE_APISERVER", "_test/kubebuilder/bin/kube-apiserver")
		defer os.Unsetenv("TEST_ASSET_KUBE_APISERVER")
	}
}
*/

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	//

	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	yandex360apiMockUrl, err := url.Parse("http://localhost:60001")
	if err != nil {
		t.FailNow()
	}

	api := yandex360api.NewYandex360ApiMock(yandex360api.Yandex360ApiMock_TestData)
	go func() {
		api.Run(":60001")
		t.Log("run")
	}()
	go func() {
		api.RunDns("59351")
		t.Log("run dns")
	}()
	defer func() {
		api.Stop(context.TODO())
		api.StopDns(context.TODO())
		t.Log("stopped servers")
	}()

	solver := New(yandex360apiMockUrl)

	fixture := acmetest.NewFixture(solver,
		acmetest.SetResolvedZone(zone),
		acmetest.SetAllowAmbientCredentials(false),
		acmetest.SetManifestPath("testdata/yandex360"),
		acmetest.SetDNSServer("127.0.0.1:59351"),
		acmetest.SetUseAuthoritative(false),
	)

	// fixture := acmetest.NewFixture(solver,
	// 	acmetest.SetResolvedZone("example.com."),
	// 	acmetest.SetManifestPath("testdata/my-custom-solver"),
	// 	acmetest.SetDNSServer("127.0.0.1:59351"),
	// 	acmetest.SetUseAuthoritative(false),
	// )
	//need to uncomment and  RunConformance delete runBasic and runExtended once https://github.com/cert-manager/cert-manager/pull/4835 is merged
	//fixture.RunConformance(t)
	fixture.RunBasic(t)
	fixture.RunExtended(t)

}
