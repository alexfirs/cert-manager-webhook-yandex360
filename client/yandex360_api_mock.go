package yandex360api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/miekg/dns"
)

const ErrTemplate = `{"code":%d,"message":"%s","details":[{"@type":"type.googleapis.com/google.rpc.RequestInfo","requestId":"00000000-0000-0000-0000-000000000000","servingData":""}]}`

type RequestContextKey string

const (
	OrganizationContextKey  = RequestContextKey("organization")
	DomainEntriesContextKey = RequestContextKey("domainentries")
	DnsEntryContextKey      = RequestContextKey("dnsentry")
)

type Yandex360ApiMockSettings struct {
	authKey                 string
	organizationsAndDomains map[int]Domains
}
type Domains map[string]Records
type Records []DnsRecord

// Simplified API-mock
type Yandex360ApiMock struct {
	server    *http.Server
	dnsServer *dns.Server
	settings  Yandex360ApiMockSettings
	sync.RWMutex
}

func NewYandex360ApiMock(settings Yandex360ApiMockSettings) *Yandex360ApiMock {
	return &Yandex360ApiMock{
		settings: settings,
	}
}

func (y *Yandex360ApiMock) Run(addr string) error {
	if y.server != nil {
		return errors.New("server is running")
	}

	router := mux.NewRouter()

	router.Handle(
		"/directory/v1/org/{organizationId:[0-9]+}/domains/{tlDomain}/dns",
		y.authMiddleware(
			y.organizationMiddleware(
				y.domainMiddleware(
					http.HandlerFunc(y.DnsListHandler),
				),
			),
		),
	).Methods("GET")

	router.Handle(
		"/directory/v1/org/{organizationId:[0-9]+}/domains/{tlDomain}/dns",
		y.authMiddleware(
			y.organizationMiddleware(
				y.domainMiddleware(
					http.HandlerFunc(y.DnsCreateEntryHandler),
				),
			),
		),
	).Methods("POST")

	router.Handle(
		"/directory/v1/org/{organizationId:[0-9]+}/domains/{tlDomain}/dns/{recordId:[0-9]+}",
		y.authMiddleware(
			y.organizationMiddleware(
				y.domainMiddleware(
					http.HandlerFunc(y.DnsDeleteRecordHandler),
				),
			),
		),
	).Methods("DELETE")

	y.server = &http.Server{Addr: addr, Handler: router}

	return y.server.ListenAndServe()
}

func (y *Yandex360ApiMock) RunDns(port string) {

	y.dnsServer = &dns.Server{
		Addr:    ":" + port,
		Net:     "udp",
		Handler: dns.HandlerFunc(y.handleDNSRequest),
	}

	y.dnsServer.ListenAndServe()
}

func (y *Yandex360ApiMock) Stop(ctx context.Context) error {
	return y.server.Shutdown(ctx)
}

func (y *Yandex360ApiMock) StopDns(_ context.Context) error {
	return y.dnsServer.Shutdown()
}

// API handlers

