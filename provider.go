package inwx

import (
	"context"
	"net/http"
	"strings"

	"github.com/libdns/libdns"
)

// Provider implements the libdns interfaces for INWX
type Provider struct {
	// AuthUsername is the INWX Username
	AuthUsername string `json:"auth_username"`
	// AuthPassword is the INWX Password
	AuthPassword string `json:"auth_password"`

	cookieJar http.CookieJar
}

func (p *Provider) login(ctx context.Context) error {
	if p.cookieJar != nil {
		return nil
	}
	var err error
	p.cookieJar, err = login(ctx, p.AuthUsername, p.AuthPassword)
	return err
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	if err := p.login(ctx); err != nil {
		return nil, err
	}

	records, err := getAllRecords(ctx, p.cookieJar, unFQDN(zone))
	if err != nil {
		return nil, err
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.login(ctx); err != nil {
		return nil, err
	}

	var appendedRecords []libdns.Record

	for _, record := range records {
		newRecord, err := createRecord(ctx, p.cookieJar, unFQDN(zone), record)
		if err != nil {
			return nil, err
		}
		appendedRecords = append(appendedRecords, newRecord)
	}

	return appendedRecords, nil
}

// DeleteRecords deletes the records from the zone.
func (p *Provider) DeleteRecords(ctx context.Context, _ string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.login(ctx); err != nil {
		return nil, err
	}

	for _, record := range records {
		err := deleteRecord(ctx, p.cookieJar, record)
		if err != nil {
			return nil, err
		}
	}

	return records, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones. It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.login(ctx); err != nil {
		return nil, err
	}

	var setRecords []libdns.Record

	for _, record := range records {
		setRecord, err := createOrUpdateRecord(ctx, p.cookieJar, unFQDN(zone), record)
		if err != nil {
			return setRecords, err
		}
		setRecords = append(setRecords, setRecord)
	}

	return setRecords, nil
}

// unFQDN trims any trailing "." from fqdn. INWX's API does not use FQDNs.
func unFQDN(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
