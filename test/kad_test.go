package kad

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lotier/kad"
	clipper "github.com/swill/go.clipper"
)

func TestUsageWithJSON(t *testing.T) {
	json_str := `{
		"switch-type":3,
		"stab-type":1,
		"connected-stabs":true,
		"layout":[
			["Num Lock","/","*","-"],
			[{"f":3},"7\nHome","8\n↑","9\nPgUp",{"h":2}," "],
			["4\n←","5","6\n→"],["1\nEnd","2\n↓","3\nPgDn",{"h":2},"Enter"],
			[{"w":2},"0\nIns",".\nDel"]
		],
		"case": {
			"case-type":"sandwich",
			"usb-location":100,
			"usb-width":12,
			"mount-holes-num":4,
			"mount-holes-size":3,
			"mount-holes-edge":6
		},
		"top-padding":9,
		"left-padding":9,
		"right-padding":9,
		"bottom-padding":9,
		"fillet":3
	}`

	cad := kad.New()

	// populate the kad object with the request POST json, oh and get the hash...
	decoder := json.NewDecoder(strings.NewReader(json_str))
	err := decoder.Decode(cad)
	if err != nil {
		t.Errorf("TestUsageWithJSON: failed to parse json data into KAD file")
		return
	}

	cad.Hash = "usage_with_json"
	cad.FileStore = kad.STORE_LOCAL
	cad.FileDirectory = "./output/"
	cad.FileServePath = "/test/output/"

	err = cad.Draw()
	if err != nil {
		t.Errorf("TestUsageWithJSON: failed to Draw the KAD file")
		return
	}
}

func TestUsageWithGo(t *testing.T) {
	json_str := `{
		"layout":[
			["Num Lock","/","*","-"],
			[{"f":3},"7\nHome","8\n↑","9\nPgUp",{"h":2}," "],
			["4\n←","5","6\n→"],["1\nEnd","2\n↓","3\nPgDn",{"h":2},"Enter"],
			[{"w":2},"0\nIns",".\nDel"]
		]}`

	cad := kad.New()

	// populate the kad object with the request POST json, oh and get the hash...
	decoder := json.NewDecoder(strings.NewReader(json_str))
	err := decoder.Decode(cad)
	if err != nil {
		t.Errorf("TestUsageWithGo: failed to parse json data into KAD file")
		return
	}

	cad.Hash = "usage_with_go"
	cad.FileStore = kad.STORE_LOCAL
	cad.FileDirectory = "./output/"
	cad.FileServePath = "/test/output/"

	cad.SwitchType = kad.SWITCHMXH
	cad.StabType = kad.STABCHERRYCOSTAR
	cad.Case.Type = "sandwich"
	cad.Case.Holes = 4
	cad.Case.HoleDiameter = 3
	cad.Case.EdgeWidth = 6
	cad.Case.UsbLocation = 100
	cad.Case.UsbWidth = 12
	cad.TopPad = 9
	cad.LeftPad = 9
	cad.RightPad = 9
	cad.BottomPad = 9
	cad.Fillet = 3

	err = cad.Draw()
	if err != nil {
		t.Errorf("TestUsageWithGo: failed to Draw the KAD file")
		return
	}
}

func TestSurfaceArea(t *testing.T) {
	cases := []struct {
		paths []kad.Path
		area  float64
	}{
		{[]kad.Path{{{2, 2}, {2, 4}, {4, 4}, {4, 2}}}, 4},
		{[]kad.Path{{{2, 2}, {4, 2}, {4, 4}, {2, 4}}}, 4},
	}
	for _, c := range cases {
		sa := kad.SurfaceArea(c.paths)
		if sa != c.area {
			t.Errorf("SurfaceArea([]Path) == %f, expected %f", sa, c.area)
		}
	}
}

func TestClipperCleanPolygon(t *testing.T) {
	cases := []struct {
		input  clipper.Path
		output clipper.Path
	}{
		{
			clipper.Path{{1, 1}, {1, 2}, {1, 2}, {2, 2}, {2, 2}, {2, 1}},
			clipper.Path{{1, 1}, {1, 2}, {2, 2}, {2, 1}},
		},
		{
			clipper.Path{{5, 5}, {5, 10}, {5, 10}, {10, 10}, {10, 10}, {10, 5}},
			clipper.Path{{5, 5}, {5, 10}, {10, 10}, {10, 5}},
		},
	}
	for _, c := range cases {
		p := clipper.NewClipper(clipper.IoNone)
		r := p.CleanPolygon(c.input, .7) // In order to just clean duplicate points: .7*.7 = 0.49 < 0.5
		if len(r) != len(c.output) {
			t.Errorf("CleanPolygon(clipper.Path) == %#v, expected %#v", r, c.output)
		}
	}
}
