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
	Prio    int    `json:"prio"`
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
	Prio    int    `json:"prio"`
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

const Url = "https://api.domrobot.com/jsonrpc/"

// const Url = "https://api.ote.domrobot.com/jsonrpc/"

func login(ctx context.Context, username string, password string) (http.CookieJar, error) {
	data, err := json.Marshal(&genericRequest{Method: "account.login", Params: &loginRequest{User: username, Pass: password}})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", Url, bytes.NewBuffer(data))
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

func getAllRecords(ctx context.Context, jar http.CookieJar, zone string) ([]libdns.Record, error) {
	reqData, err := json.Marshal(&genericRequest{Method: "nameserver.info", Params: &nameserverInfoRequest{Domain: zone}})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", Url, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, err
	}
	result := nameserverInfoResponse{}
	_, err = doRequest(jar, req, &result)
	if err != nil {
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

	reqData, err := json.Marshal(&genericRequest{Method: "nameserver.createRecord", Params: reqParam})
	if err != nil {
		return libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", Url, bytes.NewBuffer(reqData))
	if err != nil {
		return libdns.Record{}, err
	}
	result := nameserverCreateRecordResponse{}
	_, err = doRequest(jar, req, &result)
	if err != nil {
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

	reqData, err := json.Marshal(&genericRequest{Method: "nameserver.deleteRecord", Params: reqParam})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", Url, bytes.NewBuffer(reqData))
	if err != nil {
		return err
	}
	result := nameserverDeleteRecordResponse{}
	_, err = doRequest(jar, req, &result)
	if err != nil {
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

	reqData, err := json.Marshal(&genericRequest{Method: "nameserver.updateRecord", Params: reqParam})
	if err != nil {
		return libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", Url, bytes.NewBuffer(reqData))
	if err != nil {
		return libdns.Record{}, err
	}
	result := nameserverUpdateRecordResponse{}
	_, err = doRequest(jar, req, &result)
	if err != nil {
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
