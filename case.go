package kad

import "math"

const (
	CASE_NONE        = ""
	CASE_POKER       = "poker"
	CASE_SANDWICH    = "sandwich"
	TOPLAYER         = "top"
	SWITCHLAYER      = "switch"
	BOTTOMLAYER      = "bottom"
	CLOSEDLAYER      = "closed"
	OPENLAYER        = "open"
	TOPLAYER_NAME    = "Top Layer"
	SWITCHLAYER_NAME = "Switch Layer"
	BOTTOMLAYER_NAME = "Bottom Layer"
	CLOSEDLAYER_NAME = "Closed Layer"
	OPENLAYER_NAME   = "Open Layer"
)

type Case struct {
	Type             string  `json:"case-type"`
	HoleDiameter     float64 `json:"mount-holes-size"`
	Holes            int     `json:"mount-holes-num"`
	EdgeWidth        float64 `json:"mount-holes-edge"`
	LeftWidth        float64
	RightWidth       float64
	TopWidth         float64
	BottomWidth      float64
	Xholes           int
	Yholes           int
	RemovePokerSlots bool    `json:"poker-slots-remove"`
	UsbLocation      float64 `json:"usb-location"`
	UsbWidth         float64 `json:"usb-width"`
}

func (k *KAD) InitCaseLayers() {
	// setup drawings details
	switch k.Case.Type {
	case CASE_NONE:
		k.Result.Plates = []string{SWITCHLAYER}
		k.Result.Details[SWITCHLAYER] = &ResultDetails{
			Name: SWITCHLAYER_NAME,
		}
	case CASE_POKER:
		k.Result.Plates = []string{SWITCHLAYER}
		k.Result.Details[SWITCHLAYER] = &ResultDetails{
			Name: SWITCHLAYER_NAME,
		}
	case CASE_SANDWICH:
		k.Result.Plates = []string{SWITCHLAYER, OPENLAYER, CLOSEDLAYER, TOPLAYER, BOTTOMLAYER}
		k.Result.Details[SWITCHLAYER] = &ResultDetails{
			Name: SWITCHLAYER_NAME,
		}
		k.Result.Details[OPENLAYER] = &ResultDetails{
			Name: OPENLAYER_NAME,
		}
		k.Result.Details[CLOSEDLAYER] = &ResultDetails{
			Name: CLOSEDLAYER_NAME,
		}
		k.Result.Details[TOPLAYER] = &ResultDetails{
			Name: TOPLAYER_NAME,
		}
		k.Result.Details[BOTTOMLAYER] = &ResultDetails{
			Name: BOTTOMLAYER_NAME,
		}
	}
	// initialize the layer objects
	for _, layer := range k.Result.Plates {
		k.Layers[layer] = &Layer{}
	}
}

func (k *KAD) InitCaseEdges() {
	// update the case edge width details
	k.Case.LeftWidth, k.Case.RightWidth, k.Case.TopWidth, k.Case.BottomWidth = k.LeftPad, k.RightPad, k.TopPad, k.BottomPad
	if k.LeftPad > k.Case.EdgeWidth && k.Case.EdgeWidth != 0 {
		k.Case.LeftWidth = k.Case.EdgeWidth
	}
	if k.RightPad > k.Case.EdgeWidth && k.Case.EdgeWidth != 0 {
		k.Case.RightWidth = k.Case.EdgeWidth
	}
	if k.TopPad > k.Case.EdgeWidth && k.Case.EdgeWidth != 0 {
		k.Case.TopWidth = k.Case.EdgeWidth
	}
	if k.BottomPad > k.Case.EdgeWidth && k.Case.EdgeWidth != 0 {
		k.Case.BottomWidth = k.Case.EdgeWidth
	}
}

// Draw the holes for the KAD based on the type of case selected.
func (k *KAD) DrawHoles() {
	switch k.Case.Type {
	case CASE_POKER:
		points := k.GetPokerHoles()
		for i := range points {
			// create circle polygons with 5 segments per 1/4 turn
			k.Layers[SWITCHLAYER].CutPolys = append(k.Layers[SWITCHLAYER].CutPolys,
				CirclePolygon(points[i].X, points[i].Y, (k.Case.HoleDiameter/2)-k.Kerf, 5))
		}
		if !k.Case.RemovePokerSlots {
			// calculate polygon slots for poker layer
			depth := 6.0 - k.Kerf                             // total depth of the side slots in mm
			sc := k.Width/2 - (depth-k.Case.HoleDiameter/2)/2 // round rectangle center for slot
			slots := Path{{sc, 9.2}, {-sc, 9.2}}
			slots.Rel(k.CaseCenter) // make relative to the actual cad coords
			for _, center := range slots {
				slot := RoundRectanglePolygon(center.X, center.Y,
					depth+k.Case.HoleDiameter/2, k.Case.HoleDiameter-k.Kerf*2,
					k.Case.HoleDiameter/2-k.Kerf-.001, 5)
				k.Layers[SWITCHLAYER].CutPolys = append(k.Layers[SWITCHLAYER].CutPolys, slot)
			}
		}
	case CASE_SANDWICH:
		points := k.GetSandwichHoles()
		for _, layer := range k.Result.Plates {
			for i := range points {
				// create circle polygons with 5 segments per 1/4 turn
				k.Layers[layer].CutPolys = append(k.Layers[layer].CutPolys,
					CirclePolygon(points[i].X, points[i].Y, (k.Case.HoleDiameter/2)-k.Kerf, 5))
			}
		}
	}
}

