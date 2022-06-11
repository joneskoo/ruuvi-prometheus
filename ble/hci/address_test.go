package hci

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestFromString(t *testing.T) {

	testdata := []struct {
		name  string
		val   string
		valid bool
	}{
		{
			"valid", "00:11:22:33:44:55", true,
		},
		{
			"valid2", "11:aa:bb:dd:22:33", true,
		},
		{
			"short", "00:11", false,
		},
		{
			"empty", "", false,
		},
		{
			"invalid number", "00:11:2:33:44:55", false,
		},
		{
			"invalid char", "00:aa:bb:vv:33:44", false,
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			addr, err := BtAddressFromString(test.val)
			if test.valid {
				if err != nil {
					t.Errorf("Unexpected error while parsing address: %s", err.Error())
				} else if addr.String() != test.val {
					t.Errorf("Parsed address did not match, expected %s, got %s", addr.String(), test.val)
				}
			} else {
				if err == nil {
					t.Errorf("Parsed invalid address")
				}
			}
		})
	}
}

func TestJSONSimple(t *testing.T) {

	baddr, _ := BtAddressFromString("00:11:22:33:44:55")
	baddr.Atype = LeRandomAddress

	encoded, err := json.Marshal(baddr)
	if err != nil {
		t.Errorf("Unable to Marshal Address to JSON: %s", err.Error())
	}

	decoded := BtAddress{}
	err = json.Unmarshal(encoded, &decoded)
	if err != nil {
		t.Errorf("Unable to Unmarshal Address from JSON: %s", err.Error())
	}

	if decoded != baddr {
		t.Errorf("Decoded address (%+v) is not same is encoded address (%+v)", decoded, baddr)
	}
}

func TestRandomAddressTypes(t *testing.T) {

	testdata := []struct {
		address  string
		expected string
	}{
		{"55:D0:F7:48:79:D1", "resolvable"},
		{"4F:74:12:3E:A2:F1", "resolvable"},
		{"69:F5:58:7D:3F:59", "resolvable"},
		{"DC:15:32:FD:71:1F", "static"},
		{"C8:C6:4B:BD:12:10", "static"},
		{"70:7D:0F:37:8C:FA", "resolvable"},
		{"30:7D:0F:37:8C:FA", "non-resolvable"},
		{"54:BD:79:CF:BD:AD", ""},
	}

	for _, test := range testdata {
		t.Run(test.address, func(t *testing.T) {
			addr, _ := BtAddressFromString(test.address)
			if test.expected == "" {
				addr.Atype = LePublicAddress
			} else {
				addr.Atype = LeRandomAddress
			}

			switch test.expected {
			case "resolvable":
				if !addr.IsResolvable() {
					t.Errorf("Address %s not resolvable, should be", test.address)
				}
				if addr.IsStatic() || addr.IsNonResolvable() {
					t.Errorf("also static or non-resolvable")

				}
			case "static":
				if !addr.IsStatic() {
					t.Errorf("Address %s not static, should be", test.address)
				}
				if addr.IsResolvable() || addr.IsNonResolvable() {
					t.Errorf("also resolvable or non-resolvable")

				}
			case "non-resolvable":
				if !addr.IsNonResolvable() {
					t.Errorf("Address %s not non-resolvable, should be", test.address)
				}
				if addr.IsStatic() || addr.IsResolvable() {
					t.Errorf("also static or resolvable")
				}
			case "":
				if addr.IsNonResolvable() || addr.IsResolvable() || addr.IsStatic() {
					t.Errorf("Public address")
				}
			}
		})
	}
}

func TestResolve(t *testing.T) {

	irk, _ := hex.DecodeString("1ABC39E76110FF5EC8715B7907D056AD")
	address, _ := BtAddressFromString("75:d3:32:a3:db:3a")
	address.Atype = LeRandomAddress

	if !address.Resolve(irk) {
		t.Errorf("Failed to resolve with proper IRK")
	}

	address2, _ := BtAddressFromString("75:d3:32:4e:1e:ab")
	address2.Atype = LeRandomAddress

	if address2.Resolve(irk) {
		t.Errorf("Resolved address which should not be resolved")
	}
}

func TestPut(t *testing.T) {

	addr, _ := BtAddressFromString("11:22:33:44:55:66")
	buf := make([]byte, 6)

	addr.Put(buf)
	if !bytes.Equal(buf, []byte{0x66, 0x55, 0x44, 0x33, 0x22, 0x11}) {
		t.Errorf("Invalid data in Put() buffer")
	}
}

func TestPutSmall(t *testing.T) {
	addr, _ := BtAddressFromString("11:22:33:44:55:66")
	buf := make([]byte, 3)

	addr.Put(buf)
	if !bytes.Equal(buf, []byte{0x66, 0x55, 0x44}) {
		t.Errorf("Invalid data in Put() buffer")
	}
}

func TestHasPrefix(t *testing.T) {
	addr, _ := BtAddressFromString("aa:bb:cc:dd:ee:ff")
	buf := make([]byte, 2)
	buf[0] = 0xaa
	buf[1] = 0xbb
	if !addr.HasPrefix(buf) {
		t.Errorf("Did not match prefix")
	}
	buf = make([]byte, 6)
	buf[0] = 0xaa
	buf[1] = 0xbb
	buf[2] = 0xcc
	buf[3] = 0xdd
	buf[4] = 0xee
	buf[5] = 0xff
	if !addr.HasPrefix(buf) {
		t.Errorf("Did not match full address as prefix")
	}

}

func TestHasPrefixNot(t *testing.T) {
	addr, _ := BtAddressFromString("aa:bb:cc:dd:ee:ff")
	buf := make([]byte, 2)
	buf[0] = 0xaa
	buf[1] = 0xcc
	if addr.HasPrefix(buf) {
		t.Errorf("Matched invalid prefix")
	}

	buf = make([]byte, 7)
	buf[0] = 0xaa
	buf[1] = 0xbb
	buf[2] = 0xcc
	buf[3] = 0xdd
	buf[4] = 0xee
	buf[5] = 0xff
	buf[6] = 0x00
	if addr.HasPrefix(buf) {
		t.Errorf("Matched against too long prefix")
	}
}