func (y *Yandex360ApiMock) DnsCreateEntryHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("DnsPutEntryHandler")

	orgId, domain := getOrganizatonIdAndDomainFromRequestContext(req)

	bdy, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var newDnsRecord DnsRecord
	err = json.Unmarshal(bdy, &newDnsRecord)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	y.Lock()
	domainEntries := y.settings.organizationsAndDomains[orgId][domain]
	newDnsRecord.RecordID = len(domainEntries) + 1
	y.settings.organizationsAndDomains[orgId][domain] = append(domainEntries, newDnsRecord)
	y.Unlock()

	response, err := json.Marshal(newDnsRecord)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unexpected mock error: unable to marshal"))
	}

	fmt.Print("Added ")
	fmt.Println(string(response))

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (y *Yandex360ApiMock) DnsDeleteRecordHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("DnsDeleteRecordHandler")

	orgId, domain := getOrganizatonIdAndDomainFromRequestContext(req)
	domainEntries, ok := y.settings.organizationsAndDomains[orgId][domain]

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: extract to middleware?
	vars := mux.Vars(req)
	recordIdString, ok := vars["recordId"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	recordId, err := strconv.Atoi(recordIdString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Printf("DnsDeleteRecordHandler: orgId:%d; domain:%s; recId:%d, LOOKUP\n", orgId, domain, recordId)
	index := -1
	for i, val := range domainEntries {
		if val.RecordID == recordId {
			index = i
			break
		}
	}
	if index == -1 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	success := false

	y.Lock()
	// TODO: make number of attempts here if it's a real code but for mock purposes KISS
	if domainEntries[index].RecordID == recordId {
		fmt.Printf("DnsDeleteRecordHandler: orgId:%d; domain:%s; recId:%d, DELETING\n", orgId, domain, recordId)
		// not very effective but not a lot of records typically
		newListOfEntries := append(domainEntries[:index], domainEntries[index+1:]...)
		y.settings.organizationsAndDomains[orgId][domain] = newListOfEntries
		success = true
	}
	y.Unlock()

	fmt.Printf("DnsDeleteRecordHandler: orgId:%d; domain:%s; recId:%d, success:%v\n", orgId, domain, recordId, success)

	if !success {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

func (y *Yandex360ApiMock) DnsListHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("DnsListHandler")

	// defaults
	perPage := 10
	page := 1
	page, perPage = getPagingAttributes(req, 1, 10)

	orgId, domain := getOrganizatonIdAndDomainFromRequestContext(req)
	domainEntries, ok := y.settings.organizationsAndDomains[orgId][domain]

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	total := len(domainEntries)
	records := domainEntries[maxInt(0, (page-1)*perPage):minInt((page)*perPage, total)]

	resp := GetDataResponse{
		Page:    page,
		PerPage: perPage,
		Pages:   int(math.Ceil(float64(total) / float64(perPage))),
		Total:   total,
		Records: records,
	}

	response, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unexpected mock error: unable to marshal"))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (y *Yandex360ApiMock) organizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		orgIdString, ok := vars["organizationId"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		orgId, err := strconv.Atoi(orgIdString)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		domains, ok := y.settings.organizationsAndDomains[orgId]
		if !ok || domains == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(getJsonErrorUnauthorized()))
			return
		}

		ctx := context.WithValue(r.Context(), OrganizationContextKey, orgId)

		fmt.Println("organizationMiddleware fine, orgid=" + strconv.Itoa(orgId))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (y *Yandex360ApiMock) domainMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tlDomain, ok := vars["tlDomain"]
		if !ok || len(tlDomain) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		currentContex := r.Context()
		orgId := currentContex.Value(OrganizationContextKey).(int)
		if orgId < 1 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(getJsonErrorUnauthorized()))
		}

		domains, ok := y.settings.organizationsAndDomains[orgId][tlDomain]
		if !ok || domains == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(getJsonErrorUnauthorized()))
			return
		}

		ctx := context.WithValue(currentContex, DomainEntriesContextKey, tlDomain)

		fmt.Println("domainMiddleware fine,  domain=" + tlDomain)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (y *Yandex360ApiMock) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ("OAuth " + y.settings.authKey) != r.Header.Get("Authorization") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(getJsonErrorUnauthorized()))
			return
		}
		fmt.Println("authMiddleware fine")
		next.ServeHTTP(w, r)
	})
}

// helpers

func getOrganizatonIdAndDomainFromRequestContext(req *http.Request) (int, string) {
	currentContex := req.Context()
	orgId := currentContex.Value(OrganizationContextKey).(int)
	domain := currentContex.Value(DomainEntriesContextKey).(string)
	return orgId, domain
}

func getPagingAttributes(req *http.Request, defaultPage int, defaultPerPage int) (int, int) {
	page := defaultPage
	perPage := defaultPerPage
	querystring := req.URL.Query()
	if tempVal, err := strconv.Atoi(querystring.Get("page")); err == nil {
		page = tempVal
	}
	if tempVal, err := strconv.Atoi(querystring.Get("perPage")); err == nil {
		perPage = tempVal
	}
	return page, perPage
}

// utilities

// get rid of these functions if migrate to go 1.21
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getJsonError(code int, message string) string {
	return fmt.Sprintf(ErrTemplate, code, message)
}

func getJsonErrorUnauthorized() string {
	return getJsonError(16, "Unauthorized")
}
