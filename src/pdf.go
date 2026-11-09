package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Native text extraction only. The uploaded bytes stay local; no OCR or cloud upload.
func (a *App) pdfText(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		fail(w, 501, errors.New("PDF text extraction is supported on macOS and Linux servers."))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 12*1024*1024)
	var v struct {
		Data string `json:"data"`
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(&v); e != nil {
		fail(w, 400, errors.New("Invalid PDF request format or size"))
		return
	}
	if d.Decode(&struct{}{}) != io.EOF {
		fail(w, 400, errors.New("Unexpected trailing request content"))
		return
	}
	b, e := base64.StdEncoding.DecodeString(v.Data)
	if e != nil || len(b) > 8*1024*1024 || !bytes.HasPrefix(b, []byte("%PDF-")) {
		fail(w, 400, errors.New("Please upload a PDF no larger than 8 MB"))
		return
	}
	f, e := os.CreateTemp("", "investone-pdf-*.pdf")
	if e != nil {
		fail(w, 500, errors.New("Could not create a local temporary file"))
		return
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		_ = f.Close()
		fail(w, 500, errors.New("Could not save the temporary PDF"))
		return
	}
	_ = f.Close()
	script := `ObjC.import('Foundation'); ObjC.import('PDFKit'); function run(argv) { var doc = $.PDFDocument.alloc.initWithURL($.NSURL.fileURLWithPath(argv[0])); var count = Number(doc.pageCount); if (!count || count > 500) throw new Error('Invalid PDF or more than 500 pages'); var out = ''; for (var i = 0; i < count; i++) { var page = doc.pageAtIndex(i); var text = ObjC.unwrap(page.string); if (text) out += '\n[Page ' + (i+1) + ']\n' + text; if(out.length > 400000) throw new Error('Text exceeds 400000 characters'); } return out; }`
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	var out []byte
 if runtime.GOOS=="linux" {
  out,e=exec.CommandContext(ctx,"pdftotext","-layout","-nopgbrk",f.Name(),"-").Output()
 }else{
  out,e=exec.CommandContext(ctx,"/usr/bin/osascript","-l","JavaScript","-e",script,f.Name()).Output()
 }
	if e != nil {
		fail(w, 422, errors.New("Could not extract this PDF. It may be scanned, encrypted or unsupported; paste the text instead."))
		return
	}
	if len(out)>800000 { fail(w,413,errors.New("Extracted PDF text exceeds 800 KB"));return }
 text := strings.TrimSpace(string(out))
	if len([]rune(text)) < 20 {
		fail(w, 422, errors.New("Not enough extractable PDF text. Scanned pages are not OCR'd; paste the document text instead."))
		return
	}
	send(w, 200, map[string]any{"text": text, "source": "Extracted PDF text (no OCR)", "characters": len([]rune(text))})
}
