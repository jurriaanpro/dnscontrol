package mijnhost

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/StackExchange/dnscontrol/v4/pkg/diff2"
	"github.com/StackExchange/dnscontrol/v4/providers"
)

var features = providers.DocumentationNotes{}

func init() {
	const providerName = "MIJNHOST"
	const providerMaintainer = "@jurriaanpro"
	providers.RegisterDomainServiceProviderType(providerName, providers.DspFuncs{
		Initializer:   NewMijnHost,
		RecordAuditor: AuditRecords,
	}, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

type mijnhostProvider struct {
	client *client
}

func NewMijnHost(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	apiKey := settings["api_key"]
	client := newClient(apiKey)
	return &mijnhostProvider{client: client}, nil
}

func (p *mijnhostProvider) GetZoneRecords(domain string, meta map[string]string) (models.Records, error) {
	records := models.Records{}
	apiRecs, err := p.client.FetchDNS(domain)
	if err != nil {
		return nil, err
	}

	// print records for devbugging
	for _, r := range apiRecs {
		fmt.Printf("Record: Type=%s, Name=%s, Value=%s, TTL=%d\n", r.Type, r.Name, r.Value, r.TTL)
	}

	for _, r := range apiRecs {
		record := &models.RecordConfig{
			Type: r.Type,
			TTL:  uint32(r.TTL),
		}
		record.SetLabelFromFQDN(r.Name, domain)
		if err := record.PopulateFromStringFunc(r.Type, r.Value, domain, nil); err != nil {
			return nil, fmt.Errorf("unparsable record received from mijn.host: %w", err)
		}

		// print record for debugging
		fmt.Printf("Parsed Record: %+v\n", record)
		records = append(records, record)
	}

	return records, nil
}

func (p *mijnhostProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, curRecords models.Records) ([]*models.Correction, int, error) {

	result, err := diff2.ByZone(curRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	if !result.HasChanges {
		return []*models.Correction{}, result.ActualChangeCount, nil
	}

	fmt.Printf("Zone update for %s\n%s", dc.Name, strings.Join(result.Msgs, "\n"))

	// convert result.DesignedPlus to mijn.host native records
	corrections := make([]*models.Correction, 0)

	// Create a correction for changes
	correction := &models.Correction{
		Msg: fmt.Sprintf("Update DNS records for %s", dc.Name),
		F: func() error {
			records := make([]map[string]interface{}, 0)

			for _, r := range result.DesiredPlus {
				record := map[string]interface{}{
					"type":  r.Type,
					"name":  r.GetLabelFQDN() + ".",
					"value": r.GetTargetCombinedFunc(nil),
					"ttl":   r.TTL,
				}
				records = append(records, record)
			}

			return p.client.UpdateDNS(dc.Name, records)
		},
	}

	corrections = append(corrections, correction)

	return corrections, result.ActualChangeCount, nil
}

func (p *mijnhostProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return []*models.Nameserver{}, nil
}
