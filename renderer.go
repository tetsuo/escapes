package escapes

import (
	"bufio"
	"strconv"
)

// Cell is a single character cell. ch is the raw CP437 byte (0 = empty).
// At 4 bytes per cell, a full 80×200 canvas costs 64 KB.
type Cell struct {
	ch    uint8
	fg    uint8
	bg    uint8
	flags uint8
}

const (
	flagBright     = 1 << 0
	flagBgExplicit = 1 << 1
	flagBlink      = 1 << 2
)

// Renderer holds the rendering state and grid buffer.
type Renderer struct {
	width      int
	grid       []Cell
	cursorX    int
	cursorY    int
	maxY       int
	fg         uint8
	bg         uint8
	flags      uint8
	wrapMode   bool
	savedX     int
	savedY     int
	savedFg    uint8
	savedBg    uint8
	savedFlags uint8
	// rowWidth[y] tracks index+1 of the last non-empty cell in row y.
	rowWidth []int
}

// NewRenderer creates a new Renderer with the given width and buffer.
func NewRenderer(width int, slab []Cell) *Renderer {
	return &Renderer{
		width:    width,
		grid:     slab,
		rowWidth: make([]int, len(slab)/width+1),
		fg:       37,
		bg:       40,
		wrapMode: true,
	}
}

// Render processes CP437-encoded ANSI art from raw bytes into bw.
func Render(bw *bufio.Writer, input []byte, gr *Renderer) error {
	var params [8]int
	i, n := 0, len(input)

	for i < n {
		b := input[i]
		i++

		// Private mode sequences: \x1b[?Nh \x1b[?Nl
		if b == 0x1B && i+2 < n && input[i] == '[' && input[i+1] == '?' {
			j := i + 2
			val := 0
			for j < n && input[j] >= '0' && input[j] <= '9' {
				val = val*10 + int(input[j]-'0')
				j++
			}
			if j < n {
				switch input[j] {
				case 'h':
					if val == 7 {
						gr.wrapMode = true
					}
				case 'l':
					if val == 7 {
						gr.wrapMode = false
					}
				}
				i = j + 1
				continue
			}
		}

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
					switch c {
					case 'A':
						d := params[0]
						if d == 0 {
							d = 1
						}
						gr.cursorY -= d
						if gr.cursorY < 0 {
							gr.cursorY = 0
						}
					case 'B':
						d := params[0]
						if d == 0 {
							d = 1
						}
						gr.cursorY += d
					case 'C':
						d := params[0]
						if d == 0 {
							d = 1
						}
						gr.cursorX += d
					case 'D':
						d := params[0]
						if d == 0 {
							d = 1
						}
						gr.cursorX -= d
						if gr.cursorX < 0 {
							gr.cursorX = 0
						}
					case 'H', 'f':
						y, x := 1, 1
						if pIdx > 0 && params[0] > 0 {
							y = params[0]
						}
						if pIdx > 1 && params[1] > 0 {
							x = params[1]
						}
						gr.cursorX, gr.cursorY = x-1, y-1
					case 's':
						gr.savedX, gr.savedY = gr.cursorX, gr.cursorY
						gr.savedFg, gr.savedBg, gr.savedFlags = gr.fg, gr.bg, gr.flags
					case 'u':
						gr.cursorX, gr.cursorY = gr.savedX, gr.savedY
						gr.fg, gr.bg, gr.flags = gr.savedFg, gr.savedBg, gr.savedFlags
					case 'K':
						if params[0] == 0 { // cursor to end of line
							gr.eraseToEOL()
						}
					case 'm':
						if pIdx == 0 {
							gr.fg, gr.bg, gr.flags = 37, 40, 0
							break
						}
						for j := 0; j < pIdx; j++ {
							p := params[j]
							switch {
							case p == 0:
								gr.fg, gr.bg, gr.flags = 37, 40, 0
							case p == 1:
								gr.flags |= flagBright
							case p == 5:
								gr.flags |= flagBlink
							case p == 22:
								gr.flags &^= flagBright
							case p == 25:
								gr.flags &^= flagBlink
							case p == 27:
								// Reverse video off
							case p == 39:
								gr.fg = 37
							case p == 49:
								gr.bg = 40
								gr.flags &^= flagBgExplicit
							case p >= 30 && p <= 37:
								gr.fg = uint8(p)
							case p >= 40 && p <= 47:
								gr.bg = uint8(p)
								gr.flags |= flagBgExplicit
							}
						}
					case 'J':
						if params[0] == 2 {
							gr.cursorX, gr.cursorY = 0, 0
						}
					}
					break
				}
			}
			continue
		}

		switch b {
		case 0x0D:
			gr.cursorX = 0
		case 0x0A:
			gr.cursorY++
		case 0x1A, 0x00:
			goto done
		default:
			gr.putChar(b)
		}
	}

