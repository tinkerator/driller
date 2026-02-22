// Program driller converts a combined -PTH.drl file into one-per tool
// drl files and a combined summary SVG file. This has only been
// tested with kicad generated drl files.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"zappem.net/pub/math/polygon"
)

var (
	src  = flag.String("src", "", "required input *-PTH.drl file")
	dest = flag.String("dest", "./default", "destination path for output")
)

// Tool defines drill usage details in mm units.
type Tool struct {
	Diameter float64
	Holes    []polygon.Point
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

	sc := bufio.NewScanner(f)
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
				Diameter: width * factor,
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
			use.Holes = append(use.Holes, polygon.Point{
				X: x * factor,
				Y: y * factor,
			})
		default:
			fmt.Printf("WARNING:%d: ignoring %q\n", count, line)
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
	for _, use := range tools {
		path := fmt.Sprintf("%s-C%.3f-PTH.drl", *dest, use.Diameter)
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
`, use.Diameter)
		for _, pt := range use.Holes {
			fmt.Fprintf(w, "X%.2fY%.2f\n", pt.X, pt.Y)
		}
		fmt.Fprintln(w, "M30")
		w.Close()
	}
}
