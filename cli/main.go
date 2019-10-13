package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	".."
)

var (
	outputHash = flag.String("hash", "output", "File prefix for the output")
	outputDir  = flag.String("dir", ".", "Output directory")
	configFile = flag.String("config", "config.json", "JSON Configuration file")
	layoutFile = flag.String("layout", "", "Keyboard layout file. If specified, it will override whatever layout in in the configuration file")
)

func main() {
	flag.Parse()

	json_bytes, errFile := ioutil.ReadFile(*configFile)
	if errFile != nil {
		log.Fatalf("Failed to parse json data into the KAD file\nError: %s", errFile.Error())
	}

	// create a new KAD instance
	cad := kad.New()

	// populate the 'cad' instance with the JSON contents
	err := json.Unmarshal(json_bytes, cad)
	if err != nil {
		log.Fatalf("Failed to parse json data into the KAD file\nError: %s", err.Error())
	}

	if *layoutFile != "" {
		layoutBytes, errLayout := ioutil.ReadFile(*layoutFile)
		if errLayout != nil {
			log.Fatalf("Failed to parse json layout file\nError: %s", errLayout.Error())
		}
		errLayoutJson := json.Unmarshal(layoutBytes, &cad.RawLayout)
		if errLayoutJson != nil {
			log.Fatalf("Failed to parse json data into the KAD file\nError: %s", errLayoutJson.Error())
		}
	}

	cad.FileStore = kad.STORE_LOCAL // store the files locally
	cad.FileServePath = "/"         // the url path for the 'results' (don't worry about this)

	cad.Hash = *outputHash
	cad.FileDirectory = *outputDir + "/"

	// here are some more settings defined for this case
	cad.Case.UsbWidth = 12 // all dimension are in 'mm'
	cad.Fillet = 3         // 3mm radius on the rounded corners of the case

	// lets draw the SVG files now
	err = cad.Draw()
	if err != nil {
		log.Fatal("Failed to Draw the KAD file\nError: %s", err.Error())
	}
}
