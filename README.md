KAD - Keyboard Automated Design
===============================

KAD is the SVG CAD engine which powers the mechanical keyboard CAD generation site [builder.swillkb.com](http://builder.swillkb.com/).  If you are going to use this library, you should probably review the documentation site for the published builder in order to understand what all the different features are.  The documentation site is available and kept up to date at [builder-docs.swillkb.com](http://builder-docs.swillkb.com/).

KAD is a Golang library to aid in the design of mechanical keyboard plates and cases.  The keyboard layout uses the standard format developed by the [www.keyboard-layout-editor.com](http://www.keyboard-layout-editor.com/) project.  KAD is designed to produce SVG files which can be taken to a laser or water cutting fabrication shop to be cut.  KAD supports a huge number of features, including but not limited it; 4 switch types, 4 stabilizer types, 3 case types, rounded corners, padding, mount holes, usb cutouts, etc...


## Get Started

```
$ go get github.com/swill/kad
```


## Example Usage

``` go
package main

import (
	"encoding/json"
	"log"

	"github.com/swill/kad"
)

func main() {
	// you can define settings and the layout in JSON
	json_bytes := []byte(`{
		"switch-type":3,
		"stab-type":1,
		"layout":[
			["Num Lock","/","*","-"],
			[{"f":3},"7\nHome","8\n↑","9\nPgUp",{"h":2}," "],
			["4\n←","5","6\n→"],["1\nEnd","2\n↓","3\nPgDn",{"h":2},"Enter"],
			[{"w":2},"0\nIns",".\nDel"]
		],
		"case": {
			"case-type":"sandwich",
			"mount-holes-num":4,
			"mount-holes-size":3,
			"mount-holes-edge":6
		},
		"top-padding":9,
		"left-padding":9,
		"right-padding":9,
		"bottom-padding":9
	}`)

	// create a new KAD instance
	cad := kad.New()

	// populate the 'cad' instance with the JSON contents
	err := json.Unmarshal(json_bytes, cad)
	if err != nil {
		log.Fatalf("Failed to parse json data into the KAD file\nError: %s", err.Error())
	}

	// And you can define settings via the KAD instance
	cad.Hash = "usage_example"      // the name of the design
	cad.FileStore = kad.STORE_LOCAL // store the files locally
	cad.FileDirectory = "./"        // the path location where the files will be saved
	cad.FileServePath = "/"         // the url path for the 'results' (don't worry about this)

	// Here are some more settings defined for this case
	cad.Case.UsbWidth = 12 // all dimension are in 'mm'
	cad.Fillet = 3         // 3mm radius on the rounded corners of the case

	// lets draw the SVG files now
	err = cad.Draw()
	if err != nil {
		log.Fatal("Failed to Draw the KAD file\nError: %s", err.Error())
	}
}
```


## License

```
KAD generates SVG CAD files based on a keyboard layout described in JSON.

Copyright (C) 2015-2016  Will Stevens (swill)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
```

