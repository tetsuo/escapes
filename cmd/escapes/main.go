package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	esc "github.com/tetsuo/escapes"
)

func main() {
	inputFile := flag.String("i", "", "input file")

	flag.Parse()

	var input []byte
	var err error

	if *inputFile != "" {
		input, err = os.ReadFile(*inputFile)
	} else {
		input, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "escapes: %v\n", err)
		os.Exit(1)
	}

	const width = 80
	slab := make([]esc.Cell, slabSize(input, width))

	bw := bufio.NewWriterSize(os.Stdout, 65536)
	gr := esc.NewRenderer(width, slab)
	if err := esc.Render(bw, input, gr); err != nil {
		fmt.Fprintf(os.Stderr, "escapes: %v\n", err)
		os.Exit(1)
	}
	bw.Flush()
}

// slabSize scans input to find the exact canvas dimensions at the given width
// and returns the number of cells required.
func slabSize(input []byte, width int) int {
	var params [8]int
	cursorX, cursorY, maxY := 0, 0, 0
	savedX, savedY := 0, 0
	i, n := 0, len(input)
	for i < n {
		b := input[i]
		i++
		if b == 0x1B && i < n && input[i] == '[' {
			i++
			pIdx, val, hasVal := 0, 0, false
			params = [8]int{}
			for i < n {
				c := input[i]
				i++
				if c >= '0' && c <= '9' {
					val = val*10 + int(c-'0')
					hasVal = true
				} else if c == ';' {
					if hasVal && pIdx < 8 {
						params[pIdx] = val
						pIdx++
					}
					val, hasVal = 0, false
				} else {
					if hasVal && pIdx < 8 {
						params[pIdx] = val
						pIdx++
					}
					if pIdx == 0 {
						pIdx, params[0] = 1, 1
					}
					switch c {
					case 'A':
						cursorY -= params[0]
						if cursorY < 0 {
							cursorY = 0
						}
					case 'B':
						cursorY += params[0]
						if cursorY > maxY {
							maxY = cursorY
						}
					case 'C':
						cursorX += params[0]
					case 'D':
						cursorX -= params[0]
						if cursorX < 0 {
							cursorX = 0
						}
					case 'H', 'f':
						y, x := 1, 1
						if pIdx > 0 && params[0] > 0 {
							y = params[0]
						}
						if pIdx > 1 && params[1] > 0 {
							x = params[1]
						}
						cursorX, cursorY = x-1, y-1
						if cursorY > maxY {
							maxY = cursorY
						}
					case 's':
						savedX, savedY = cursorX, cursorY
					case 'u':
						cursorX, cursorY = savedX, savedY
						if cursorY > maxY {
							maxY = cursorY
						}
					case 'J':
						if pIdx > 0 && params[0] == 2 {
							cursorX, cursorY = 0, 0
						}
					}
					break
				}
			}
			continue
		}
		switch b {
		case 0x0D:
			cursorX = 0
		case 0x0A:
			cursorY++
			if cursorY > maxY {
				maxY = cursorY
			}
		case 0x1A, 0x00:
			goto done
		default:
			if cursorX >= 0 && cursorX < width {
				if cursorY > maxY {
					maxY = cursorY
				}
			}
			cursorX++
			if cursorX >= width {
				cursorX = 0
				cursorY++
				if cursorY > maxY {
					maxY = cursorY
				}
			}
		}
	}
done:
	return (maxY + 1) * width
}
