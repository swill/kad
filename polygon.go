package kad

import (
	"log"
	"math"
	"strings"

	"github.com/Knetic/govaluate"
	clipper "github.com/swill/go.clipper"
)

const (
	CLIP_DIST = 0.7 // In order to just clean duplicate points: .7*.7 = 0.49 < 0.5
)

type Point struct { // point, saving space as it is used a LOT
	X, Y float64
}

type Path []Point

type CustomPolygon struct {
	Diameter float64  `json:"diameter"`
	Height   float64  `json:"height"`
	Layers   []string `json:"layers"`
	Op       string   `json:"op"`
	Points   string   `json:"points"`
	Polygon  string   `json:"polygon"`
	Radius   float64  `json:"radius"`
	RelTo    string   `json:"rel_to"`
	RelAbs   string   `json:"-"`
	Width    float64  `json:"width"`
}

// Sets the path to the absolute positioning of the Path relative to an origin Point 'r'.
func (ps Path) Rel(r Point) {
	for i := range ps {
		ps[i].X += r.X
		ps[i].Y += r.Y
	}
}

// Copies the Path
func (ps Path) Copy() Path {
	dup := Path{}
	for i := range ps {
		dup = append(dup, Point{ps[i].X, ps[i].Y})
	}
	return dup
}

// Rotates each point in a set and rotates them 'r' degrees around 'a'.
func (ps Path) RotatePath(r float64, a Point) {
	for i := range ps {
		px := ps[i].X
		py := ps[i].Y
		ps[i].X = math.Cos(radians(r))*(px-a.X) - math.Sin(radians(r))*(py-a.Y) + a.X
		ps[i].Y = math.Sin(radians(r))*(px-a.X) + math.Cos(radians(r))*(py-a.Y) + a.Y
	}
}

// SplitOnAxis path to be drawn by SVGo
func (ps Path) SplitOnAxis() ([]float64, []float64) {
	xs := make([]float64, 0)
	ys := make([]float64, 0)
	for i := range ps {
		xs = append(xs, ps[i].X)
		ys = append(ys, ps[i].Y)
	}
	return xs, ys
}

func (ps Path) ToClipperPath() clipper.Path {
	p := make(clipper.Path, 0)
	for i := range ps {
		p = append(p, &clipper.IntPoint{clipper.CInt(ps[i].X * PRECISION), clipper.CInt(ps[i].Y * PRECISION)})
	}
	c := clipper.NewClipper(clipper.IoNone)
	p = c.CleanPolygon(p, CLIP_DIST) // clean duplicates
	return p
}

func FromClipperPath(cp clipper.Path) Path {
	c := clipper.NewClipper(clipper.IoNone)
	cp = c.CleanPolygon(cp, CLIP_DIST) // clean duplicates
	p := make(Path, 0)
	for i := range cp {
		p = append(p, Point{float64(cp[i].X) / PRECISION, float64(cp[i].Y) / PRECISION})
	}
	return p
}

