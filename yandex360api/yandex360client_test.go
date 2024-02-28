package yandex360api

import (
	"context"
	"net/url"
	"testing"

	//"github.com/boryashkin/cert-manager-webhook-beget/yandex360api"
	//	yandex360api "github.com/cert-manager/webhook-example/client"

	"github.com/stretchr/testify/suite"
)

type ApiClientTestSuite struct {
	suite.Suite
	yandex360api *Yandex360ApiMock
	client       *ApiClient
	apiUrl       *url.URL
}

func (suite *ApiClientTestSuite) SetupTest() {
	suite.yandex360api = NewYandex360ApiMock(
		Yandex360ApiMock_TestData,
	)
	go func() {
		suite.yandex360api.Run(":12943")
	}()
	apiUrl, err := url.Parse("http://localhost:12943")
	suite.Require().NoError(err)

	suite.apiUrl = apiUrl
	suite.client = NewApiClient()
}

func (suite *ApiClientTestSuite) TearDownSuite() {
	suite.yandex360api.Stop(context.TODO())
}

func TestApiClientTestSuiteSuite(t *testing.T) {
	suite.Run(t, new(ApiClientTestSuite))
}

func (suite *ApiClientTestSuite) TestApiClient_GetDnsRecords() {
	_, err := suite.client.GetDnsRecords(&ApiSettings{ApiUrl: suite.apiUrl, OrganizationId: 1001, Domain: "example1.com", Token: Yandex360ApiMock_TestData.authKey})
	suite.Require().NoError(err, "getData returned an err %s", err)
}

func (suite *ApiClientTestSuite) TestApiClient_AddTxtRecord() {

	// basic add
	err := suite.client.AddTxtRecord(&ApiSettings{ApiUrl: suite.apiUrl, OrganizationId: 1001, Domain: "example1.com", Token: Yandex360ApiMock_TestData.authKey}, "sometxt10", "sometxtvalue", 300)
	suite.Require().NoError(err, "AddTxtRecord returned error")
}

func (suite *ApiClientTestSuite) TestApiClient_DeleteTxtRecordByName() {

	// fail if not found
	err := suite.client.DeleteTxtRecordByName(&ApiSettings{ApiUrl: suite.apiUrl, OrganizationId: 1001, Domain: "example1.com", Token: Yandex360ApiMock_TestData.authKey}, "sometxt0")
	suite.Require().Error(err, "DeleteTxtRecordByName 1 not returned error")

	// not delete cname
	err = suite.client.DeleteTxtRecordByName(&ApiSettings{ApiUrl: suite.apiUrl, OrganizationId: 1001, Domain: "example1.com", Token: Yandex360ApiMock_TestData.authKey}, "cname")
	suite.Require().Error(err, "DeleteTxtRecordByName 2 not returned error")

	// delete txt
	err = suite.client.DeleteTxtRecordByName(&ApiSettings{ApiUrl: suite.apiUrl, OrganizationId: 1001, Domain: "example1.com", Token: Yandex360ApiMock_TestData.authKey}, "sometxt1")
	suite.Require().NoError(err, "DeleteTxtRecordByName 3 returned an err %s", err)
}

func (suite *ApiClientTestSuite) TestApiClient_AddRecords() {
	r := DnsRecord{
		Name: "txt1",
		Text: "txtValue",
		Type: "TXT",
		TTL:  21600,
	}
	err := suite.client.AddDnsRecord(&ApiSettings{ApiUrl: suite.apiUrl, OrganizationId: 1001, Domain: "example1.com", Token: Yandex360ApiMock_TestData.authKey}, r)
	suite.Require().NoError(err, "getData returned an err %s", err)
}

func (suite *ApiClientTestSuite) TestApiClient_DeleteRecord() {
	err := suite.client.DeleteDnsRecord(&ApiSettings{ApiUrl: suite.apiUrl, OrganizationId: 1001, Domain: "example1.com", Token: Yandex360ApiMock_TestData.authKey}, 4)
	suite.Require().NoError(err, "getData returned an err %s", err)
}
