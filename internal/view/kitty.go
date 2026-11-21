package view

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
)

type encoded struct {
	data []byte
	w, h int
}

// encodeImage packs RGBA as zlib-compressed raw pixels (Kitty f=32,o=z),
// matching the format Rex/rexpad already display.
func encodeImage(img *image.RGBA) encoded {
	b := img.Bounds()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	row := make([]byte, b.Dx()*4)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		src := img.Pix[img.PixOffset(b.Min.X, y):][:len(row)]
		for i := 0; i < len(row); i += 4 {
			switch a := src[i+3]; a {
			case 0, 255:
				copy(row[i:i+4], src[i:i+4])
			default:
				// image.RGBA is premultiplied; the protocol wants straight alpha.
				for c := 0; c < 3; c++ {
					row[i+c] = uint8(min(255, (int(src[i+c])*255+int(a)/2)/int(a)))
				}
				row[i+3] = a
			}
		}
		_, _ = zw.Write(row)
	}
	_ = zw.Close()
	return encoded{buf.Bytes(), b.Dx(), b.Dy()}
}

func kittyUpload(w *bufio.Writer, img *image.RGBA, id int) {
	e := encodeImage(img)
	data := base64.StdEncoding.EncodeToString(e.data)
	const chunk = 4096
	for i := 0; i < len(data); i += chunk {
		end := min(i+chunk, len(data))
		more := 0
		if end < len(data) {
			more = 1
		}
		if i == 0 {
			fmt.Fprintf(w, "\x1b_Ga=t,f=32,o=z,s=%d,v=%d,i=%d,q=2,m=%d;%s\x1b\\", e.w, e.h, id, more, data[i:end])
		} else {
			fmt.Fprintf(w, "\x1b_Gm=%d;%s\x1b\\", more, data[i:end])
		}
	}
}

func kittyPlace(w *bufio.Writer, id, y, width, height, cols, rows int) {
	fmt.Fprintf(w, "\x1b7\x1b[H\x1b_Ga=p,i=%d,p=1,x=0,y=%d,w=%d,h=%d,c=%d,r=%d,C=1,q=2\x1b\\\x1b8",
		id, y, width, height, cols, rows)
}

func kittyDelete(w *bufio.Writer, id int) {
	fmt.Fprintf(w, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}
