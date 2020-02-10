package kad

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/ncw/swift"
	"github.com/ajstarks/svgo/float"
)

const (
	PRECISION   float64 = 1000
	OVERLAP             = 0.001
	STORE_SWIFT         = "swift"
	STORE_LOCAL         = "local"
)

type KAD struct {
	Hash           string
	UOM            string
	U1             float64 `json:"key-unit"`
	DMZ            float64
	Width          float64
	Height         float64
	LayoutCenter   Point
	CaseCenter     Point
	Fillet         float64 `json:"fillet"`
	Kerf           float64 `json:"kerf"`
	Xoff           float64
	TopPad         float64         `json:"top-padding"`
	LeftPad        float64         `json:"left-padding"`
	RightPad       float64         `json:"right-padding"`
	BottomPad      float64         `json:"bottom-padding"`
	Xgrow          float64         `json:"grow_x"`
	Ygrow          float64         `json:"grow_y"`
	SwitchType     int             `json:"switch-type"`
	StabType       int             `json:"stab-type"`
	Case           Case            `json:"case"`
	CustomPolygons []CustomPolygon `json:"custom"`
	RawLayout      []interface{}   `json:"layout"`
	Layout         [][]Key         `json:"-"` // ignore in 'unmarshal'
	Svgs           map[string]SvgWrapper
	Layers         map[string]*Layer
	SvgStyle       string
	LineColor      string  `json:"line-color"`
	LineWeight     float64 `json:"line-weight"`
	Result         Result
	Bounds         Bounds
	Swift          *swift.Connection
	SwiftBucket    string
	FileStore      string
	FileDirectory  string
	FileServePath  string
}

type Result struct {
	HasLayers bool                      `json:"has_layers"`
	Plates    []string                  `json:"plates"`
	Formats   []string                  `json:"formats"`
	Details   map[string]*ResultDetails `json:"details"`
}

type ResultDetails struct {
	Name    string   `json:"name"`
	Width   float64  `json:"width"`
	Height  float64  `json:"height"`
	Area    float64  `json:"area"`
	Exports []Export `json:"exports"`
}

type Export struct {
	Ext string `json:"ext"`
	Url string `json:"url"`
}

type SvgWrapper struct {
	File *os.File
	Svg  *svg.SVG
}
type Layer struct {
	CutPolys  []Path
	KeepPolys []Path
	Width     float64
	Height    float64
}

type Bounds struct {
	Xmin float64
	Xmax float64
	Ymin float64
	Ymax float64
}

type UploadCtl struct {
	Export    *Export
	FailedExt string
	DelFile   string
	Error     error
	Attempt   int
}

func New() *KAD {
	k := &KAD{
		Hash:         "",
		UOM:          "mm",
		U1:           19.05,
		DMZ:          5,
		Width:        0,
		Height:       0,
		LayoutCenter: Point{},
		CaseCenter:   Point{},
		Fillet:       0,
		Kerf:         0,
		Xoff:         0,
		TopPad:       0,
		LeftPad:      0,
		RightPad:     0,
		BottomPad:    0,
		Xgrow:        0,
		Ygrow:        0,
		SwitchType:   SWITCHMXH,
		StabType:     STABCHERRYCOSTAR,
		Case: Case{
			EdgeWidth:   0,
			LeftWidth:   0,
			RightWidth:  0,
			TopWidth:    0,
			BottomWidth: 0,
			UsbLocation: 0,
			UsbWidth:    10,
		},
		Svgs:       make(map[string]SvgWrapper),
		Layers:     make(map[string]*Layer),
		SvgStyle:   "fill:none",
		LineColor:  "black",
		LineWeight: 0.05,
		Result: Result{
			HasLayers: false,
			Plates:    []string{},
			Formats:   []string{"svg"},
			Details:   make(map[string]*ResultDetails),
		},
	}

	// if linux we can handle the DXF export so add it
	if runtime.GOOS == "linux" {
		k.Result.Formats = append(k.Result.Formats, []string{"dxf", "eps"}...)
	}
	return k
}

