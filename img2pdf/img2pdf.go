package main

import (
	"os"
	"strings"

	"github.com/kazu/cmd/lib/img2pdf"
)

func main() {

	if len(os.Args) < 3 {
		println("Usage: img2pdf <input> <output>")
		return
	}

	src := strings.ReplaceAll(os.Args[1], "'", "")
	dst := strings.ReplaceAll(os.Args[2], "'", "")
	// src := os.Args[1]
	// dst := os.Args[2]

	err := img2pdf.ConvertToPDF(src, dst, img2pdf.UseZip(true), img2pdf.Debug(false))

	if err != nil {
		panic(err)
	}
}
