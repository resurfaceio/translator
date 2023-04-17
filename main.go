// © 2016-2022 Resurface Labs Inc.

package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	resurfaceio "github.com/resurfaceio/logger-go/v3"
)

type Message struct {
	LogPolicy string   `json:"log_policy"`
	Version   string   `json:"@version"`
	Tags      []string `json:"tags"`
	GatewayIp string   `json:"gateway_ip"`

	UriPath            string              `json:"uri_path"`
	RequestMethod      string              `json:"request_method"`
	RequestProtocol    string              `json:"request_protocol"`
	RequestHttpHeaders []map[string]string `json:"request_http_headers"`
	RequestBody        string              `json:"request_body"`
	Host               string              `json:"host"`
	HttpUserAgent      string              `json:"http_user_agent"`
	QueryString        string              `json:"query_string"`
	ClientIp           string              `json:"client_ip"`
	ImmediateClientIp  string              `json:"immediate_client_ip"`

	StatusCode          string              `json:"status_code"`
	ResponseHttpHeaders []map[string]string `json:"response_http_headers"`
	ResponseBody        string              `json:"response_body"`

	TimeToServeRequest string `json:"time_to_serve_request"`
	Datetime           string `json:"datetime"`
	Timestamp          string `json:"@timestamp"`

	BytesSent           string              `json:"bytes_sent"`
	BytesReceived       string              `json:"bytes_received"`
	TransactionId       string              `json:"transaction_id"`
	GlobalTransactionId string              `json:"global_transaction_id"`
	LatencyInfo         []map[string]string `json:"latency_info"`
	OpentracingInfo     []interface{}       `json:"opentracing_info"`

	Headers     map[string]string `json:"headers"`
	DomainName  string            `json:"domain_name"`
	EndpointUrl string            `json:"endpoint_url"`

	ApiId             string                 `json:"api_id"`
	ApiName           string                 `json:"api_name"`
	ApiVersion        string                 `json:"api_version"`
	OrgId             string                 `json:"org_id"`
	OrgName           string                 `json:"org_name"`
	AppName           string                 `json:"app_name"`
	ProductName       string                 `json:"product_name"`
	DeveloperOrgId    string                 `json:"developer_org_id"`
	DeveloperOrgName  string                 `json:"developer_org_name"`
	DeveloperOrgTitle string                 `json:"developer_org_title"`
	ResourceId        string                 `json:"resource_id"`
	ResourcePath      string                 `json:"resource_path"`
	PlanId            string                 `json:"plan_id"`
	PlanName          string                 `json:"plan_name"`
	CatalogId         string                 `json:"catalog_id"`
	CatalogName       string                 `json:"catalog_name"`
	ClientId          string                 `json:"client_id"`
	Billing           map[string]interface{} `json:"billing"`
}

var l *resurfaceio.HttpLogger

func jsonBytes(req *http.Request) (m Message, err error) {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return
	}
	return
}

func handler(w http.ResponseWriter, r *http.Request) {
	m, err := jsonBytes(r)
	if err != nil {
		w.WriteHeader(500)
	}
	w.WriteHeader(200)
	if m.LogPolicy != "payload" {
		log.Println("API Call not logged. Verify that log policy is set to \"payload\"")
		return
	}

	req := &http.Request{}
	req.Method = m.RequestMethod
	rawURL := m.RequestProtocol + "://" + m.Host + m.UriPath + "?" + m.QueryString
	req.URL, err = url.Parse(rawURL)
	if err != nil {
		return
	}
	req.Host = m.Host
	if m.RequestProtocol == "https" {
		req.TLS = &tls.ConnectionState{}
	}
	req.Header = http.Header{}
	for _, i := range m.RequestHttpHeaders {
		for k, v := range i {
			req.Header.Set(k, v)
		}
	}
	req.Body = io.NopCloser(strings.NewReader(m.RequestBody))

	res := &http.Response{}
	res.StatusCode, err = strconv.Atoi(m.StatusCode[:3])
	if err != nil {
		return
	}
	res.Header = http.Header{}
	for _, i := range m.ResponseHttpHeaders {
		for k, v := range i {
			res.Header.Set(k, v)
		}
	}
	res.Body = io.NopCloser(strings.NewReader(m.ResponseBody))

	nowTime, err := time.Parse(time.RFC3339Nano, m.Datetime)
	if err != nil {
		return
	}

	now := nowTime.UnixNano() / int64(time.Millisecond)

	interval, err := strconv.Atoi(m.TimeToServeRequest)
	if err != nil {
		return
	}

	resurfaceio.SendHttpMessage(l, res, req, now, int64(interval), nil)
}

func main() {
	var err error
	opts := resurfaceio.Options{
		Rules: "include debug",
		Url:   os.Getenv("USAGE_LOGGERS_URL"),
	}

	l, err = resurfaceio.NewHttpLogger(opts)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
