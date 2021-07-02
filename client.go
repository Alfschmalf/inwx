package inwx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"time"

	"github.com/libdns/libdns"
)

type record struct {
	ID      int    `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

type loginRequest struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

type nameserverInfoRequest struct {
	Domain string `json:"domain"`
}

type nameserverInfoResponse struct {
	Domain  string   `json:"domain"`
	Records []record `json:"record"`
}

type nameserverCreateRecordRequest struct {
	Domain  string `json:"domain"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

type nameserverCreateRecordResponse struct {
	ID int `json:"id"`
}

type nameserverDeleteRecordRequest struct {
	ID int `json:"id"`
}

type nameserverDeleteRecordResponse struct {
}

type nameserverUpdateRecordRequest struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

type nameserverUpdateRecordResponse struct {
}

type genericRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type genericResponse struct {
	Code       int         `json:"code"`
	Msg        string      `json:"msg"`
	ReasonCode string      `json:"reasonCode"`
	ResData    interface{} `json:"resData"`
}

func Url() string {
	return "https://api.domrobot.com/jsonrpc/"
	// return "https://api.ote.domrobot.com/jsonrpc/"
}

func createGenericRequest(ctx context.Context, method string, params interface{}) (*http.Request, error) {
	data, err := json.Marshal(&genericRequest{Method: method, Params: params})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", Url(), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func login(ctx context.Context, username string, password string) (http.CookieJar, error) {
	req, err := createGenericRequest(ctx, "account.login", &loginRequest{User: username, Pass: password})
	if err != nil {
		return nil, err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	client.Jar = jar

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return jar, nil
}

func doRequest(jar http.CookieJar, req *http.Request, respData interface{}) (*genericResponse, error) {
	client := &http.Client{}
	client.Jar = jar

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s (%d)", http.StatusText(resp.StatusCode), resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var genResp genericResponse
	genResp.ResData = respData
	if err = json.Unmarshal(data, &genResp); err != nil {
		return nil, err
	}
	if genResp.Code != 1000 {
		return nil, fmt.Errorf("%d: %s", genResp.Code, genResp.ReasonCode)
	}

	return &genResp, nil
}

func doGenericRequest(ctx context.Context, jar http.CookieJar, method string, params interface{}, response interface{}) error {
	req, err := createGenericRequest(ctx, method, params)
	if err != nil {
		return err
	}
	_, err = doRequest(jar, req, response)
	return err
}

func getAllRecords(ctx context.Context, jar http.CookieJar, zone string) ([]libdns.Record, error) {
	result := nameserverInfoResponse{}
	if err := doGenericRequest(ctx, jar, "nameserver.info", &nameserverInfoRequest{Domain: zone}, &result); err != nil {
		return nil, err
	}

	records := []libdns.Record{}
	for _, r := range result.Records {
		records = append(records, libdns.Record{
			ID:    strconv.Itoa(r.ID),
			Type:  r.Type,
			Name:  libdns.RelativeName(r.Name, zone),
			Value: r.Content,
			TTL:   time.Duration(r.TTL) * time.Second,
		})
	}

	return records, nil
}

func createRecord(ctx context.Context, jar http.CookieJar, zone string, r libdns.Record) (libdns.Record, error) {
	reqParam := &nameserverCreateRecordRequest{
		Domain:  zone,
		Type:    r.Type,
		Name:    libdns.AbsoluteName(r.Name, zone),
		Content: r.Value,
		TTL:     int(r.TTL.Seconds()),
	}

	result := &nameserverCreateRecordResponse{}
	if err := doGenericRequest(ctx, jar, "nameserver.createRecord", reqParam, result); err != nil {
		return libdns.Record{}, err
	}

	return libdns.Record{
		ID:    strconv.Itoa(result.ID),
		Type:  r.Type,
		Name:  r.Name,
		Value: r.Value,
		TTL:   r.TTL,
	}, nil
}

func deleteRecord(ctx context.Context, jar http.CookieJar, record libdns.Record) error {
	id, err := strconv.Atoi(record.ID)
	if err != nil {
		return err
	}

	reqParam := &nameserverDeleteRecordRequest{
		ID: id,
	}
	result := &nameserverDeleteRecordResponse{}
	if err := doGenericRequest(ctx, jar, "nameserver.deleteRecord", reqParam, result); err != nil {
		return err
	}

	return nil
}

func updateRecord(ctx context.Context, jar http.CookieJar, zone string, r libdns.Record) (libdns.Record, error) {
	id, err := strconv.Atoi(r.ID)
	if err != nil {
		return libdns.Record{}, err
	}

	reqParam := &nameserverUpdateRecordRequest{
		ID:      id,
		Type:    r.Type,
		Name:    libdns.RelativeName(r.Name, zone),
		Content: r.Value,
		TTL:     int(r.TTL.Seconds()),
	}
	result := &nameserverUpdateRecordResponse{}
	if err := doGenericRequest(ctx, jar, "nameserver.updateRecord", reqParam, result); err != nil {
		return libdns.Record{}, err
	}

	return r, nil
}

func createOrUpdateRecord(ctx context.Context, jar http.CookieJar, zone string, r libdns.Record) (libdns.Record, error) {
	if len(r.ID) == 0 {
		return createRecord(ctx, jar, zone, r)
	}

	return updateRecord(ctx, jar, zone, r)
}