done:
	bw.WriteString("\x1b[2J\x1b[H\x1b[0m\x1b[?7l")
	gr.draw(bw)
	bw.WriteString("\n\x1b[?7h")
	return nil
}

func (gr *Renderer) draw(bw *bufio.Writer) {
	lastFg, lastBg, lastFlags := uint8(255), uint8(255), uint8(255)
	var intBuf [10]byte

	for y := 0; y <= gr.maxY; y++ {
		off := y * gr.width

		lineWidth := gr.width
		if y < len(gr.rowWidth) && gr.rowWidth[y] < lineWidth {
			lineWidth = gr.rowWidth[y]
		}

		for x := 0; x < lineWidth; x++ {
			c := gr.grid[off+x]
			b := c.ch
			fg, bg, flags := c.fg, c.bg, c.flags
			if b == 0 {
				fg, bg, flags = 37, 40, 0
			}

			if fg != lastFg || bg != lastBg || flags != lastFlags {
				bw.WriteString("\x1b[")
				if flags&flagBright != 0 {
					bw.WriteString("1;")
				} else {
					bw.WriteString("0;")
				}
				if flags&flagBlink != 0 {
					bw.WriteString("5;")
				}
				bw.Write(strconv.AppendInt(intBuf[:0], int64(fg), 10))
				bw.WriteByte(';')
				if flags&flagBgExplicit != 0 {
					bw.Write(strconv.AppendInt(intBuf[:0], int64(bg), 10))
				} else {
					bw.WriteString("49")
				}
				bw.WriteByte('m')
				lastFg, lastBg, lastFlags = fg, bg, flags
			}

			if b == 0 {
				bw.WriteByte(' ')
			} else {
				bw.Write(cp437utf8[b][:cp437utf8len[b]])
			}
		}
		bw.WriteString("\x1b[0m\n")
		lastFg, lastBg, lastFlags = 255, 255, 255
	}
}

// Reset clears state and zeroes the used portion of the slab for reuse.
func (gr *Renderer) Reset() {
	used := min((gr.maxY+1)*gr.width, len(gr.grid))
	clear(gr.grid[:used])
	clear(gr.rowWidth[:min(gr.maxY+1, len(gr.rowWidth))])
	gr.cursorX, gr.cursorY, gr.maxY = 0, 0, 0
	gr.fg, gr.bg, gr.flags = 37, 40, 0
	gr.wrapMode = true
	gr.savedX, gr.savedY = 0, 0
	gr.savedFg, gr.savedBg, gr.savedFlags = 37, 40, 0
}

func (gr *Renderer) putChar(b uint8) {
	if gr.cursorX >= 0 && gr.cursorX < gr.width && gr.cursorY >= 0 {
		idx := gr.cursorY*gr.width + gr.cursorX
		if idx < len(gr.grid) {
			gr.grid[idx] = Cell{b, gr.fg, gr.bg, gr.flags}
			if gr.cursorY > gr.maxY {
				gr.maxY = gr.cursorY
			}
			if gr.cursorY < len(gr.rowWidth) && gr.cursorX+1 > gr.rowWidth[gr.cursorY] {
				gr.rowWidth[gr.cursorY] = gr.cursorX + 1
			}
		}
	}
	gr.cursorX++
	if gr.wrapMode && gr.cursorX >= gr.width {
		gr.cursorX = 0
		gr.cursorY++
	}
}

// eraseToEOL clears from the current cursor position to the end of the line (CSI K).
func (gr *Renderer) eraseToEOL() {
	if gr.cursorY < 0 {
		return
	}
	off := gr.cursorY * gr.width
	for x := gr.cursorX; x < gr.width; x++ {
		idx := off + x
		if idx >= len(gr.grid) {
			break
		}
		gr.grid[idx] = Cell{}
	}
	if gr.cursorY < len(gr.rowWidth) && gr.rowWidth[gr.cursorY] > gr.cursorX {
		gr.rowWidth[gr.cursorY] = gr.cursorX
	}
}
