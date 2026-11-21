package view

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"bytes"
)

func encodeImage(img *image.RGBA) string {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func kittyUpload(w *bufio.Writer, img *image.RGBA, id int) {
	data := encodeImage(img)
	chunk := 4096
	for i := 0; i < len(data); i += chunk {
		end := min(i+chunk, len(data))
		more := 1
		if end == len(data) {
			more = 0
		}
		fmt.Fprintf(w, "\x1b_G%s,i=%d,m=%d,f=100,a=T,p=%d,q=2\x1b\\",
			data[i:end], id, more, id)
	}
}

func kittyPlace(w *bufio.Writer, id, y, width, height, cols, rows int) {
	fmt.Fprintf(w, "\x1b7\x1b_Ga=p,i=%d,p=1,x=0,y=%d,w=%d,h=%d,c=%d,r=%d,C=1,q=2\x1b\\\x1b8",
		id, y, width, height, cols, rows)
}

func kittyDelete(w *bufio.Writer, id int) {
	fmt.Fprintf(w, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}