// Finalize the polygons before they go for file processing
func (k *KAD) FinalizePolygons() {
	has_err := false
	corner_segments := 20
	if k.Fillet == 0 {
		corner_segments = 0 // square corner
	}
	for _, layer := range k.Result.Plates {
		// handle layer specific details
		switch {
		case layer == OPENLAYER:
			usb_shift := k.Case.UsbLocation
			if usb_shift < -(k.Width/2 - k.Case.EdgeWidth - k.Case.UsbWidth/2) {
				usb_shift = -(k.Width/2 - k.Case.EdgeWidth - k.Case.UsbWidth/2)
			}
			if usb_shift > (k.Width/2 - k.Case.EdgeWidth - k.Case.UsbWidth/2) {
				usb_shift = k.Width/2 - k.Case.EdgeWidth - k.Case.UsbWidth/2
			}
			usb_width := k.Case.UsbWidth
			if usb_width > (k.Width - 2*k.Case.EdgeWidth - 2*k.Kerf) {
				usb_width = k.Width - 2*k.Case.EdgeWidth - 2*k.Kerf
				usb_shift = 0
			}
			c := Point{k.LayoutCenter.X + usb_shift, k.DMZ + k.TopPad/2 + k.Kerf}
			usb_pts := Path{
				{-usb_width/2 + k.Kerf, -k.TopPad/2 - k.Kerf}, {usb_width/2 - k.Kerf, -k.TopPad/2 - k.Kerf},
				{usb_width/2 - k.Kerf, k.TopPad/2 + k.Kerf}, {-usb_width/2 + k.Kerf, k.TopPad/2 + k.Kerf}}
			usb_pts.Rel(c)
			// overlap top side so it is completely removed...
			usb_pts[0].Y -= 1
			usb_pts[1].Y -= 1
			k.Layers[layer].CutPolys = append(k.Layers[layer].CutPolys, usb_pts)

			mid_pts := Path{
				{-k.Width/2 + 2*k.Kerf + k.Case.LeftWidth, -k.Height/2 + 2*k.Kerf + k.Case.TopWidth},
				{k.Width/2 - 2*k.Kerf - k.Case.RightWidth, -k.Height/2 + 2*k.Kerf + k.Case.TopWidth},
				{k.Width/2 - 2*k.Kerf - k.Case.RightWidth, k.Height/2 - 2*k.Kerf - k.Case.BottomWidth},
				{-k.Width/2 + 2*k.Kerf + k.Case.LeftWidth, k.Height/2 - 2*k.Kerf - k.Case.BottomWidth}}
			mid_pts.Rel(k.CaseCenter)
			k.Layers[layer].CutPolys = append(k.Layers[layer].CutPolys, mid_pts)

		case layer == CLOSEDLAYER:
			mid_pts := Path{
				{-k.Width/2 + 2*k.Kerf + k.Case.LeftWidth, -k.Height/2 + 2*k.Kerf + k.Case.TopWidth},
				{k.Width/2 - 2*k.Kerf - k.Case.RightWidth, -k.Height/2 + 2*k.Kerf + k.Case.TopWidth},
				{k.Width/2 - 2*k.Kerf - k.Case.RightWidth, k.Height/2 - 2*k.Kerf - k.Case.BottomWidth},
				{-k.Width/2 + 2*k.Kerf + k.Case.LeftWidth, k.Height/2 - 2*k.Kerf - k.Case.BottomWidth}}
			mid_pts.Rel(k.CaseCenter)
			k.Layers[layer].CutPolys = append(k.Layers[layer].CutPolys, mid_pts)
		}

		// starting point for the 'keep' polygon
		keep_poly := RoundRectanglePolygon(k.DMZ+(k.Width/2), k.DMZ+(k.Height/2),
			k.Width, k.Height, k.Fillet, corner_segments)
		k.Layers[layer].KeepPolys = []Path{keep_poly}

		// union all of the cut polygons to make sure we don't have any crossing cut paths
		if len(k.Layers[layer].CutPolys) > 0 { // union all inside
			c := clipper.NewClipper(clipper.IoNone)
			c.AddPath(Path{}.ToClipperPath(), clipper.PtSubject, true)
			for _, poly := range k.Layers[layer].CutPolys {
				c.AddPath(poly.ToClipperPath(), clipper.PtClip, true)
			}
			solution, ok := c.Execute1(clipper.CtUnion, clipper.PftNonZero, clipper.PftNonZero)
			if !ok {
				log.Printf("ERROR drawing layout: %s, %s", k.Hash, layer)
				log.Printf("ERROR drawing inner union...\nCutPolys: %#v", k.Layers[layer].CutPolys)
				has_err = true
			} else {
				cut_union := make([]Path, 0)
				for _, cpath := range solution {
					cut_union = append(cut_union, FromClipperPath(cpath))
				}
				k.Layers[layer].CutPolys = cut_union
			}
		}

		// union all of the keep polygons to make sure we don't have any crossing keep paths
		if len(k.Layers[layer].KeepPolys) > 0 { // union all inside
			c := clipper.NewClipper(clipper.IoNone)
			c.AddPath(Path{}.ToClipperPath(), clipper.PtSubject, true)
			for _, poly := range k.Layers[layer].KeepPolys {
				c.AddPath(poly.ToClipperPath(), clipper.PtClip, true)
			}
			solution, ok := c.Execute1(clipper.CtUnion, clipper.PftNonZero, clipper.PftNonZero)
			if !ok {
				log.Printf("ERROR drawing layout: %s, %s", k.Hash, layer)
				log.Printf("ERROR drawing inner union...\nKeepPolys: %#v", k.Layers[layer].KeepPolys)
				has_err = true
			} else {
				keep_union := make([]Path, 0)
				for _, cpath := range solution {
					keep_union = append(keep_union, FromClipperPath(cpath))
				}
				k.Layers[layer].KeepPolys = keep_union
			}
		}

		// at this point we have everything we need to evaluate if any cut polygons cross the exterior keep boundary

		// get the surface areas before we 'cut' from the 'keep' paths
		k.Result.Details[layer].Area = SurfaceArea(k.Layers[layer].KeepPolys) - SurfaceArea(k.Layers[layer].CutPolys)

		// get the difference when we do the cut from keep
		if len(k.Layers[layer].CutPolys) > 0 { // difference with cuts
			c := clipper.NewClipper(clipper.IoNone)
			for _, poly := range k.Layers[layer].KeepPolys {
				c.AddPath(poly.ToClipperPath(), clipper.PtSubject, true)
			}
			for _, poly := range k.Layers[layer].CutPolys {
				c.AddPath(poly.ToClipperPath(), clipper.PtClip, true)
			}
			solution, ok := c.Execute1(clipper.CtDifference, clipper.PftNonZero, clipper.PftNonZero)
			if !ok {
				log.Printf("ERROR drawing layout: %s, %s", k.Hash, layer)
				log.Printf("ERROR drawing outer / inner difference...\nKeepPolys: %#v\nCutPolys: %#v",
					k.Layers[layer].KeepPolys, k.Layers[layer].CutPolys)
				has_err = true
			} else {
				keep_polys := make([]Path, 0)
				for _, cpath := range solution {
					keep_polys = append(keep_polys, FromClipperPath(cpath))
				}
				k.Layers[layer].KeepPolys = keep_polys
			}
		}

		// ****************************************************
		// *** handle custom polygons added to this drawing ***
		// ****************************************************
		for _, cp := range k.CustomPolygons {
			if in_strings(layer, cp.Layers) { // apply this custom polygon to this layer
				paths := k.ParsePoints(cp.Points, cp.RelTo, true)
				paths = append(paths, k.ParsePoints(cp.Points, cp.RelAbs, false)...)
				polygons := make([]Path, 0)
				if len(paths) > 0 && len(paths[0]) > 0 {
					switch cp.Polygon {
					case "custom-circle":
						for _, path := range paths {
							for _, pt := range path {
								polygons = append(polygons, CirclePolygon(pt.X, pt.Y, cp.Diameter/2, 20))
							}
						}
						break
					case "custom-superellipse":
						for _, path := range paths {
							for _, pt := range path {
								polygons = append(polygons, SuperellipsePolygon(pt.X, pt.Y, cp.Radius, 20))
							}
						}
						break
					case "custom-rectangle":
						for _, path := range paths {
							for _, pt := range path {
								polygons = append(polygons, RoundRectanglePolygon(pt.X, pt.Y, cp.Width, cp.Height, 0, 0))
							}
						}
						break
					case "custom-rounded-rectangle":
						for _, path := range paths {
							for _, pt := range path {
								polygons = append(polygons, RoundRectanglePolygon(pt.X, pt.Y, cp.Width, cp.Height, cp.Radius, 20))
							}
						}
						break
					case "custom-path":
						for _, path := range paths {
							if len(path) > 2 {
								polygons = append(polygons, path)
							}
						}
						break
					}
				}
				// apply the polygons to the kad
				if len(polygons) > 0 {
					switch cp.Op {
					case "add":
						k.Layers[layer].KeepPolys = append(k.Layers[layer].KeepPolys, polygons...)
						break
					case "cut":
						k.Layers[layer].CutPolys = append(k.Layers[layer].CutPolys, polygons...)
						break
					}
				}
			}
		}

		// union all of the cut polygons to make sure we don't have any crossing cut paths
		if len(k.Layers[layer].CutPolys) > 0 { // union all inside
			c := clipper.NewClipper(clipper.IoNone)
			c.AddPath(Path{}.ToClipperPath(), clipper.PtSubject, true)
			for _, poly := range k.Layers[layer].CutPolys {
				c.AddPath(poly.ToClipperPath(), clipper.PtClip, true)
			}
			solution, ok := c.Execute1(clipper.CtUnion, clipper.PftNonZero, clipper.PftNonZero)
			if !ok {
				log.Printf("ERROR drawing layout: %s, %s", k.Hash, layer)
				log.Printf("ERROR drawing inner union...\nCutPolys: %#v", k.Layers[layer].CutPolys)
				has_err = true
			} else {
				cut_union := make([]Path, 0)
				for _, cpath := range solution {
					cut_union = append(cut_union, FromClipperPath(cpath))
				}
				k.Layers[layer].CutPolys = cut_union
			}
		}

		// union all of the keep polygons to make sure we don't have any crossing keep paths
		if len(k.Layers[layer].KeepPolys) > 0 { // union all inside
			c := clipper.NewClipper(clipper.IoNone)
			c.AddPath(Path{}.ToClipperPath(), clipper.PtSubject, true)
			for _, poly := range k.Layers[layer].KeepPolys {
				c.AddPath(poly.ToClipperPath(), clipper.PtClip, true)
			}
			solution, ok := c.Execute1(clipper.CtUnion, clipper.PftNonZero, clipper.PftNonZero)
			if !ok {
				log.Printf("ERROR drawing layout: %s, %s", k.Hash, layer)
				log.Printf("ERROR drawing inner union...\nKeepPolys: %#v", k.Layers[layer].KeepPolys)
				has_err = true
			} else {
				keep_union := make([]Path, 0)
				for _, cpath := range solution {
					keep_union = append(keep_union, FromClipperPath(cpath))
				}
				k.Layers[layer].KeepPolys = keep_union
			}
		}

		// get the difference when we do the cut from keep
		if len(k.Layers[layer].CutPolys) > 0 { // difference with cuts
			c := clipper.NewClipper(clipper.IoNone)
			for _, poly := range k.Layers[layer].KeepPolys {
				c.AddPath(poly.ToClipperPath(), clipper.PtSubject, true)
			}
			for _, poly := range k.Layers[layer].CutPolys {
				c.AddPath(poly.ToClipperPath(), clipper.PtClip, true)
			}
			solution, ok := c.Execute1(clipper.CtDifference, clipper.PftNonZero, clipper.PftNonZero)
			if !ok {
				log.Printf("ERROR drawing layout: %s, %s", k.Hash, layer)
				log.Printf("ERROR drawing outer / inner difference...\nKeepPolys: %#v\nCutPolys: %#v",
					k.Layers[layer].KeepPolys, k.Layers[layer].CutPolys)
				has_err = true
			} else {
				keep_polys := make([]Path, 0)
				for _, cpath := range solution {
					keep_polys = append(keep_polys, FromClipperPath(cpath))
				}
				k.Layers[layer].KeepPolys = keep_polys
			}
		}
	}

	if has_err {
		log.Printf("ERROR Context (raw layout):\n%#v", k.RawLayout)
	}
}

