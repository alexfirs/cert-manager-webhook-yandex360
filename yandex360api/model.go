package yandex360api

import (
	"net/http"
	"net/url"
)

type ApiSettings struct {
	ApiUrl         *url.URL
	Token          string
	OrganizationId int
	Domain         string
	TTL            int
}

type ApiClient struct {
	client *http.Client
}

type ErrorResponse struct {
	Code    int `json:"code"`
	Details []struct {
		Type string `json:"@type"`
	} `json:"details"`
	Message string `json:"message"`
}

type GetDataResponse struct {
	Page    int         `json:"page"`
	Pages   int         `json:"pages"`
	PerPage int         `json:"perPage"`
	Records []DnsRecord `json:"records"`
	Total   int         `json:"total"`
}

type DnsRecord struct {
	Address    string `json:"address,omitempty"`
	Exchange   string `json:"exchange,omitempty"`
	Flag       int    `json:"flag,omitempty"`
	Name       string `json:"name" binding:"required"`
	Port       int    `json:"port,omitempty"`
	Preference int    `json:"preference,omitempty"`
	Priority   int    `json:"priority,omitempty"`
	RecordID   int    `json:"recordId"`
	Tag        string `json:"tag,omitempty"`
	Target     string `json:"target,omitempty"`
	Text       string `json:"text,omitempty"`
	TTL        int    `json:"ttl" binding:"required"`
	Type       string `json:"type" binding:"required"`
	Value      string `json:"value,omitempty"`
	Weight     int    `json:"weight,omitempty"`
}