// Draw the SVGs needed for this layout.
func (k *KAD) Draw() error {
	k.Kerf = k.Kerf / 2 // set kerf to be half of the real kerf as we are working from the center of the kerf
	k.SvgStyle = fmt.Sprintf("%s;stroke-width:%fmm;stroke:%s", k.SvgStyle, k.LineWeight, k.LineColor)

	k.InitCaseLayers()
	k.InitCaseEdges()

	if err := k.ParseLayout(); err != nil { // populates k.Layout with Keys
		log.Printf("ERROR in ParseLayout, exiting early...")
		return err
	}
	k.DrawLayout()
	k.UpdateLayerDimensions()
	k.DrawHoles()
	k.FinalizePolygons()
	k.FinalizeLayerDimensions()
	if err := k.DrawOutputFiles(); err != nil {
		log.Printf("ERROR drawing SVGs, exiting early...\n%s", err.Error())
		return err
	}
	if k.FileStore == STORE_SWIFT {
		k.StoreSwiftFiles()
	}
	if k.FileStore == STORE_LOCAL {
		k.StoreLocalFiles()
	}
	return nil
}

// Parse the layout and populate all the important information in the KAD object.
func (k *KAD) ParseLayout() error {
	var err error
	kad_map := false // if there is user data passed to be added to the KAD object
	// see if the first element in the list is a map of user defined settings
	if len(k.RawLayout) > 0 && (reflect.ValueOf(k.RawLayout[0])).Kind() == reflect.Map {
		log.Printf("user settings: %s\n", json_str(k.RawLayout[0]))
		tmp_json, err := json.Marshal(k.RawLayout[0])
		if err != nil {
			log.Printf("ERROR Marshaling user settings\nRawLayout[0]: %s\n%s", json_str(k.RawLayout[0]), err.Error())
			return err
		}
		err = json.Unmarshal(tmp_json, &k) // popluate the KAD with the user specified
		if err != nil {
			log.Printf("ERROR Unmarshaling user settings\nRawLayout[0]: %s\n%s", json_str(k.RawLayout[0]), err.Error())
			return err
		}
		// to simplify things later, we will divide the grow values by two if they are not zero
		k.Xgrow = k.Xgrow / 2
		k.Ygrow = k.Ygrow / 2
		kad_map = true
	}
	// parse the keyboard layout from the slice of slices
	key_map := false // boolean to track if the key has already been handled by a map
	// we now need to make it a [][]interface{} as we now know it has that format (in theory)
	raw_layout := make([][]interface{}, 0)
	var tmp_raw []byte
	if kad_map && len(k.RawLayout) > 1 {
		tmp_raw, err = json.Marshal(k.RawLayout[1:])
		if err != nil {
			log.Printf("ERROR Marshaling layout\nRawLayout[1:]: %s\n%s", json_str(k.RawLayout[1:]), err.Error())
			return err
		}
	} else {
		tmp_raw, err = json.Marshal(k.RawLayout)
		if err != nil {
			log.Printf("ERROR Marshaling layout\nRawLayout: %s\n%s", json_str_ary(k.RawLayout), err.Error())
			return err
		}
	}
	err = json.Unmarshal(tmp_raw, &raw_layout) // use provided description of the key
	if err != nil {
		log.Printf("ERROR Unmarshaling layout\nRawLayout: %s\n%s", json_str_ary(k.RawLayout), err.Error())
		return err
	}
	for row := range raw_layout {
		row_layout := make([]Key, 0)
		for k := range raw_layout[row] {
			key := &Key{}
			key.Stab = -1 // since 0 is a valid entry
			// populate the Key
			if (reflect.ValueOf(raw_layout[row][k])).Kind() == reflect.Map {
				tmp_key, err := json.Marshal(raw_layout[row][k])
				if err != nil {
					log.Printf("ERROR Marshaling key details\nraw_layout[row][k]: %s\n%s", json_str(raw_layout[row][k]), err.Error())
					return err
				}
				err = json.Unmarshal(tmp_key, &key) // use provided description of the key
				if err != nil {
					log.Printf("ERROR Unmarshaling key details\nraw_layout[row][k]: %s\n%s", json_str(raw_layout[row][k]), err.Error())
					return err
				}
				if key.Width < 1 {
					key.Width = 1
				}
				if key.Height < 1 {
					key.Height = 1
				}
				if key.Xrel < 0 && len(row_layout) > 0 { // set stacked on previous key
					prev_key := row_layout[len(row_layout)-1] // get the prev_key
					prev_key.Stacked = true                   // set stacked for prev_key
					row_layout[len(row_layout)-1] = prev_key  // update the row_layout
				}
				row_layout = append(row_layout, *key)
				key_map = true // this will ignore the next non-map key
			} else {
				if !key_map { // only handle if it was not already handled as a key_map, set to defaults
					key.Width = 1
					key.Height = 1
					row_layout = append(row_layout, *key)
				}
				key_map = false
			}
		}
		// add the row of Keys
		k.Layout = append(k.Layout, row_layout)
	}
	return nil
}

