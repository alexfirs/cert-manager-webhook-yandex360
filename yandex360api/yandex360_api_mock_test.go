package yandex360api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const baseUrl = "http://localhost:8489/directory/v1/org/"

type yandex360apiMockTestSuite struct {
	suite.Suite
	yandex360api *Yandex360ApiMock
	client       *http.Client
}

func (suite *yandex360apiMockTestSuite) SetupTest() {
	suite.yandex360api = NewYandex360ApiMock(Yandex360ApiMock_TestData)
	suite.client = http.DefaultClient

	go func() {
		suite.yandex360api.Run(":8489")
	}()
}

func (suite *yandex360apiMockTestSuite) TearDownSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	suite.yandex360api.Stop(ctx)
}

func TestYandex360apiMockTestSuite(t *testing.T) {
	suite.Run(t, new(yandex360apiMockTestSuite))
}

func (suite *yandex360apiMockTestSuite) TestYandex360apiMock_GetDnsRecords_Validation() {

	// invalid url
	suite.requestAndValidateGetDnsEntries("INVALIDORG", "example1.com", false, http.StatusNotFound)
	suite.requestAndValidateGetDnsEntries("INVALIDORG", "example1.com", true, http.StatusNotFound)

	// no access token
	suite.requestAndValidateGetDnsEntries("1001", "example1.com", false, http.StatusUnauthorized)

	// unauthorized organization
	suite.requestAndValidateGetDnsEntries("1000", "example1.com", true, http.StatusUnauthorized)

	// unauthorized domain
	suite.requestAndValidateGetDnsEntries("1001", "example0.com", true, http.StatusUnauthorized)

	//invalid domain for different organization
	suite.requestAndValidateGetDnsEntries("1003", "example3.com", true, http.StatusUnauthorized)
	suite.requestAndValidateGetDnsEntries("1003", "example1.com", true, http.StatusUnauthorized)

	suite.requestAndValidateGetDnsEntries("1001", "example1.com", true, http.StatusOK)
}

func (suite *yandex360apiMockTestSuite) TestYandex360apiMock_GetDnsRecords_TestData() {
	orgId := 1001
	domain := "example1.com"

	// good request, default paging
	rsp := suite.requestListData(orgId, domain, 1, 10)

	suite.Require().Equal(1, rsp.Page)
	suite.Require().Equal(1, rsp.Pages)
	suite.Require().Equal(10, rsp.PerPage)
	suite.Require().Equal(3, rsp.Total)
	suite.Require().Equal(3, len(rsp.Records))

	// good request, custom paging
	rsp = suite.requestListData(orgId, domain, 2, 2)

	suite.Require().Equal(2, rsp.Page)
	suite.Require().Equal(2, rsp.Pages)
	suite.Require().Equal(2, rsp.PerPage)
	suite.Require().Equal(3, rsp.Total)
	suite.Require().Equal(1, len(rsp.Records))
}

func (suite *yandex360apiMockTestSuite) TestYandex360apiMock_DeleteRecord() {
	orgId := 1001
	domain := "example1.com"

	// wrong method
	suite.requestDelete("GET", orgId, domain, 1, http.StatusMethodNotAllowed)

	// nonexisting record
	suite.requestDelete("DELETE", orgId, domain, 0, http.StatusNotFound)

	// deletion test
	rsp := suite.requestListData(orgId, domain, 1, 10)
	domainsCount := len(rsp.Records)

	suite.requestDelete("DELETE", orgId, domain, 1, http.StatusOK)

	rsp = suite.requestListData(orgId, domain, 1, 10)
	suite.Require().Equal(domainsCount-1, len(rsp.Records))
}

func (suite *yandex360apiMockTestSuite) TestYandex360apiMock_AddRecord() {
	orgId := 1001
	domain := "example1.com"

	rsp := suite.requestListData(orgId, domain, 1, 10)
	domainsCount := len(rsp.Records)

	suite.requestAdd("POST", orgId, domain, DnsRecord{Type: "TXT", Name: "_text_record", Text: "TxtValue", TTL: 21600}, http.StatusOK)

	rsp = suite.requestListData(orgId, domain, 1, 10)
	suite.Require().Equal(domainsCount+1, len(rsp.Records))
	addedDnsRecord := rsp.Records[len(rsp.Records)-1]
	suite.Require().Equal("TXT", addedDnsRecord.Type)
	suite.Require().Equal(21600, addedDnsRecord.TTL)
	suite.Require().Equal("_text_record", addedDnsRecord.Name)
	suite.Require().Equal("TxtValue", addedDnsRecord.Text)

}

func (suite *yandex360apiMockTestSuite) requestAdd(method string, orgId int, domain string, dnsRecord DnsRecord, expectedCode int) {
	body, _ := json.Marshal(dnsRecord)
	req, _ := http.NewRequest(method, baseUrl+strconv.Itoa(orgId)+"/domains/"+domain+"/dns", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "OAuth "+Yandex360ApiMock_TestData.authKey)

	r, err := suite.client.Do(req)
	suite.Require().NoError(err)
	suite.Require().Equal(expectedCode, r.StatusCode)
}

func (suite *yandex360apiMockTestSuite) requestDelete(method string, orgId int, domain string, recordId int, expectedCode int) {
	req, _ := http.NewRequest(method, baseUrl+strconv.Itoa(orgId)+"/domains/"+domain+"/dns/"+strconv.Itoa(recordId), nil)
	req.Header.Set("Authorization", "OAuth "+Yandex360ApiMock_TestData.authKey)
	r, err := suite.client.Do(req)
	suite.Require().NoError(err)
	suite.Require().Equal(expectedCode, r.StatusCode)
}

func (suite *yandex360apiMockTestSuite) requestAndValidateGetDnsEntries(orgIdString string, domain string, passToken bool, expectedStatusCode int) {
	req, _ := http.NewRequest("GET", baseUrl+orgIdString+"/domains/"+domain+"/dns", nil)
	if passToken {
		req.Header.Set("Authorization", "OAuth "+Yandex360ApiMock_TestData.authKey)
	}
	r, err := suite.client.Do(req)
	suite.Require().NoError(err)
	suite.Require().Equal(expectedStatusCode, r.StatusCode)
}

func (suite *yandex360apiMockTestSuite) requestListData(orgId int, domain string, page int, perPage int) GetDataResponse {
	var rsp GetDataResponse

	req, _ := http.NewRequest("GET", baseUrl+strconv.Itoa(orgId)+"/domains/"+domain+"/dns?page="+strconv.Itoa(page)+"&perPage="+strconv.Itoa(perPage)+"", nil)
	req.Header.Set("Authorization", "OAuth "+Yandex360ApiMock_TestData.authKey)
	r, err := suite.client.Do(req)
	suite.Require().NoError(err)
	suite.Require().Equal(http.StatusOK, r.StatusCode)

	bdy, err := io.ReadAll(r.Body)
	suite.Require().NoError(err)
	err = json.Unmarshal(bdy, &rsp)
	suite.Require().NoError(err)
	return rsp
}
