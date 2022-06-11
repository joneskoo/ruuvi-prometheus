package filter

import (
	"bytes"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

//AdFilter can be used to filter incoming Advertising Reports
//if Filter() returns true, matching ScanReport is created and
//passed to the channel returned by Host.StartScan()
type AdFilter interface {
	Filter(*hci.AdvertisingReport) bool
}

type addressFilter struct {
	addr hci.BtAddress
}

func (f *addressFilter) Filter(rep *hci.AdvertisingReport) bool {
	return rep.Address == f.addr
}

//ByAddress returns AdFilter which filters Advertising Reports by sender
//address. Filter passes report if it comes from given address
func ByAddress(address hci.BtAddress) AdFilter {
	return &addressFilter{addr: address}
}

type partAddrFilter struct {
	part []byte
}

func (f *partAddrFilter) Filter(rep *hci.AdvertisingReport) bool {
	return rep.Address.HasPrefix(f.part)
}

//ByPartialAddress returns filter which matches a partial address bytes
//against bytes given in buffer.
func ByPartialAddress(buf []byte) AdFilter {
	return &partAddrFilter{part: buf}
}

//ByVendor returns AdFilter which can be used to filter Advertising Reports
//based on the start of its vendor specific Advertising Data. Filter
//passes Advertising Reports which have vendor -specific advertising data and
//the first bytes of the vendor specific data match the given preamble.
func ByVendor(preamble []byte) AdFilter {
	return &adDataFilter{typ: hci.AdManufacturerSpecific, preamble: preamble}
}

type adDataFilter struct {
	typ      hci.AdType
	preamble []byte
}

func (a *adDataFilter) Filter(report *hci.AdvertisingReport) bool {
	ret := false
	for _, data := range report.Data {
		if data.Typ == a.typ {
			if len(data.Data) < len(a.preamble) {
				continue
			}
			if bytes.Equal(data.Data[:len(a.preamble)], a.preamble) {
				ret = true
				break
			}
		}
	}

	return ret
}

// ByAdData return AdFilter which can be used to filter Advertising Reports
// based on Ad Type and start of the advertising data. The filter matches if
// both type and the start of the data match the ones given to the filter
func ByAdData(typ hci.AdType, preamble []byte) AdFilter {
	return &adDataFilter{typ: typ, preamble: preamble}
}

type adTypeFilter struct {
	typ hci.AdType
}

func (f *adTypeFilter) Filter(report *hci.AdvertisingReport) bool {
	ret := false
	for _, data := range report.Data {
		if data.Typ == f.typ {
			ret = true
			break
		}
	}
	return ret
}

//ByAdType returns filter which can be used to filter Advertising Reports
//based on the Type field in the Ad Structures contained on the Advertising Reports
func ByAdType(typ hci.AdType) AdFilter {
	return &adTypeFilter{typ: typ}
}

type irkFilter struct {
	irk []byte
	// if non-nil, latest resolved address, match first
	resolved *hci.BtAddress
}

func (f *irkFilter) Filter(report *hci.AdvertisingReport) bool {
	// Resolve will check if the address is of right type
	addr := report.Address
	ret := false
	if f.resolved != nil && *f.resolved == addr {
		ret = true
	} else {
		if addr.Resolve(f.irk) {
			f.resolved = &addr
			ret = true
		}
	}
	return ret
}

//ByIrk returns filter which returns true if advertiser address is resolvable using given irk
func ByIrk(irk []byte) AdFilter {
	return &irkFilter{irk: irk}
}

type filterCollection struct {
	filters []AdFilter
}

func (fc *filterCollection) iterate(iter func(f AdFilter) bool) {
	for _, f := range fc.filters {
		if !iter(f) {
			break
		}
	}
}

type anyCollection struct {
	filterCollection
}

func (a *anyCollection) Filter(report *hci.AdvertisingReport) bool {

	match := false
	a.iterate(func(f AdFilter) bool {
		if f.Filter(report) {
			// Any: it is enough that one filter match, break on first match
			match = true
			return false
		}
		return true
	})
	return match
}

//Any returns a filter which matches if any of the given filters would match
func Any(filters []AdFilter) AdFilter {
	if len(filters) == 1 {
		return filters[0]
	}
	return &anyCollection{
		filterCollection{filters: filters},
	}
}

type allCollection struct {
	filterCollection
}

func (a *allCollection) Filter(report *hci.AdvertisingReport) bool {
	match := true
	a.iterate(func(f AdFilter) bool {
		if !f.Filter(report) {
			match = false
			return false
		}
		return true
	})
	return match
}

//All returns a filter which matches if all of the given filters would match
func All(filters []AdFilter) AdFilter {
	if len(filters) == 1 {
		return filters[0]
	}
	return &allCollection{
		filterCollection{filters: filters},
	}
}
