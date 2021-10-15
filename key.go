package kad

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	SWITCHMX         = 1
	SWITCHMXALPS     = 2
	SWITCHMXH        = 3
	SWITCHALPS       = 4
	STABREMOVE       = 0
	STABCHERRYCOSTAR = 1
	STABCHERRY       = 2
	STABCOSTAR       = 3
	STABALPS         = 4
)

type Key struct {
	Width         float64 `json:"w"`  // width in key units
	Height        float64 `json:"h"`  // height in key units
	AltWidth      float64 `json:"w2"` // alternate width in key units for strangely shaped keys
	AltHeight     float64 `json:"h2"` // alternate height in key units for strangely shaped keys
	Xrel          float64 `json:"x"`  // x relative position in key units
	Yrel          float64 `json:"y"`  // y relative position in key units
	Xabs          float64 `json:"rx"` // x absolute position in key units
	Yabs          float64 `json:"ry"` // y absolute position in key units
	Xalt          float64 `json:"x2"` // x relative position in key units for strangely shaped keys
	Yalt          float64 `json:"y2"` // y relative position in key units for strangely shaped keys
	Type          int     `json:"_t"` // switch type as int
	Stab          int     `json:"_s"` // stab type as int
	Kerf          float64 `json:"_k"` // kerf for this key
	Custom        string  `json:"_c"` // center point as custom index
	Stacked       bool
	Bounds        Path
	Rotate        float64 `json:"_r"`  // rotate switch opening in degrees
	RotateStab    float64 `json:"_rs"` // rotate stabilizer opening in degrees
	RotateCluster float64 `json:"r"`   // rotate the following cluster of keys (in degrees)
}

func GetCherryStabOffset(size float64) (float64, error) {
	switch size {
	case 2: // 2u
		return 11.9, nil
	case 2.25: // 2.25u
		return 11.9, nil
	case 2.75: // 2.75u
		return 11.9, nil
	case 3: // 3u
		return 19.05, nil
	case 4: // 4u
		return 28.575, nil
	case 4.5: // 4.5u
		return 34.671, nil
	case 5.5: // 5.5u
		return 42.8625, nil
	case 6: // 6u
		return 47.5, nil
	case 6.25: // 6.25u
		return 50, nil
	case 6.5: // 6.5u
		return 52.38, nil
	case 7: // 7u
		return 57.15, nil
	case 8: // 8u
		return 66.675, nil
	case 9: // 9u
		return 66.675, nil
	case 10: // 10u
		return 66.675, nil
	default:
		return 0, errors.New(fmt.Sprintf("No cherry stabilizer offset defined for a %fu key.", size))
	}
}

func GetAlpsStabOffset(size float64) (float64, error) {
	switch size {
	case 1.75: // 1.75u
		return 11.938, nil
	case 2.0: // 2.0u
		return 14.096, nil
	case 2.25: // 2.25u
		return 14.096, nil
	case 2.75: // 2.75u
		return 14.096, nil
	case 6.25: // 6.25u
		return 41.859, nil
	case 6.5: // 6.5u
		return 45.3, nil
	default:
		return 0, errors.New(fmt.Sprintf("No alps stabilizer offset defined for a %fu key.", size))
	}
}