// Draw the switch and stabilizer openings for this KAD layout.
func (k *KAD) DrawLayout() {
	prev_width := 0.0
	prev_y_off := 0.0
	c := &Key{}
	p := &Point{k.DMZ + k.Kerf + k.LeftPad, k.DMZ + k.Kerf + k.TopPad}
	for ri, row := range k.Layout {
		for ki, key := range row {
			// handle absolute positioned keys and rotated clusters
			if key.RotateCluster != 0 || key.Xabs != 0 || key.Yabs != 0 {
				if key.RotateCluster != 0 {
					c.RotateCluster = key.RotateCluster
				}
				if key.Xabs != 0 {
					c.Xabs = key.Xabs
				}
				if key.Yabs != 0 {
					c.Yabs = key.Yabs
				}
			}
			switch {
			case ri == 0 && ki == 0: // first key
				p.X += key.Xrel*k.U1 + key.Width*k.U1/2
				p.Y += key.Yrel*k.U1 + k.U1/2
				if c.Xabs != 0 || c.Yabs != 0 { // handle absolute positioned keys
					p.X += c.Xabs * k.U1
					p.Y += c.Yabs * k.U1
				}
			case ki == 0: // change rows
				p.X = k.DMZ + k.LeftPad + k.Kerf + key.Xrel*k.U1 + key.Width*k.U1/2
				switch {
				case key.Xabs != 0 || key.Yabs != 0: // the first row in a cluster
					p.X += c.Xabs * k.U1
					p.Y = k.DMZ + k.TopPad + k.Kerf + c.Yabs*k.U1 + key.Yrel*k.U1 + k.U1/2
				case c.Xabs != 0 || c.Yabs != 0: // a cluster row, but not the first cluster row
					p.X += c.Xabs * k.U1
					p.Y += key.Yrel*k.U1 + k.U1
				default: // all other keys
					p.Y += key.Yrel*k.U1 + k.U1
				}
			default:
				p.X += prev_width*k.U1/2 + key.Xrel*k.U1 + key.Width*k.U1/2
			}
			if prev_y_off != 0 {
				p.Y += -prev_y_off
				prev_y_off = 0.0
			}
			if key.Height > 1 {
				prev_y_off = key.Height*k.U1/2 - k.U1/2
				p.Y += prev_y_off
			}
			var init bool
			if ri == 0 && ki == 0 {
				init = true
			} else {
				init = false
			}
			key.Draw(k, *p, *c, init)
			//k.draw_switch(*p, key, *c, init)
			prev_width = key.Width
		}
	}
}

// update the dimensions of the kad based on what has been added
func (k *KAD) UpdateLayerDimensions() {
	k.Width = k.Bounds.Xmax + k.RightPad + k.Kerf - k.DMZ
	k.Height = k.Bounds.Ymax + k.BottomPad + k.Kerf - k.DMZ
	k.CaseCenter = Point{k.DMZ + (k.Width / 2), k.DMZ + (k.Height / 2)}
	k.LayoutCenter = Point{
		(k.Bounds.Xmax-k.Bounds.Xmin)/2 + k.Bounds.Xmin,
		(k.Bounds.Ymax-k.Bounds.Ymin)/2 + k.Bounds.Ymin}

	// update the result dimensions while we are at it
	for _, layer := range k.Result.Plates {
		switch {
		case (layer == OPENLAYER || layer == CLOSEDLAYER) && k.TopPad < 0 && k.BottomPad < 0:
			if k.Case.EdgeWidth > 0 {
				k.Result.Details[layer].Width = 2*k.Case.EdgeWidth + 4*k.Kerf + 10 // layout the two parts 10mm apart
			} else {
				k.Result.Details[layer].Width = k.LeftPad + k.RightPad + 4*k.Kerf + 10 // layout the two parts 10mm apart
			}
			k.Result.Details[layer].Height = k.Height
		case (layer == OPENLAYER || layer == CLOSEDLAYER) && k.LeftPad < 0 && k.RightPad < 0:
			if k.Case.EdgeWidth > 0 {
				k.Result.Details[layer].Height = 2*k.Case.EdgeWidth + 4*k.Kerf + 10 // layout the two parts 10mm apart
			} else {
				k.Result.Details[layer].Height = k.TopPad + k.BottomPad + 4*k.Kerf + 10 // layout the two parts 10mm apart
			}
			k.Result.Details[layer].Width = k.Width
		default:
			k.Result.Details[layer].Width = k.Width
			k.Result.Details[layer].Height = k.Height
		}
	}
}

