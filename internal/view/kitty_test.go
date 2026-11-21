package view

import (
	"bufio"
	"bytes"
	"image"
	"strings"
	"testing"
)

func TestKittyUploadFramesPayloadAfterSemicolon(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	kittyUpload(w, img, 7)
	kittyPlace(w, 7, 0, 4, 4, 10, 5)
	_ = w.Flush()
	got := buf.String()
	if !strings.HasPrefix(got, "\x1b_Ga=t,f=32,o=z,s=4,v=4,i=7,q=2,") {
		t.Fatalf("bad upload prefix: %q", got[:min(80, len(got))])
	}
	if !strings.Contains(got, ";") {
		t.Fatal("expected base64 payload after semicolon")
	}
	wantPlace := "\x1b7\x1b[H\x1b_Ga=p,i=7,p=1,x=0,y=0,w=4,h=4,c=10,r=5,C=1,q=2\x1b\\\x1b8"
	if !strings.HasSuffix(got, wantPlace) {
		t.Fatalf("bad place suffix: %q", got[max(0, len(got)-len(wantPlace)-20):])
	}
}
