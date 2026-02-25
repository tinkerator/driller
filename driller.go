// Program driller converts a combined -PTH.drl file into one-per tool
// drl files and a combined summary SVG file. This has only been
// tested with kicad generated drl files.
//
// Since the program is generating SVGs and KiCad plots layers to SVGs
// with positive Y axis values, the program's default behavior is to
// flip the sign of the output SVG to match KiCad's SVGs. You can
// retain the original Y values by using the --keep-negative command
// line flag.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"zappem.net/pub/graphics/svgof"
	"zappem.net/pub/math/polygon"
)

var (
	src    = flag.String("src", "", "required input *-PTH.drl file")
	dest   = flag.String("dest", "./default", "destination path for output")
	dither = flag.Float64("dither", 0.05, "mm width of rendered lines")
	keep   = flag.Bool("keep-negative", false, "retain negative Y axis values for SVG holes")
)

// Tool defines drill usage details in mm units.
type Tool struct {
	Radius float64
	Holes  []polygon.Point
}

// flatCircle renders a circle polygon as a line.
func flatCircle(s *svgof.SVG, x, y, r float64) {
	pts := 16 * r / *dither
	if pts < 16 {
		pts = 16
	}
	var xs, ys []float64
	seg := 2 * math.Pi / pts
	for i := 0.0; i < pts; i++ {
		a := seg * i
		xs = append(xs, x+r*math.Cos(a))
		ys = append(ys, y+r*math.Sin(a))
	}
	s.Polygon(xs, ys, `fill="none"`, `stroke="black"`, fmt.Sprintf(`stroke-width="%.3f"`, *dither))
}

func main() {
	flag.Parse()

	if *src == "" {
		log.Fatal("the --src=some-PTH.drl argument is required")
	}
	f, err := os.Open(*src)
	if err != nil {
		log.Fatalf("failed to open %q: %v", *src, err)
	}
	defer f.Close()

	var current string
	factor := 0.0
	tools := make(map[string]*Tool)
	var ll, tr polygon.Point
	sc := bufio.NewScanner(f)
	started := false
	count := 0
	for sc.Scan() {
		line := sc.Text()
		count++
		switch line {
		case "M48", "%", "G90", "G05", "M30", "":
			continue
		case "METRIC":
			if factor != 0.0 {
				log.Fatalf("ERROR:%d: redefined length scale %q", count, line)
			}
			factor = 1
			continue
		case "INCH":
			if factor != 0.0 {
				log.Fatalf("ERROR:%d: redefined length scale %q", count, line)
			}
			factor = 25.4
			continue
		}
		switch {
		case line[:1] == "T" && strings.Contains(line, "C"):
			diam := strings.Index(line, "C")
			tool := line[1:diam]
			width, err := strconv.ParseFloat(line[diam+1:], 64)
			if err != nil {
				log.Fatalf("ERROR:%d: failed to parse tool definition %q: %v", count, line, err)
			}
			if _, present := tools[tool]; present {
				log.Fatalf("ERROR:%d: redefined tool %q", count, line)
			}
			tools[tool] = &Tool{
				Radius: 0.5 * width * factor,
			}
		case line[:1] == "T":
			tool := line[1:]
			if _, ok := tools[tool]; !ok {
				log.Fatalf("ERROR:%d: reference to undefined tool %q", count, tool)
			}
			current = tool
		case line[0] == 'X':
			at := strings.Index(line, "Y")
			x, err := strconv.ParseFloat(line[1:at], 64)
			if err != nil {
				log.Fatalf("ERROR:%d: unable to parse X value from %q: %v", count, line, err)
			}
			y, err := strconv.ParseFloat(line[at+1:], 64)
			if err != nil {
				log.Fatalf("ERROR:%d: unable to parse Y value from %q: %v", count, line, err)
			}
			use, ok := tools[current]
			if !ok {
				log.Fatalf("ERROR:%d: reference to undefined tool %q", count, current)
			}
			X, Y := x*factor, y*factor
			if !*keep {
				Y = -Y
			}
			use.Holes = append(use.Holes, polygon.Point{
				X: X,
				Y: Y,
			})
			if !started {
				ll = use.Holes[0]
				tr = use.Holes[0]
				started = true
			}
			if left := X - use.Radius; left < ll.X {
				ll.X = left
			}
			if right := X + use.Radius; right > tr.X {
				tr.X = right
			}
			if down := Y - use.Radius; down < ll.Y {
				ll.Y = down
			}
			if up := Y + use.Radius; up > tr.Y {
				tr.Y = up
			}
		default:
			fmt.Printf("WARNING:%d: ignoring %q\n", count, line)
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}

	svgPath := fmt.Sprintf("%s.drl.svg", *dest)
	s, err := os.Create(svgPath)
	if err != nil {
		log.Fatalf("failed to create %q: %v", svgPath, err)
	}
	defer s.Close()
	canvas := svgof.New(s)
	canvas.Decimals = 3
	canvas.StartviewUnit(tr.X-ll.X, tr.Y-ll.Y, "mm", ll.X, ll.Y, tr.X-ll.X, tr.Y-ll.Y)

	for _, use := range tools {
		dr := use.Radius - *dither/2
		if dr < *dither {
			dr = *dither
		}
		for _, pt := range use.Holes {
			flatCircle(canvas, pt.X, pt.Y, dr)
		}

		diameter := use.Radius * 2
		path := fmt.Sprintf("%s-C%.3f-PTH.drl", *dest, diameter)
		w, err := os.Create(path)
		if err != nil {
			log.Fatalf("unable to create %q: %v", path, err)
		}
		fmt.Fprintf(w, `M48
METRIC
T1C%.3f
%%
G90
G05
T1
`, diameter)
		for _, pt := range use.Holes {
			y := pt.Y
			if !*keep {
				y = -y
			}
			fmt.Fprintf(w, "X%.2fY%.2f\n", pt.X, y)
		}
		fmt.Fprintln(w, "M30")
		w.Close()
	}

	canvas.End()
}