// Finalize the polygons before they go for file processing
func (k *KAD) FinalizeLayerDimensions() {
	// update the bounds again
	for _, layer := range k.Result.Plates {
		for _, path := range k.Layers[layer].KeepPolys {
			k.UpdateBounds(path, false)
		}
	}

	// determine the offset from the new bounds
	offset := &Point{0.0, 0.0}
	if k.Bounds.Xmin-k.DMZ < 0 {
		offset.X = -(k.Bounds.Xmin - k.DMZ)
	}
	if k.Bounds.Ymin-k.DMZ < 0 {
		offset.Y = -(k.Bounds.Ymin - k.DMZ)
	}

	// get the new dimensions
	k.Width = k.Bounds.Xmax - k.Bounds.Xmin
	k.Height = k.Bounds.Ymax - k.Bounds.Ymin
	k.CaseCenter.X += offset.X
	k.CaseCenter.Y += offset.Y
	k.LayoutCenter.X += offset.X
	k.LayoutCenter.Y += offset.Y

	// shift the points based on the updated dimensions
	for _, layer := range k.Result.Plates {
		// shift points by offset
		for p, _ := range k.Layers[layer].KeepPolys {
			k.Layers[layer].KeepPolys[p].Rel(*offset)
		}
		// update result sizes
		switch {
		case (layer == OPENLAYER || layer == CLOSEDLAYER) && k.TopPad < 0 && k.BottomPad < 0:
			k.Result.Details[layer].Height = k.Height
		case (layer == OPENLAYER || layer == CLOSEDLAYER) && k.LeftPad < 0 && k.RightPad < 0:
			k.Result.Details[layer].Width = k.Width
		default:
			k.Result.Details[layer].Width = k.Width
			k.Result.Details[layer].Height = k.Height
		}
	}
}

// Initialize the SVG files and their File writers.
func (k *KAD) DrawOutputFiles() error {
	_ = os.Mkdir(k.FileDirectory, 0755)
	for _, layer := range k.Result.Plates {
		abs_svg, err := filepath.Abs(fmt.Sprintf("%s%s_%s.svg", k.FileDirectory, k.Hash, layer))
		file, err := os.Create(abs_svg)
		if err != nil {
			log.Printf("ERROR Creating export file: %s, %s | %s", k.Hash, layer, err.Error())
			return err
		}

		k.Svgs[layer] = SvgWrapper{File: file, Svg: svg.New(file)}
		canvas := k.Svgs[layer].Svg
		canvas.Decimals = 3
		canvas.StartviewUnit(k.Width+2*k.DMZ, k.Height+2*k.DMZ, k.UOM, 0, 0, k.Width+2*k.DMZ, k.Height+2*k.DMZ)

		// draw the elements
		if len(k.Layers[layer].KeepPolys) > 0 { // draw polygons
			xs, ys := make([]float64, 0), make([]float64, 0)
			for _, poly := range k.Layers[layer].KeepPolys {
				if len(poly) > 0 {
					xs, ys = poly.SplitOnAxis()
					canvas.Polygon(xs, ys, k.SvgStyle)
				}
			}
		}

		canvas.End()
		file.Close() // close written svg

		// create other file formats
		if in_strings("dxf", k.Result.Formats) || in_strings("eps", k.Result.Formats) {
			abs_eps := fmt.Sprintf("%s.%s", strings.TrimSuffix(abs_svg, ".svg"), "eps")
			err = exec.Command("inkscape", "-E", abs_eps, abs_svg).Run()
			if err != nil {
				log.Printf("ERROR: could not create EPS file for: %s, %s | %s", k.Hash, layer, err.Error())
				continue // skip dxf because it depends on eps
			}
			log.Println("created eps file")

			if in_strings("dxf", k.Result.Formats) {
				abs_dxf := fmt.Sprintf("%s.%s", strings.TrimSuffix(abs_svg, ".svg"), "dxf")
				cmd := exec.Command("pstoedit", "-dt", "-f", "dxf: -polyaslines -mm", abs_eps, abs_dxf)
				var out bytes.Buffer
				cmd.Stdout = &out
				err = cmd.Run()
				if err != nil {
					log.Printf("ERROR: could not create DXF file for: %s, %s | %s : %s",
						k.Hash, layer, err.Error(), out.String())
				} else {
					log.Println("created dxf file")
				}
			}
		}
	}
	return nil
}

