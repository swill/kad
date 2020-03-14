package kad

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/swill/kad"
)

func TestSwitchTypes(t *testing.T) {
	json_str := `{
		"layout":[
			[{"_t":1},"", {"_t":1,"w":2,"_s":0},"", {"_t":1,"w":2,"_s":1},"", {"_t":1,"w":2,"_s":2},"", {"_t":1,"w":2,"_s":3},"", {"_t":1,"w":2,"_s":4},""],
			[{"_t":2},"", {"_t":2,"w":2,"_s":0},"", {"_t":2,"w":2,"_s":1},"", {"_t":2,"w":2,"_s":2},"", {"_t":2,"w":2,"_s":3},"", {"_t":2,"w":2,"_s":4},""],
			[{"_t":3},"", {"_t":3,"w":2,"_s":0},"", {"_t":3,"w":2,"_s":1},"", {"_t":3,"w":2,"_s":2},"", {"_t":3,"w":2,"_s":3},"", {"_t":3,"w":2,"_s":4},""],
			[{"_t":4},"", {"_t":4,"w":2,"_s":0},"", {"_t":4,"w":2,"_s":1},"", {"_t":4,"w":2,"_s":2},"", {"_t":4,"w":2,"_s":3},"", {"_t":4,"w":2,"_s":4},""]
		]}`

	cad := kad.New()

	// populate the kad object with the request POST json, oh and get the hash...
	decoder := json.NewDecoder(strings.NewReader(json_str))
	err := decoder.Decode(cad)
	if err != nil {
		t.Errorf("TestSwitchTypes: failed to parse json data into KAD file")
		return
	}

	cad.Hash = "switch_types"
	cad.FileStore = kad.STORE_LOCAL
	cad.FileDirectory = "./output/"
	cad.FileServePath = "/test/output/"

	err = cad.Draw()
	if err != nil {
		t.Errorf("TestSwitchTypes: failed to Draw the KAD file")
		return
	}
}

func TestStabCherrySize(t *testing.T) {
	json_str := `{
		"layout":[
			[{"w":2},"", {"w":10},""],
			[{"w":3},"", {"w":9},""],
			[{"w":4},"", {"w":8},""],
			[{"w":2.25},"", {"w":2.75},"", {"w":7},""],
			[{"w":5.5},"", {"w":6.5},""],
			[{"w":1.25},"", {"w":4.5},"", {"w":6.25},""],
			[{"w":6},"", {"w":6},""]
		]}`

	cad := kad.New()

	// populate the kad object with the request POST json, oh and get the hash...
	decoder := json.NewDecoder(strings.NewReader(json_str))
	err := decoder.Decode(cad)
	if err != nil {
		t.Errorf("TestStabCherrySize: failed to parse json data into KAD file")
		return
	}

	cad.Hash = "stab_cherry_size"
	cad.FileStore = kad.STORE_LOCAL
	cad.FileDirectory = "./output/"
	cad.FileServePath = "/test/output/"

	err = cad.Draw()
	if err != nil {
		t.Errorf("TestStabCherrySize: failed to Draw the KAD file")
		return
	}
}

func TestStabAlpsSize(t *testing.T) {
	json_str := `{
		"layout":[
			[{"w":1.5},"", {"w":6.5},""],
			[{"w":1.75},"", {"w":6.25},""],
			[{"w":1},"", {"w":2},"", {"w":2.25},"", {"w":2.75},""]
		]}`

	cad := kad.New()

	// populate the kad object with the request POST json, oh and get the hash...
	decoder := json.NewDecoder(strings.NewReader(json_str))
	err := decoder.Decode(cad)
	if err != nil {
		t.Errorf("TestStabAlpsSize: failed to parse json data into KAD file")
		return
	}

	cad.Hash = "stab_alps_size"
	cad.FileStore = kad.STORE_LOCAL
	cad.FileDirectory = "./output/"
	cad.FileServePath = "/test/output/"

	cad.SwitchType = kad.SWITCHALPS
	cad.StabType = kad.STABALPS

	err = cad.Draw()
	if err != nil {
		t.Errorf("TestStabAlpsSize: failed to Draw the KAD file")
		return
	}
}

func TestStabKailhChocSize(t *testing.T) {
	json_str := `{
		"layout":[
			[{"w":2},"", {"w":6.25},""]
		]}`

	cad := kad.New()

	// populate the kad object with the request POST json, oh and get the hash...
	decoder := json.NewDecoder(strings.NewReader(json_str))
	err := decoder.Decode(cad)
	if err != nil {
		t.Errorf("TestStabKailhChocSize: failed to parse json data into KAD file")
		return
	}

	cad.Hash = "stab_kailh_choc_size"
	cad.FileStore = kad.STORE_LOCAL
	cad.FileDirectory = "./output/"
	cad.FileServePath = "/test/output/"

	cad.SwitchType = kad.SWITCHMX
	cad.StabType = kad.STABKAILHCHOCSOCKETED

	err = cad.Draw()
	if err != nil {
		t.Errorf("TestStabKailhChocSize: failed to Draw the KAD file")
		return
	}
}
