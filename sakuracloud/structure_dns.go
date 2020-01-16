// Copyright 2016-2020 terraform-provider-sakuracloud authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sakuracloud

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/sacloud/libsacloud/v2/sacloud"
	"github.com/sacloud/libsacloud/v2/sacloud/types"
)

func expandDNSCreateRequest(d *schema.ResourceData) *sacloud.DNSCreateRequest {
	return &sacloud.DNSCreateRequest{
		Name:        d.Get("zone").(string),
		Description: d.Get("description").(string),
		Tags:        expandTags(d),
		IconID:      expandSakuraCloudID(d, "icon_id"),
		Records:     expandDNSRecords(d, "record"),
	}
}

func expandDNSUpdateRequest(d *schema.ResourceData, dns *sacloud.DNS) *sacloud.DNSUpdateRequest {
	records := dns.Records
	if d.HasChange("record") {
		records = expandDNSRecords(d, "record")
	}
	return &sacloud.DNSUpdateRequest{
		Description: d.Get("description").(string),
		Tags:        expandTags(d),
		IconID:      expandSakuraCloudID(d, "icon_id"),
		Records:     records,
	}
}

func flattenDNSRecords(dns *sacloud.DNS) []interface{} {
	var records []interface{}
	for _, record := range dns.Records {
		records = append(records, flattenDNSRecord(record))
	}

	return records
}

func flattenDNSRecord(record *sacloud.DNSRecord) map[string]interface{} {
	var r = map[string]interface{}{
		"name":  record.Name,
		"type":  record.Type,
		"value": record.RData,
		"ttl":   record.TTL,
	}

	switch record.Type {
	case "MX":
		// ex. record.RData = "10 example.com."
		values := strings.SplitN(record.RData, " ", 2)
		r["value"] = values[1]
		r["priority"] = forceAtoI(values[0])
	case "SRV":
		values := strings.SplitN(record.RData, " ", 4)
		r["value"] = values[3]
		r["priority"] = forceAtoI(values[0])
		r["weight"] = forceAtoI(values[1])
		r["port"] = forceAtoI(values[2])
	default:
		delete(r, "priority")
		delete(r, "weight")
		delete(r, "port")
	}

	return r
}

func expandDNSRecords(d resourceValueGettable, key string) []*sacloud.DNSRecord {
	var records []*sacloud.DNSRecord
	for _, rawRecord := range d.Get(key).([]interface{}) {
		records = append(records, expandDNSRecord(&resourceMapValue{rawRecord.(map[string]interface{})}))
	}
	return records
}

func expandDNSRecord(d resourceValueGettable) *sacloud.DNSRecord {
	t, _ := d.GetOk("type")
	recordType := t.(string)
	name := d.Get("name")
	value := d.Get("value")
	ttl := d.Get("ttl")

	switch recordType {
	case "MX":
		pr := 10
		if p, ok := d.GetOk("priority"); ok {
			pr = p.(int)
		}
		rdata := value.(string)
		if rdata != "" && !strings.HasSuffix(rdata, ".") {
			rdata = rdata + "."
		}
		return &sacloud.DNSRecord{
			Name:  name.(string),
			Type:  types.EDNSRecordType(recordType),
			RData: fmt.Sprintf("%d %s", pr, rdata),
			TTL:   ttl.(int),
		}
	case "SRV":
		pr := 0
		if p, ok := d.GetOk("priority"); ok {
			pr = p.(int)
		}
		weight := 0
		if w, ok := d.GetOk("weight"); ok {
			weight = w.(int)
		}
		port := 1
		if po, ok := d.GetOk("port"); ok {
			port = po.(int)
		}
		rdata := value.(string)
		if rdata != "" && !strings.HasSuffix(rdata, ".") {
			rdata = rdata + "."
		}
		return &sacloud.DNSRecord{
			Name:  name.(string),
			Type:  types.EDNSRecordType(recordType),
			RData: fmt.Sprintf("%d %d %d %s", pr, weight, port, rdata),
			TTL:   ttl.(int),
		}
	default:
		return &sacloud.DNSRecord{
			Name:  name.(string),
			Type:  types.EDNSRecordType(recordType),
			RData: value.(string),
			TTL:   ttl.(int),
		}
	}
}

func expandDNSRecordCreateRequest(d *schema.ResourceData, dns *sacloud.DNS) (*sacloud.DNSRecord, *sacloud.DNSUpdateSettingsRequest) {
	record := expandDNSRecord(d)
	records := append(dns.Records, record)

	return record, &sacloud.DNSUpdateSettingsRequest{
		Records:      records,
		SettingsHash: dns.SettingsHash,
	}
}

func expandDNSRecordDeleteRequest(d *schema.ResourceData, dns *sacloud.DNS) *sacloud.DNSUpdateSettingsRequest {
	record := expandDNSRecord(d)
	var records []*sacloud.DNSRecord

	for _, r := range dns.Records {
		if !isSameDNSRecord(r, record) {
			records = append(records, r)
		}
	}

	return &sacloud.DNSUpdateSettingsRequest{
		Records:      records,
		SettingsHash: dns.SettingsHash,
	}
}