// Parse the points passed in for custom polygons
func (k *KAD) ParsePoints(points_str, rel_to_str string, rel_center bool) []Path {
	get_points := func(point_str string) Path {
		points := make(Path, 0)
		point_str = strings.ToLower(strings.Replace(point_str, " ", "", -1)) // remove spaces and make lower case
		point_ary := strings.Split(point_str, ";")
		for _, pt := range point_ary {
			pt = strings.Replace(pt, "[", "", -1)
			pt = strings.Replace(pt, "]", "", -1)
			pts := strings.Split(pt, ",")
			point_error := false
			if len(pts) == 2 {
				params := make(map[string]interface{}, 8)
				params["x"] = k.Width / 2
				params["y"] = k.Height / 2

				var err error
				var x_exp, y_exp *govaluate.EvaluableExpression
				var x_val, y_val interface{}
				x_exp, err = govaluate.NewEvaluableExpression(pts[0])
				if err != nil {
					log.Printf("ERROR Govaluating expression: %s", pts[0])
					point_error = true
				}
				if !point_error {
					x_val, err = x_exp.Evaluate(params)
					if err != nil {
						log.Printf("ERROR Govaluating '%s' w/ params: %#v", pts[0], params)
						point_error = true
					}
				}
				if !point_error {
					y_exp, err = govaluate.NewEvaluableExpression(pts[1])
					if err != nil {
						log.Printf("ERROR Govaluating expression: %s", pts[1])
						point_error = true
					}
				}
				if !point_error {
					y_val, err = y_exp.Evaluate(params)
					if err != nil {
						log.Printf("ERROR Govaluating '%s' w/ params: %#v", pts[1], params)
						point_error = true
					}
				}

				if !point_error {
					points = append(points, Point{x_val.(float64), y_val.(float64)})
				}
			}
		}
		return points
	}
	paths := make([]Path, 0)
	points := get_points(points_str)
	rel_to := get_points(rel_to_str)
	if rel_center {
		rel_to.Rel(k.CaseCenter)
	}
	if len(rel_to) > 0 {
		for p := range rel_to {
			rel_points := points.Copy()
			rel_points.Rel(rel_to[p])
			paths = append(paths, rel_points)
		}
	}
	return paths
}

