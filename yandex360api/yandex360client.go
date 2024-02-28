package yandex360api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const TXTKey = "TXT"
const TXTDataKey = "txtdata"

func NewApiClient() *ApiClient {
	client := http.Client{}

	return &ApiClient{
		client: &client,
	}
}

func (a *ApiClient) AddTxtRecord(apiSettings *ApiSettings, name string, text string, ttl int) error {
	err := a.AddDnsRecord(apiSettings, DnsRecord{Name: name, Text: text, Type: "TXT", TTL: ttl})
	if err != nil {
		return fmt.Errorf("failed to AddTxtRecord: %v", err)
	}
	return err
}

func (a *ApiClient) GetDnsRecords(apiSettings *ApiSettings) ([]DnsRecord, error) {
	// TODO: add proper paging
	const page = 1
	const perPage = 50

	data, err := getDnsRecords(*a.client, *apiSettings.ApiUrl, apiSettings.Token, apiSettings.OrganizationId, apiSettings.Domain, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("failed to GetDnsRecords: %v", err)
	}

	return data.Records, nil
}

func (a *ApiClient) AddDnsRecord(apiSettings *ApiSettings, record DnsRecord) error {
	err := addDnsRecord(*a.client, *apiSettings.ApiUrl, apiSettings.Token, apiSettings.OrganizationId, apiSettings.Domain, record)
	if err != nil {
		return fmt.Errorf("failed to AddDnsRecord: %v", err)
	}

	return nil
}

func (a *ApiClient) DeleteTxtRecordByName(apiSettings *ApiSettings, name string) error {
	records, err := a.GetDnsRecords(apiSettings)
	if err != nil {
		return fmt.Errorf("DeleteDnsRecordByName: failed to getDnsRecords: %v", err)
	}

	recordId := -1
	for _, r := range records {
		if r.Name == name && r.Type == "TXT" {
			recordId = r.RecordID
			break
		}
	}
	if recordId == -1 {
		return fmt.Errorf("DeleteDnsRecordByName: failed to Find name %s: %v, data :%v", name, err, records)
	}

	err = deleteDnsRecord(*a.client, *apiSettings.ApiUrl, apiSettings.Token, apiSettings.OrganizationId, apiSettings.Domain, recordId)
	if err != nil {
		return fmt.Errorf("DeleteDnsRecordByName: failed to DeleteDnsRecord: %v", err)
	}
	return nil
}

func (a *ApiClient) DeleteDnsRecord(apiSettings *ApiSettings, recordId int) error {
	err := deleteDnsRecord(*a.client, *apiSettings.ApiUrl, apiSettings.Token, apiSettings.OrganizationId, apiSettings.Domain, recordId)
	if err != nil {
		return fmt.Errorf("failed to DeleteDnsRecord: %v", err)
	}
	return nil
}

func deleteDnsRecord(httpClient http.Client, apiUrl url.URL, token string, companyId int, domain string, recordId int) error {
	u := apiUrl
	u.Path += "/directory/v1/org/" + strconv.Itoa(companyId) + "/domains/" + domain + "/dns/" + strconv.Itoa(recordId)

	req, _ := http.NewRequest("DELETE", u.String(), nil)

	req.Header.Set("Authorization", "OAuth "+token)

	r, err := httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("delete failed: %d", r.StatusCode)
	}

	if r.StatusCode != 200 {
		bdy, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err == nil {
			return fmt.Errorf("response failed with status code: %d and body: %s", r.StatusCode, bdy)
		} else {
			return fmt.Errorf("response failed with status code: %d and unable parse body due to : %v", r.StatusCode, err)
		}

	}
	return nil
}

func addDnsRecord(httpClient http.Client, apiUrl url.URL, token string, companyId int, domain string, record DnsRecord) error {
	u := apiUrl
	u.Path += "/directory/v1/org/" + strconv.Itoa(companyId) + "/domains/" + domain + "/dns"

	jsonValue, _ := json.Marshal(record)

	req, _ := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonValue))

	req.Header.Set("Authorization", "OAuth "+token)

	r, err := httpClient.Do(req)

	if err != nil {
		if r != nil {
			return fmt.Errorf("post failed: %d, %v", r.StatusCode, err)
		} else {
			return fmt.Errorf("post failed: %v", err)
		}

	}

	if r.StatusCode != 200 {
		bdy, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err == nil {
			return fmt.Errorf("response failed with status code: %d and body: %s", r.StatusCode, bdy)
		} else {
			return fmt.Errorf("response failed with status code: %d and unable parse body due to : %v", r.StatusCode, err)
		}

	}
	return nil
}

func getDnsRecords(httpClient http.Client, apiUrl url.URL, token string, companyId int, domain string, page int, perPage int) (*GetDataResponse, error) {
	u := apiUrl
	u.Path += "/directory/v1/org/" + strconv.Itoa(companyId) + "/domains/" + domain + "/dns"

	q := u.Query()
	q.Add("page", strconv.Itoa(page))
	q.Add("perPage", strconv.Itoa(perPage))

	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)

	req.Header.Set("Authorization", "OAuth "+token)

	r, err := httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %v", err)
	}

	bdy, err := io.ReadAll(r.Body)

	if r.StatusCode != 200 || err != nil {
		return nil, fmt.Errorf("response failed with status code: %d and body: %s", r.StatusCode, bdy)
	}

	r.Body.Close()

	var rsp GetDataResponse

	err = json.Unmarshal(bdy, &rsp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	fmt.Println(string(bdy), rsp)

	return &rsp, nil
}