// Store the generated SVG files in an object store.
func (k *KAD) StoreSwiftFiles() {
	log.Printf("started uploading %s\n", k.Hash)
	failed_exts := []string{}
	delete_files := []string{}

	concurrency := 5
	give_up_after := 3

	for _, layer := range k.Result.Plates {
		exports := []Export{}

		sem := make(chan bool, concurrency)
		buffer := make(chan UploadCtl, concurrency)
		process_file := func(ext string, control UploadCtl) {
			// block until we have a semaphore slot
			sem <- true // semaphore lock

			go func(ext string) { // create concurrent goroutine
				defer func() { <-sem }() // semaphore release
				control.Attempt = control.Attempt + 1

				file_path, err := filepath.Abs(fmt.Sprintf("%s%s_%s.%s", k.FileDirectory, k.Hash, layer, ext))
				if err != nil {
					log.Printf("ERROR: Unable to create filepath '%s'\n%s", file_path, err.Error())
					control.Error = err
					control.FailedExt = ext
					buffer <- control
					return
				}

				// check that the file exists
				if _, err := os.Stat(file_path); err == nil {
					// make sure the swift directory is in place
					obj, _, err := k.Swift.Object(k.SwiftBucket, k.Hash)
					if err != nil || obj.ContentType != "application/directory" {
						err = k.Swift.ObjectPutString(k.SwiftBucket, k.Hash, "", "application/directory")
						if err != nil {
							log.Printf("ERROR: Problem creating folder '%s' (not required)\n%s", k.Hash, err.Error())
						}
					}

					// upload the file
					obj_path := fmt.Sprintf("%s/%s_%s.%s", k.Hash, k.Hash, layer, ext)
					f, err := os.Open(file_path)
					if err != nil {
						log.Printf("ERROR: Problem opening file '%s'\n%s", file_path, err.Error())
						control.Error = err
						control.FailedExt = ext
						buffer <- control
						return
					}
					defer f.Close()
					_, err = k.Swift.ObjectPut(k.SwiftBucket, obj_path, f, false, "", "", nil)
					if err != nil {
						log.Printf("ERROR: Problem uploading object '%s'\n%s", obj_path, err.Error())
						control.Error = err
						control.FailedExt = ext
						buffer <- control
						return
					}
					control.Export = &Export{
						Ext: ext,
						Url: fmt.Sprintf(
							"%s%s/%s/%s_%s.%s", k.FileServePath, k.SwiftBucket, k.Hash, k.Hash, layer, ext),
					}
					control.DelFile = file_path
					// send control by default
				} else {
					log.Printf("ERROR: File not found '%s'", file_path)
					control.Error = err
					control.FailedExt = ext
					buffer <- control
					return
				}

				buffer <- control
			}(ext) // call function
		}

		// since the semaphore channel has limited size,
		// this loop must exit in order for the reads to happen
		for _, ext := range k.Result.Formats {
			go process_file(ext, UploadCtl{}) // queue up the files
		}

		// read from the channel of concurrent results
		expect_total := len(k.Result.Formats) // number of tries we know of so far
		for i := 0; i < expect_total; i++ {
			select {
			case result := <-buffer:

				if result.Export != nil {
					exports = append(exports, *result.Export)
					if result.DelFile != "" {
						delete_files = append(delete_files, result.DelFile) // clean up after upload
					}
				} else {
					if result.FailedExt != "" {
						if result.Attempt == give_up_after {
							failed_exts = append(failed_exts, result.FailedExt)
							if result.DelFile != "" {
								delete_files = append(delete_files, result.DelFile) // clean up after upload
							}
						} else {
							expect_total += 1
							go process_file(result.FailedExt, result) // queue up another attempt
						}
					}
				}

			}
		}
		close(buffer)
		close(sem)

		k.Result.Details[layer].Exports = exports

	}
	log.Printf("finished uploading %s\n", k.Hash)
	// remove formats that failed
	if len(failed_exts) > 0 {
		formats := k.Result.Formats
		for i := len(k.Result.Formats) - 1; i >= 0; i-- { // loop backwards so we don't break index on change
			if in_strings(k.Result.Formats[i], failed_exts) {
				formats = append(formats[:i], formats[i+1:]...) // remove index
			}
		}
		k.Result.Formats = formats
	}
	// clean up local files
	for _, f := range delete_files {
		err := os.Remove(f)
		if err != nil {
			log.Printf("ERROR: problem deleting '%s'\n%s", f, err.Error())
		}
	}
}

