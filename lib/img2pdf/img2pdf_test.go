package img2pdf_test

import (
	"testing"

	"github.com/kazu/cmd/lib/img2pdf"
)

func Test_ConvertToPDF(t *testing.T) {

	img2pdf.ConvertToPDF("../../tmp/img2pdf/test.zip", "../../tmp/img2pdf/test.pdf", img2pdf.UseZip(true))

}
