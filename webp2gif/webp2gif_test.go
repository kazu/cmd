package main

import (
	"os"
	"testing"
)

func Test_main(t *testing.T) {

	os.Args = []string{"webp2gif", "../tmp/tumblr_1.webp", "../tmp/tetetete.gif"}
	main()
}