// Store and serve the generated SVG files locally.
func (k *KAD) StoreLocalFiles() {
	log.Printf("saving locally %s\n", k.Hash)
	failed_exts := []string{}
	_ = os.Mkdir(k.FileDirectory, 0755)
	for _, layer := range k.Result.Plates {
		exports := []Export{}
		failed := false
		for _, ext := range k.Result.Formats {
			file_path, err := filepath.Abs(fmt.Sprintf("%s%s_%s.%s", k.FileDirectory, k.Hash, layer, ext))
			if err != nil {
				log.Printf("ERROR: Unable to create filepath '%s'\n%s", file_path, err.Error())
				failed = true
			}
			if !failed {
				exports = append(exports, Export{
					Ext: ext,
					Url: fmt.Sprintf(
						"%s%s_%s.%s", k.FileServePath, k.Hash, layer, ext),
				})
			}
			if failed {
				// failed extension
				if !in_strings(ext, failed_exts) {
					failed_exts = append(failed_exts, ext)
				}
			}
		}
		k.Result.Details[layer].Exports = exports
	}
	log.Printf("saved locally %s\n", k.Hash)
	// remove formats that failed
	if len(failed_exts) > 0 {
		formats := k.Result.Formats
		for i := len(k.Result.Formats) - 1; i >= 0; i-- { // loop backwards so we don't break index on change
			if in_strings(k.Result.Formats[i], failed_exts) {
				formats = append(formats[:i], formats[i+1:]...) // remove index
			}
		}
		k.Result.Formats = formats
	}
}

// determine the size of the canvas by checking the bounds of the keys.
func (k *KAD) UpdateBounds(path Path, init bool) {
	for _, point := range path {
		if point.X < k.Bounds.Xmin || init {
			k.Bounds.Xmin = point.X
		}
		if point.X > k.Bounds.Xmax || init {
			k.Bounds.Xmax = point.X
		}
		if point.Y < k.Bounds.Ymin || init {
			k.Bounds.Ymin = point.Y
		}
		if point.Y > k.Bounds.Ymax || init {
			k.Bounds.Ymax = point.Y
		}
	}
}

func json_str_ary(input []interface{}) string {
	raw_json, err := json.Marshal(input)
	if err != nil {
		return fmt.Sprintf("%+v", input)
	}
	return string(raw_json)
}

func json_str(input interface{}) string {
	raw_json, err := json.Marshal(input)
	if err != nil {
		return fmt.Sprintf("%+v", input)
	}
	return string(raw_json)
}

func in_strings(query string, strings []string) bool {
	for _, x := range strings {
		if x == query {
			return true
		}
	}
	return false
}

func in_ints(query int, ints []int) bool {
	for _, x := range ints {
		if x == query {
			return true
		}
	}
	return false
}

func in_floats(query float64, floats []float64) bool {
	for _, x := range floats {
		if x == query {
			return true
		}
	}
	return false
}