// Draw an individual switch/stabilizer opening.
func (key *Key) Draw(k *KAD, c Point, ctx Key, init bool) {
	// set the key defaults and update items like kerf to the functional value
	if !in_ints(key.Type, []int{SWITCHMX, SWITCHMXALPS, SWITCHMXH, SWITCHALPS}) {
		key.Type = k.SwitchType
	}
	if !in_ints(key.Stab, []int{STABREMOVE, STABCHERRYCOSTAR, STABCHERRY, STABCOSTAR, STABALPS}) {
		key.Stab = k.StabType
	}
	if key.Kerf != 0 {
		key.Kerf = key.Kerf / 2
	} else {
		key.Kerf = k.Kerf
	}
	// handle custom polygons centered at this key
	if key.Custom != "" {
		index_parts := strings.Split(strings.Replace(key.Custom, " ", "", -1), ",")
		for _, i := range index_parts {
			if index_int, err := strconv.ParseInt(i, 10, 64); err == nil {
				if int64(len(k.CustomPolygons)) > index_int {
					point_ary := strings.Split(strings.Replace(k.CustomPolygons[index_int].RelAbs, " ", "", -1), ";")
					point_ary = append(point_ary, fmt.Sprintf("[%f,%f]", c.X, c.Y))
					k.CustomPolygons[index_int].RelAbs = strings.Join(point_ary, ";")
				}
			}
		}
	}

	vertical := false
	if key.Height > key.Width {
		vertical = true
	}

	// determine the bounds of the keycap
	var bound_path Path
	b := c // bounds center point
	x_point := k.U1 * key.Width / 2
	y_point := k.U1 * key.Height / 2
	if key.AltWidth > key.Width {
		x_point = k.U1 * key.AltWidth / 2
		b = Point{b.X + k.U1*(key.AltWidth-key.Width)/2, b.Y}
	}
	if key.AltHeight > key.Height {
		y_point = k.U1 * key.AltHeight / 2
	}
	bound_path = Path{
		{x_point + OVERLAP, -y_point - OVERLAP},
		{x_point + OVERLAP, y_point + OVERLAP},
		{-x_point - OVERLAP, y_point + OVERLAP},
		{-x_point - OVERLAP, -y_point - OVERLAP},
	}
	// adapt the bounds to account for alternate offsets
	if key.Xalt != 0 {
		b = Point{b.X + k.U1*key.Xalt, b.Y}
	}
	bound_path.Rel(b) // make the path relative to key center
	if key.Rotate != 0 {
		bound_path.RotatePath(key.Rotate, b)
	}
	if ctx.RotateCluster != 0 {
		bound_path.RotatePath(ctx.RotateCluster, Point{ctx.Xabs*k.U1 + k.DMZ + k.LeftPad, ctx.Yabs*k.U1 + k.DMZ + k.TopPad})
	}
	k.UpdateBounds(bound_path, init)

	// add the top layer cutouts for sandwich cases
	if k.Case.Type == CASE_SANDWICH {
		k.Layers[TOPLAYER].CutPolys = append(k.Layers[TOPLAYER].CutPolys, bound_path)
	}

	// draw the switch cutout path
	var switch_path Path
	switch key.Type {
	case SWITCHMX: // standard square mx
		switch_path = Path{
			{7 - key.Kerf + k.Xgrow, -7 + key.Kerf - k.Ygrow}, {7 - key.Kerf + k.Xgrow, 7 - key.Kerf + k.Ygrow},
			{-7 + key.Kerf - k.Xgrow, 7 - key.Kerf + k.Ygrow}, {-7 + key.Kerf - k.Xgrow, -7 + key.Kerf - k.Ygrow},
		}
	case SWITCHMXALPS: // alps + mx compatible
		switch_path = Path{
			{7 - key.Kerf, -7 + key.Kerf}, {7 - key.Kerf, -6.4 + key.Kerf}, {7.8 - key.Kerf, -6.4 + key.Kerf}, {7.8 - key.Kerf, 6.4 - key.Kerf},
			{7 - key.Kerf, 6.4 - key.Kerf}, {7 - key.Kerf, 7 - key.Kerf}, {-7 + key.Kerf, 7 - key.Kerf}, {-7 + key.Kerf, 6.4 - key.Kerf},
			{-7.8 + key.Kerf, 6.4 - key.Kerf}, {-7.8 + key.Kerf, -6.4 + key.Kerf}, {-7 + key.Kerf, -6.4 + key.Kerf}, {-7 + key.Kerf, -7 + key.Kerf},
		}
	case SWITCHMXH: // mx with side wings
		switch_path = Path{
			{7 - key.Kerf, -7 + key.Kerf}, {7 - key.Kerf, -6 + key.Kerf}, {7.8 - key.Kerf, -6 + key.Kerf}, {7.8 - key.Kerf, -2.9 - key.Kerf},
			{7 - key.Kerf, -2.9 - key.Kerf}, {7 - key.Kerf, 2.9 + key.Kerf}, {7.8 - key.Kerf, 2.9 + key.Kerf}, {7.8 - key.Kerf, 6 - key.Kerf},
			{7 - key.Kerf, 6 - key.Kerf}, {7 - key.Kerf, 7 - key.Kerf}, {-7 + key.Kerf, 7 - key.Kerf}, {-7 + key.Kerf, 6 - key.Kerf},
			{-7.8 + key.Kerf, 6 - key.Kerf}, {-7.8 + key.Kerf, 2.9 + key.Kerf}, {-7 + key.Kerf, 2.9 + key.Kerf}, {-7 + key.Kerf, -2.9 - key.Kerf},
			{-7.8 + key.Kerf, -2.9 - key.Kerf}, {-7.8 + key.Kerf, -6 + key.Kerf}, {-7 + key.Kerf, -6 + key.Kerf}, {-7 + key.Kerf, -7 + key.Kerf},
		}
	case SWITCHALPS: // alps cutout
		switch_path = Path{
			{7.8 - key.Kerf, -6.4 + key.Kerf}, {7.8 - key.Kerf, 6.4 - key.Kerf},
			{-7.8 + key.Kerf, 6.4 - key.Kerf}, {-7.8 + key.Kerf, -6.4 + key.Kerf},
		}
	}

	if vertical {
		switch_path.RotatePath(90, Point{0, 0})
	}
	if key.Rotate != 0 {
		switch_path.RotatePath(key.Rotate, Point{0, 0})
	}
	switch_path.Rel(c) // make the path relative to center

	if ctx.RotateCluster != 0 {
		switch_path.RotatePath(ctx.RotateCluster, Point{ctx.Xabs*k.U1 + k.DMZ + k.LeftPad, ctx.Yabs*k.U1 + k.DMZ + k.TopPad})
	}

	// check if the key needs stabilizer cutouts
	flip_stab := false
	if ctx.RotateCluster > 0 && (key.Width >= 2 || (vertical && key.Height >= 2)) {
		flip_stab = true
	}

	key.DrawStabs(k, c, ctx, vertical, flip_stab, key.Stab, k.ConnectedStabs)

	if key.Width == 6 || (vertical && key.Height == 6) { // adjust for offcenter stem switch
		switch_path.Rel(Point{k.U1 / 2, 0}) // off center is 1/2 a switch right
	}

	k.Layers[SWITCHLAYER].CutPolys = append(k.Layers[SWITCHLAYER].CutPolys, switch_path)
}