// create a rectangle as a polygon with optional rounded corners.
// set 'r' and 's' to zero to have a non-rounded corner.
func RoundRectanglePolygon(cx, cy, w, h, r float64, s int) Path {
	// make a rounded corner
	corner := func(x, y, r, a float64, s int, ps *Path) {
		n := float64(s)
		p := &Point{x, y}
		*ps = append(*ps, *p)
		la := radians(90 - (90.0 / (2 * n))) // angle to determine the segment length
		for j := 1.0; j < n+1; j++ {
			sa := radians(90 - ((90.0 / (2 * n)) * (2*j - 1)) + a) // angle of the vector of each successive segment
			p.X += 2 * r * math.Cos(la) * math.Sin(sa)
			p.Y += 2 * r * math.Cos(la) * math.Cos(sa)
			*ps = append(*ps, *p)
		}
	}
	// draw the rounded rectangle
	pts := make(Path, 0)
	corner((cx + w/2 - r), (cy - h/2), r, 0, s, &pts)
	corner((cx + w/2), (cy + h/2 - r), r, -90, s, &pts)
	corner((cx - w/2 + r), (cy + h/2), r, 180, s, &pts)
	corner((cx - w/2), (cy - h/2 + r), r, 90, s, &pts)
	return pts
}

// create a circle as a polygon with each quarter made up of 's' segments.
func CirclePolygon(cx, cy, r float64, s int) Path {
	// make a circle
	circle := func(x, y, r float64, s int, ps *Path) {
		n := float64(s)
		p := &Point{x, y}
		*ps = append(*ps, *p)
		la := radians(90 - (90.0 / (2 * n))) // angle to determine the segment length
		for j := 1.0; j < 4*n; j++ {
			sa := radians(90 - ((90.0 / (2 * n)) * (2*j - 1))) // angle of the vector of each successive segment
			p.X += 2 * r * math.Cos(la) * math.Sin(sa)
			p.Y += 2 * r * math.Cos(la) * math.Cos(sa)
			*ps = append(*ps, *p)
		}
	}
	// draw the circle
	pts := make(Path, 0)
	circle(cx, cy-r, r, s, &pts)
	return pts
}

