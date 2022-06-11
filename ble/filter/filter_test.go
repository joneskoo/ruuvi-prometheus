package filter

import (
	"encoding/hex"
	"testing"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

func buildAdvertisingReport(addrString string, data []*hci.AdStructure) *hci.AdvertisingReport {
	addr, _ := hci.BtAddressFromString(addrString)
	return &hci.AdvertisingReport{
		Address:   addr,
		EventType: hci.ScanRsp,
		Rssi:      -70,
		Data:      data,
	}
}

func TestAddressFilter(t *testing.T) {
	addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
	filt := ByAddress(addr)

	rep := buildAdvertisingReport("11:22:33:44:55:66", nil)
	if !filt.Filter(rep) {
		t.Errorf("Expected filter to match")
	}

	rep = buildAdvertisingReport("00:11:22:33:44:55", nil)
	if filt.Filter(rep) {
		t.Errorf("Filter should not have matched")
	}
}

func TestPartialAddrFilter(t *testing.T) {
	buf := make([]byte, 2)
	buf[0] = 0x11
	buf[1] = 0x22
	filt := ByPartialAddress(buf)
	rep := buildAdvertisingReport("11:22:33:44:55:66", nil)
	if !filt.Filter(rep) {
		t.Errorf(("Expected filter to match"))
	}

	buf[0] = 0x11
	buf[1] = 0xaa
	filt = ByPartialAddress(buf)
	if filt.Filter(rep) {
		t.Errorf(("Did not expect filter to match"))
	}
}

func TestVendorData(t *testing.T) {

	buf, _ := hex.DecodeString("0a0b0c")
	filt := ByVendor(buf)

	vdata, _ := hex.DecodeString("0a0b0c0d0e0f")
	structs := make([]*hci.AdStructure, 1)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdManufacturerSpecific
	structs[0].Data = vdata

	rep := buildAdvertisingReport("11:22:33:44:55:66", structs)

	if !filt.Filter(rep) {
		t.Errorf("Expected vendor specific filter to match")
	}
}
func TestVendorDataShort(t *testing.T) {

	buf, _ := hex.DecodeString("0a0b0c")
	filt := ByVendor(buf)

	vdata, _ := hex.DecodeString("0a0b")
	structs := make([]*hci.AdStructure, 1)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdManufacturerSpecific
	structs[0].Data = vdata

	rep := buildAdvertisingReport("11:22:33:44:55:66", structs)

	if filt.Filter(rep) {
		t.Errorf("Did not expect the filter to match")
	}
}
func TestVendorDataNoVendor(t *testing.T) {

	buf, _ := hex.DecodeString("0a0b0c")
	filt := ByVendor(buf)

	vdata, _ := hex.DecodeString("0a0b0c0d0e0f")
	structs := make([]*hci.AdStructure, 1)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdCompleteLocalName
	structs[0].Data = vdata

	rep := buildAdvertisingReport("11:22:33:44:55:66", structs)

	if filt.Filter(rep) {
		t.Errorf("Expected vendor specific filter to match")
	}
}

func TestAdDataFiltering(t *testing.T) {
	buf, _ := hex.DecodeString("414243")
	filter := ByAdData(hci.AdCompleteLocalName, buf)

	ndata, _ := hex.DecodeString("414243444546")
	structs := make([]*hci.AdStructure, 1)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdCompleteLocalName
	structs[0].Data = ndata

	report := buildAdvertisingReport("11:22:33:44:55:66", structs)
	if !filter.Filter(report) {
		t.Errorf("Expected the filter to match")
	}
}
func TestAdDataFilteringNotMatch(t *testing.T) {
	buf, _ := hex.DecodeString("414243")
	filter := ByAdData(hci.AdCompleteLocalName, buf)

	ndata, _ := hex.DecodeString("414043444546")
	structs := make([]*hci.AdStructure, 1)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdCompleteLocalName
	structs[0].Data = ndata

	report := buildAdvertisingReport("11:22:33:44:55:66", structs)
	if filter.Filter(report) {
		t.Errorf("Expected the filter to not match")
	}
}

func TestAdTypeFiltering(t *testing.T) {

	filt := ByAdType(hci.AdCompleteLocalName)
	structs := make([]*hci.AdStructure, 2)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdManufacturerSpecific
	structs[0].Data = nil
	structs[1] = new(hci.AdStructure)
	structs[1].Typ = hci.AdCompleteLocalName
	structs[1].Data = []byte("local")
	rep := buildAdvertisingReport("11:22:33:44:55:66", structs)

	if !filt.Filter(rep) {
		t.Errorf("Expected Ad Type filter to match")
	}

}

func TestAdTypeFilteringNoMatch(t *testing.T) {

	filt := ByAdType(hci.AdCompleteLocalName)
	structs := make([]*hci.AdStructure, 2)
	structs[0] = new(hci.AdStructure)
	structs[0].Typ = hci.AdManufacturerSpecific
	structs[0].Data = nil
	structs[1] = new(hci.AdStructure)
	structs[1].Typ = hci.AdShortenedLocalName
	structs[1].Data = []byte("local")
	rep := buildAdvertisingReport("11:22:33:44:55:66", structs)

	if filt.Filter(rep) {
		t.Errorf("Did not expect the filter to match")
	}

}

func TestIrkFiltering(t *testing.T) {
	irk, _ := hex.DecodeString("1ABC39E76110FF5EC8715B7907D056AD")
	filt := ByIrk(irk)

	rep := buildAdvertisingReport("75:d3:32:a3:db:3a", nil)
	rep.Address.Atype = hci.LeRandomAddress
	rep2 := buildAdvertisingReport("75:d3:32:a3:db:3b", nil)
	rep2.Address.Atype = hci.LeRandomAddress

	if !filt.Filter(rep) {
		t.Errorf("Expected filter to match")
	}

	if filt.Filter(rep2) {
		t.Errorf("Did not expect the filter to match")
	}

	if !filt.Filter(rep) {
		t.Errorf("Expected to match also 2nd time")
	}

}

func TestAnyFilterMatch(t *testing.T) {

	addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
	afilt := ByAddress(addr)
	tfilt := ByAdType(hci.AdCompleteLocalName)

	filt := Any([]AdFilter{afilt, tfilt})

	namestruct := hci.AdStructure{
		Typ:  hci.AdCompleteLocalName,
		Data: nil,
	}

	snamestruct := hci.AdStructure{
		Typ:  hci.AdShortenedLocalName,
		Data: nil,
	}

	report1 := buildAdvertisingReport("11:22:33:44:55:66", []*hci.AdStructure{&snamestruct})
	if !filt.Filter(report1) {
		t.Errorf("Expected ANY to match")
	}

	report2 := buildAdvertisingReport("11:22:33:44:55:66", []*hci.AdStructure{&namestruct})
	if !filt.Filter(report2) {
		t.Errorf("Expected ANY to match")
	}

	report3 := buildAdvertisingReport("00:22:33:44:55:66", []*hci.AdStructure{&namestruct})
	if !filt.Filter(report3) {
		t.Errorf("Expected ANY to match")
	}

	report4 := buildAdvertisingReport("00:22:33:44:55:66", []*hci.AdStructure{&snamestruct})
	if filt.Filter(report4) {
		t.Errorf("Did not expect ANY to match")
	}

	// Try Any with only single filter
	filt2 := Any([]AdFilter{afilt})
	if !filt2.Filter(report1) {
		t.Errorf("Expected Any w/ single filter to match")
	}

	if filt2.Filter(report3) {
		t.Errorf("Did not expect Any w/ single filter to match")
	}
}

func TestAllFilterMatch(t *testing.T) {

	addr, _ := hci.BtAddressFromString("11:22:33:44:55:66")
	afilt := ByAddress(addr)
	tfilt := ByAdType(hci.AdCompleteLocalName)

	filt := All([]AdFilter{afilt, tfilt})

	namestruct := hci.AdStructure{
		Typ:  hci.AdCompleteLocalName,
		Data: nil,
	}

	snamestruct := hci.AdStructure{
		Typ:  hci.AdShortenedLocalName,
		Data: nil,
	}

	report1 := buildAdvertisingReport("11:22:33:44:55:66", []*hci.AdStructure{&snamestruct})
	if filt.Filter(report1) {
		t.Errorf("Did not expected ALL to match")
	}

	report2 := buildAdvertisingReport("11:22:33:44:55:66", []*hci.AdStructure{&namestruct})
	if !filt.Filter(report2) {
		t.Errorf("Did expect ALL to match")
	}

	report3 := buildAdvertisingReport("00:22:33:44:55:66", []*hci.AdStructure{&namestruct})
	if filt.Filter(report3) {
		t.Errorf("Did not expected ALL to match")
	}

	report4 := buildAdvertisingReport("00:22:33:44:55:66", []*hci.AdStructure{&snamestruct})
	if filt.Filter(report4) {
		t.Errorf("Did not expected ALL to match")
	}

	filt2 := All([]AdFilter{afilt})
	if !filt2.Filter(report1) {
		t.Errorf("Expected to All w/ single filter to match")
	}

	if filt2.Filter(report3) {
		t.Errorf("Did not expect All w/ single filter to match")
	}
}