// Get the Path for a Poker case hole placement.
func (k *KAD) GetPokerHoles() Path {
	// the slots at {139, 9.2}, {-139, 9.2} are handled by the 'DrawHoles' function
	points := Path{{-117.3, -19.4}, {-14.3, 0}, {48, 37.9}, {117.55, -19.4}} // relative to center
	points.Rel(k.CaseCenter)
	return points
}

// Get the Path for a Sandwich case hole placement.
func (k *KAD) GetSandwichHoles() Path {
	points := make(Path, 0)
	if k.Case.Holes >= 4 && math.Mod(float64(k.Case.Holes), 2) == 0 {
		switch {
		case k.Case.TopWidth == k.Case.EdgeWidth && k.Case.BottomWidth == k.Case.EdgeWidth &&
			k.Case.LeftWidth == k.Case.EdgeWidth && k.Case.RightWidth == k.Case.EdgeWidth:

			var x_len, y_len float64
			x_len = k.Width - k.Case.EdgeWidth - k.Kerf  // x length to split
			y_len = k.Height - k.Case.EdgeWidth - k.Kerf // y length to split
			x_num := 0.0
			y_num := 0.0
			for i := 0.0; i < (float64(k.Case.Holes-4) / 2); i++ {
				switch {
				case x_len/(x_num+1) == y_len/(y_num+1):
					if x_len >= y_len { // if equal, add the hole to the longer side
						x_num += 1
					} else {
						y_num += 1
					}
				case x_len/(x_num+1) > y_len/(y_num+1):
					x_num += 1
				case x_len/(x_num+1) < y_len/(y_num+1):
					y_num += 1
				}
			}
			// the hole layout has been determined
			x_gap := (x_len - k.Kerf) / (x_num + 1)
			y_gap := (y_len - k.Kerf) / (y_num + 1)
			// start layout out the points
			p := &Point{X: k.DMZ + k.Case.EdgeWidth/2 + k.Kerf, Y: k.DMZ + k.Case.EdgeWidth/2 + k.Kerf} // start at top left  // LeftPad, TopPad
			for i := 0.0; i < x_num+1; i++ {
				p.X += x_gap
				points = append(points, Point{p.X, p.Y})
			}
			for i := 0.0; i < y_num+1; i++ {
				p.Y += y_gap
				points = append(points, Point{p.X, p.Y})
			}
			for i := 0.0; i < x_num+1; i++ {
				p.X -= x_gap
				points = append(points, Point{p.X, p.Y})
			}
			for i := 0.0; i < y_num+1; i++ {
				p.Y -= y_gap
				points = append(points, Point{p.X, p.Y})
			}
		case k.Case.TopWidth == k.Case.EdgeWidth && k.Case.BottomWidth == k.Case.EdgeWidth:
			var x_len, x_num float64
			x_len = k.Width - k.Case.EdgeWidth - k.Kerf // x length to split
			x_num = float64(k.Case.Holes-4) / 2

			// the hole layout has been determined
			x_gap := (x_len - k.Kerf) / (x_num + 1)
			// start layout out the points
			p := &Point{X: k.DMZ + k.Case.EdgeWidth/2 + k.Kerf, Y: k.DMZ + k.Case.EdgeWidth/2 + k.Kerf} // start at top left
			points = append(points, Point{p.X, p.Y})
			for i := 0.0; i < x_num+1; i++ {
				p.X += x_gap
				points = append(points, Point{p.X, p.Y})
			}
			p = &Point{X: k.DMZ + k.Case.EdgeWidth/2 + k.Kerf, Y: k.DMZ + k.Height - k.Case.EdgeWidth/2 - k.Kerf} // start at bottom left
			points = append(points, Point{p.X, p.Y})
			for i := 0.0; i < x_num+1; i++ {
				p.X += x_gap
				points = append(points, Point{p.X, p.Y})
			}
		case k.Case.LeftWidth == k.Case.EdgeWidth && k.Case.RightWidth == k.Case.EdgeWidth:
			var y_len, y_num float64
			y_len = k.Height - k.Case.EdgeWidth - k.Kerf // y length to split
			y_num = float64(k.Case.Holes-4) / 2

			// the hole layout has been determined
			y_gap := (y_len - k.Kerf) / (y_num + 1)
			// start layout out the points
			p := &Point{X: k.DMZ + k.Case.EdgeWidth/2 + k.Kerf, Y: k.DMZ + k.Case.EdgeWidth/2 + k.Kerf} // start at top left
			points = append(points, Point{p.X, p.Y})
			for i := 0.0; i < y_num+2; i++ {
				p.Y += y_gap
				points = append(points, Point{p.X, p.Y})
			}
			p = &Point{X: k.DMZ + k.Width - k.Case.EdgeWidth/2 - k.Kerf, Y: k.DMZ + k.Case.EdgeWidth/2 + k.Kerf} // start at top right
			points = append(points, Point{p.X, p.Y})
			for i := 0.0; i < y_num+2; i++ {
				p.Y += y_gap
				points = append(points, Point{p.X, p.Y})
			}
		}
	}
	return points
}
