package escapes_test

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/tetsuo/escapes"
)

func sequentialANSI(rows int) []byte {
	var buf bytes.Buffer
	for range rows {
		buf.WriteString("\x1b[1;37m")
		for x := range 80 {
			buf.WriteByte(byte(0x41 + x%26))
		}
		buf.WriteString("\r\n")
	}
	return buf.Bytes()
}

func jumpANSI(jumps int) []byte {
	var buf bytes.Buffer
	for i := range jumps {
		y := (i*7)%200 + 1
		x := (i*13)%80 + 1
		buf.WriteString("\x1b[")
		writeDecimal(&buf, y)
		buf.WriteByte(0x3B)
		writeDecimal(&buf, x)
		buf.WriteByte(0x48)
		buf.WriteString("\x1b[1;32mTEST")
	}
	return buf.Bytes()
}

func writeDecimal(buf *bytes.Buffer, n int) {
	if n >= 100 {
		buf.WriteByte(byte(0x30 + n/100))
	}
	if n >= 10 {
		buf.WriteByte(byte(0x30 + (n/10)%10))
	}
	buf.WriteByte(byte(0x30 + n%10))
}

func BenchmarkRenderSequential(b *testing.B) {
	input := sequentialANSI(50)
	slab := make([]escapes.Cell, 80*50)
	b.ReportAllocs()
	bw := bufio.NewWriter(io.Discard)
	for b.Loop() {
		gr := escapes.NewRenderer(80, slab)
		escapes.Render(bw, input, gr)
	}
}

func BenchmarkRenderJumpy(b *testing.B) {
	input := jumpANSI(100)
	slab := make([]escapes.Cell, 80*200)
	gr := escapes.NewRenderer(80, slab)
	b.ReportAllocs()
	bw := bufio.NewWriter(io.Discard)
	for b.Loop() {
		gr.Reset()
		escapes.Render(bw, input, gr)
	}
}
