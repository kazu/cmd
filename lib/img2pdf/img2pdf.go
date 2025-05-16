package img2pdf

import (
	"archive/zip"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"

	"github.com/kazu/cmd/ext"
	"github.com/schollz/progressbar/v3"
	"github.com/signintech/gopdf"
)

var isDebug = true
var dWriter io.Writer = os.Stdout

const (
	InputImageNone = 0
	InputImageJpg  = 1
	InputImagePng  = 2
)

var encoder = map[int]func(io.Reader) (image.Image, error){
	InputImageJpg: jpeg.Decode,
	InputImagePng: png.Decode,
}

func ImageFromOpt(img image.Image, pageRect gopdf.Rect, itype int) (gopdf.ImageFromOption, error) {
	opt := gopdf.ImageFromOption{}
	opt.X = 0
	opt.Y = 0

	switch itype {
	case InputImageJpg:
		opt.Format = "jpeg"
	case InputImagePng:
		opt.Format = "png"
	default:
		return opt, fmt.Errorf("unsupported image type: %d", itype)
	}
	opt.Rect = &gopdf.Rect{
		W: float64(img.Bounds().Max.X),
		H: float64(img.Bounds().Max.Y),
	}
	if opt.Rect.W < pageRect.W {
		opt.X = (pageRect.W - opt.Rect.W) / 2
	}
	if opt.Rect.H < pageRect.H {
		opt.Y = (pageRect.H - opt.Rect.H) / 2
	}

	return opt, nil
}

type convertOpt struct {
	isInputZip  bool `default:"false"`
	imageFormat int  `default:"0"`
}

func UseZip(t bool) ext.OptFunc[convertOpt] {
	return func(opt *convertOpt) ext.OptFunc[convertOpt] {
		opt.isInputZip = t
		return nil
	}
}
func Debug(t bool) ext.OptFunc[convertOpt] {
	return func(opt *convertOpt) ext.OptFunc[convertOpt] {
		isDebug = t
		if !isDebug {
			dWriter = io.Discard
		}
		return nil
	}
}

func MaxRect(files []*zip.File, opt *convertOpt) gopdf.Rect {

	max := gopdf.Rect{
		W: 0,
		H: 0,
	}
	for _, file := range files {
		r, _ := file.Open()
		defer r.Close()
		img, _ := encoder[opt.imageFormat](r)
		// if max.W < float64(img.Bounds().Max.X) {
		// 	max.W = float64(img.Bounds().Max.X)
		// }
		if max.H < float64(img.Bounds().Max.Y) {
			max.W = float64(img.Bounds().Max.X)
			max.H = float64(img.Bounds().Max.Y)
		}
	}

	return max
}

func ConvertToPDF(imagesrc string, output string, opts ...ext.OptFunc[convertOpt]) error {

	opt := ext.NewWithDefault[convertOpt]()
	ext.HandleOpt(opt, opts...)

	if !opt.isInputZip {
		return fmt.Errorf("only support zip input")
	}

	f, err := os.Open(imagesrc)
	if err != nil {
		return fmt.Errorf("1failed to open %s by os.Open  file: %v", imagesrc, err)
	}
	defer f.Close()

	//zipfile, err := zip.OpenReader(imagesrc)
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("2failed to open %s by os.Stat file: %v", imagesrc, err)
	}

	zipfile, err := zip.NewReader(f, stat.Size())

	if err != nil {
		return fmt.Errorf("3failed to open %s  file: %v", imagesrc, err)
	}

	//	pdf := gopdf.GoPdf{}
	var pdf *gopdf.GoPdf

	var maxRect gopdf.Rect

	count := len(zipfile.File)
	bar := progressbar.Default(int64(count), "converting")

	for _, file := range zipfile.File {
		fmt.Fprintf(dWriter, "loading: %s\n", file.Name)
		if opt.imageFormat == InputImageNone {
			opt.imageFormat = inputFormat(file.Name)
		}
		if opt.imageFormat == InputImageNone {
			continue
		}

		r, err := file.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "SKIP: %s : err=%v\n", file.Name, err)
			continue
		}
		defer r.Close()
		oimg, err := encoder[opt.imageFormat](r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SKIP: %s : encodefail=%v\n", file.Name, err)
			continue

		}
		if pdf == nil {

			maxRect = MaxRect(zipfile.File, opt)
			fmt.Fprintf(dWriter, "pdf size=%f, %f\n", maxRect.W, maxRect.H)

			pdf = &gopdf.GoPdf{}

			//img.Bounds().Max
			pdf.Start(gopdf.Config{PageSize: maxRect})

			// pdf.Start(gopdf.Config{PageSize: gopdf.Rect{
			// 	W: float64(img.Bounds().Max.X),
			// 	H: float64(img.Bounds().Max.Y),
			// }})
		}
		pdf.AddPage()
		fmt.Fprintf(dWriter, "add: %s\n", file.Name)
		//		err = pdf.ImageFrom(img, 0, 0, nil)
		img, err := scaleImage(oimg, int(maxRect.W), int(maxRect.H))
		if err != nil {
			fmt.Fprintf(os.Stderr, "SKIP: %s : scale fail=%v\n", file.Name, err)
			img = oimg
		}
		iopt, err := ImageFromOpt(img, maxRect, opt.imageFormat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SKIP: %s : imgopt fail=%v\n", file.Name, err)
			continue
		}
		err = pdf.ImageFromWithOption(img, iopt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SKIP: %s : write fail=%v\n", file.Name, err)
			continue

		}
		bar.Add(1)
	}
	if pdf != nil {
		fmt.Fprintf(dWriter, "finish: %s\n", output)
		return pdf.WritePdf(output)
	}

	return nil
}

func scaleImage(img image.Image, w, h int) (image.Image, error) {
	if img == nil {
		return nil, fmt.Errorf("image is nil")
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid width or height")
	}

	ratio := float64(0)

	ratio = float64(w) / float64(img.Bounds().Max.X)

	if ratio > float64(h)/float64(img.Bounds().Max.Y) {
		ratio = float64(h) / float64(img.Bounds().Max.Y)
	}

	dst := image.NewRGBA(image.Rect(0, 0, int(float64(img.Bounds().Max.X)*ratio), int(float64(img.Bounds().Max.Y)*ratio)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	fmt.Fprintf(dWriter, "\tscale: %d:%d -> %d:%d\n",
		img.Bounds().Max.X, img.Bounds().Max.Y,
		dst.Bounds().Max.X,
		dst.Bounds().Max.Y)

	return dst, nil
}

func inputFormat(fname string) int {

	switch filepath.Ext(fname) {
	case ".jpg", ".jpeg":
		return InputImageJpg
	case ".png":
		return InputImagePng
	default:
		return InputImageNone
	}
}