// create a superellipse as a polygon with each quarter made up of 's' segments.
func SuperellipsePolygon(cx, cy, r float64, s int) Path {
	// make a concave quadrant
	quadrant := func(x, y, r, a float64, s int, ps *Path) {
		n := float64(s)
		p := &Point{x, y}
		*ps = append(*ps, *p)
		la := radians(90 + (90.0 / (2 * n))) // angle to determine the segment length
		for j := 1.0; j < n+1; j++ {
			sa := radians(90 + ((90.0 / (2 * n)) * (2*j - 1)) + a) // angle of the vector of each successive segment
			p.X += 2 * r * math.Cos(la) * math.Sin(sa)
			p.Y += 2 * r * math.Cos(la) * math.Cos(sa)
			*ps = append(*ps, *p)
		}
	}
	// draw the superellipse
	pts := make(Path, 0)
	quadrant((cx), (cy - r), r, 90, s, &pts)
	quadrant((cx + r), (cy), r, 0, s, &pts)
	quadrant((cx), (cy + r), r, -90, s, &pts)
	quadrant((cx - r), (cy), r, 180, s, &pts)
	return pts
}

func SurfaceArea(paths []Path) float64 {
	sa := 0.0
	for _, path := range paths {
		area := 0.0
		i := len(path) - 1
		for j := 0; j < len(path); j++ {
			area += (path[i].X + path[j].X) * (path[i].Y - path[j].Y)
			i = j //set previous to current for next pass
		}
		sa += math.Abs(area / 2)
	}
	return sa
}

// convert degrees to radians
func radians(deg float64) float64 {
	return (deg * math.Pi) / 180
}