// path for cherry + costar stabilizer
func (key *Key) DrawStabs(k *KAD, c Point, ctx Key, vertical, flip_stab bool, stabType int, connector bool) {
	var stab_path_l Path
	var stab_path_r Path
	var blanked_area Path
	size := key.Width
	if vertical {
		size = key.Height
	}

	var stabOffset float64
	var err error
	switch stabType {
	case STABCHERRYCOSTAR:
		stabOffset, err = GetCherryStabOffset(size)
	case STABCHERRY:
		stabOffset, err = GetCherryStabOffset(size)
	case STABCOSTAR:
		stabOffset, err = GetCherryStabOffset(size)
	case STABALPS:
		stabOffset, err = GetAlpsStabOffset(size)
	}
	if err != nil {
		return
	}

	l := -stabOffset
	r := stabOffset

	switch stabType {
	case STABCHERRYCOSTAR:
		stab_path_l = Path{
			{l - 3.375 + key.Kerf, -2.3 + key.Kerf}, {l - 3.375 + key.Kerf, -5.53 + key.Kerf}, {l - 1.65 + key.Kerf, -5.53 + key.Kerf},
			{l - 1.65 + key.Kerf, -6.45 + key.Kerf}, {l + 1.65 - key.Kerf, -6.45 + key.Kerf}, {l + 1.65 - key.Kerf, -5.53 + key.Kerf},
			{l + 3.375 - key.Kerf, -5.53 + key.Kerf}, {l + 3.375 - key.Kerf, -2.3 + key.Kerf}, {l + 4.2 - key.Kerf, -2.3 + key.Kerf},
			{l + 4.2 - key.Kerf, 0.5 - key.Kerf}, {l + 3.375 - key.Kerf, 0.5 - key.Kerf}, {l + 3.375 - key.Kerf, 6.77 - key.Kerf},
			{l + 1.65 - key.Kerf, 6.77 - key.Kerf}, {l + 1.65 - key.Kerf, 7.75 - key.Kerf}, {l - 1.65 + key.Kerf, 7.75 - key.Kerf},
			{l - 1.65 + key.Kerf, 6.77 - key.Kerf}, {l - 3.375 + key.Kerf, 6.77 - key.Kerf}, {l - 3.375 + key.Kerf, 2.3 - key.Kerf},
			{l - 3.375 - key.Kerf, 0.5 - key.Kerf}, {l - 4.2 - key.Kerf, 0.5 - key.Kerf}, {l - 4.2 - key.Kerf, -2.3 - key.Kerf},
		}
		stab_path_r = Path{
			{r - 3.375 + key.Kerf, -2.3 + key.Kerf}, {r - 3.375 + key.Kerf, -5.53 + key.Kerf}, {r - 1.65 + key.Kerf, -5.53 + key.Kerf},
			{r - 1.65 + key.Kerf, -6.45 + key.Kerf}, {r + 1.65 - key.Kerf, -6.45 + key.Kerf}, {r + 1.65 - key.Kerf, -5.53 + key.Kerf},
			{r + 3.375 - key.Kerf, -5.53 + key.Kerf}, {r + 3.375 - key.Kerf, -2.3 + key.Kerf}, {r + 4.2 - key.Kerf, -2.3 + key.Kerf},
			{r + 4.2 - key.Kerf, 0.5 - key.Kerf}, {r + 3.375 - key.Kerf, 0.5 - key.Kerf}, {r + 3.375 - key.Kerf, 6.77 - key.Kerf},
			{r + 1.65 - key.Kerf, 6.77 - key.Kerf}, {r + 1.65 - key.Kerf, 7.75 - key.Kerf}, {r - 1.65 + key.Kerf, 7.75 - key.Kerf},
			{r - 1.65 + key.Kerf, 6.77 - key.Kerf}, {r - 3.375 + key.Kerf, 6.77 - key.Kerf}, {r - 3.375 + key.Kerf, 2.3 - key.Kerf},
			{r - 3.375 - key.Kerf, 0.5 - key.Kerf}, {r - 4.2 - key.Kerf, 0.5 - key.Kerf}, {r - 4.2 - key.Kerf, -2.3 - key.Kerf},
		}
	case STABCHERRY:
		stab_path_l = Path{
			{l - 3.375 + key.Kerf, -5.53 + key.Kerf}, {l + 3.375 - key.Kerf, -5.53 + key.Kerf}, {l + 3.375 - key.Kerf, 6.77 - key.Kerf},
			{l - 3.375 + key.Kerf, 6.77 - key.Kerf},
		}
		stab_path_r = Path{
			{r - 3.375 + key.Kerf, -5.53 + key.Kerf}, {r + 3.375 - key.Kerf, -5.53 + key.Kerf}, {r + 3.375 - key.Kerf, 6.77 - key.Kerf},
			{r - 3.375 + key.Kerf, 6.77 - key.Kerf},
		}
	case STABCOSTAR:
		stab_path_l = Path{
			{l + 1.65 - key.Kerf, -6.45 + key.Kerf}, {l - 1.65 + key.Kerf, -6.45 + key.Kerf}, {l - 1.65 + key.Kerf, 7.75 - key.Kerf},
			{l + 1.65 - key.Kerf, 7.75 - key.Kerf},
		}
		stab_path_r = Path{
			{r - 1.65 + key.Kerf, -6.45 + key.Kerf}, {r + 1.65 - key.Kerf, -6.45 + key.Kerf}, {r + 1.65 - key.Kerf, 7.75 - key.Kerf},
			{r - 1.65 + key.Kerf, 7.75 - key.Kerf},
		}
	case STABALPS:
		stab_path_l = Path{
			{l - 1.333 + key.Kerf, 3.873 + key.Kerf}, {l + 1.333 - key.Kerf, 3.873 + key.Kerf},
			{l + 1.333 - key.Kerf, 9.08 - key.Kerf}, {l - 1.333 + key.Kerf, 9.08 - key.Kerf},
		}
		stab_path_r = Path{
			{r - 1.333 + key.Kerf, 3.873 + key.Kerf}, {r + 1.333 - key.Kerf, 3.873 + key.Kerf},
			{r + 1.333 - key.Kerf, 9.08 - key.Kerf}, {r - 1.333 + key.Kerf, 9.08 - key.Kerf},
		}
	}

	blanked_area = Path{
		{r - 3.375 + key.Kerf, 2.3 - key.Kerf}, {l - 3.375 - key.Kerf, 2.3 - key.Kerf}, {l - 3.375 - key.Kerf, -2.3 - key.Kerf},
		{r - 3.375 + key.Kerf, -2.3 - key.Kerf},
	}

	if vertical {
		stab_path_l.RotatePath(90, Point{0, 0})
		stab_path_r.RotatePath(90, Point{0, 0})
		blanked_area.RotatePath(90, Point{0, 0})
	}
	if flip_stab {
		stab_path_l.RotatePath(180, Point{0, 0})
		stab_path_r.RotatePath(180, Point{0, 0})
		blanked_area.RotatePath(180, Point{0, 0})
	}
	if key.RotateStab != 0 {
		stab_path_l.RotatePath(key.RotateStab, Point{0, 0})
		stab_path_r.RotatePath(key.RotateStab, Point{0, 0})
		blanked_area.RotatePath(key.RotateStab, Point{0, 0})
	}

	stab_path_l.Rel(c)
	stab_path_r.Rel(c)
	blanked_area.Rel(c)
	if ctx.RotateCluster != 0 {
		stab_path_l.RotatePath(ctx.RotateCluster, Point{ctx.Xabs*k.U1 + k.DMZ + k.LeftPad, ctx.Yabs*k.U1 + k.DMZ + k.TopPad})
		stab_path_r.RotatePath(ctx.RotateCluster, Point{ctx.Xabs*k.U1 + k.DMZ + k.LeftPad, ctx.Yabs*k.U1 + k.DMZ + k.TopPad})
		blanked_area.RotatePath(ctx.RotateCluster, Point{ctx.Xabs*k.U1 + k.DMZ + k.LeftPad, ctx.Yabs*k.U1 + k.DMZ + k.TopPad})
	}
	k.Layers[SWITCHLAYER].CutPolys = append(k.Layers[SWITCHLAYER].CutPolys, stab_path_l)
	k.Layers[SWITCHLAYER].CutPolys = append(k.Layers[SWITCHLAYER].CutPolys, stab_path_r)
	if connector && !in_ints(stabType, []int{STABCOSTAR, STABALPS}) {
		k.Layers[SWITCHLAYER].CutPolys = append(k.Layers[SWITCHLAYER].CutPolys, blanked_area)
	}
}
