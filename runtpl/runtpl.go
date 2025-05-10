package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	fileDir = "./tmpl/"
)

func IsExist(dir string) bool {
	_, err := os.Stat(dir)

	return err == nil
}

func set(w http.ResponseWriter, r *http.Request) {

	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	name := r.FormValue("name")

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	fmt.Fprintf(w, "Uploaded File: %s\n", handler.Filename)
	fmt.Fprintf(w, "File Size: %d\n", handler.Size)
	fmt.Fprintf(w, "MIME Header: %v\n", handler.Header)

	if IsExist(filepath.Join(fileDir, name)) {
		os.Remove(filepath.Join(fileDir, name))
	}

	f, err := os.Create(filepath.Join(fileDir, name))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	io.Copy(f, file)

	fmt.Fprint(w, "File saved successfully\n")
}

func list(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(fileDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, file := range files {
		f, err := os.Open(filepath.Join(fileDir, file.Name()))
		if err != nil {
			continue
		}

		fmt.Fprintf(w, "File: %s\n", file.Name())
		b, err := io.ReadAll(f)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "Content: %s\n", string(b))
	}
}

type RunReq struct {
	URL     string         `json:"url"`
	Headers []string       `json:"headers"`
	Param   map[string]any `json:"param"`
}

func run(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	//name := r.FormValue("tmpl")
	param := r.Form
	name := param.Get("tmpl")

	f, err := os.Open(filepath.Join(fileDir, name))
	if err != nil {
		fmt.Fprint(os.Stderr, "Error1: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	var tmplReq RunReq
	if err := json.NewDecoder(f).Decode(&tmplReq); err != nil {
		fmt.Fprint(os.Stderr, "Error2: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for key, ivalue := range tmplReq.Param {
		value, ok := ivalue.(string)
		if !ok {
			continue
		}

		if !strings.HasPrefix(value, "$") {
			continue
		}
		fmt.Printf("value=%s\n", value[1:])

		if !param.Has(value[1:]) {
			continue
		}

		nVal, err := findBy(param.Get(value[1:]))
		if err != nil {
			continue
		}

		tmplReq.Param[key] = nVal
	}
	var b strings.Builder
	err = json.NewEncoder(&b).Encode(tmplReq.Param)
	if err != nil {
		fmt.Fprintf(os.Stderr, "str=%s Error3: %s\n", b.String(), err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("request body %s\n", b.String())

	in := strings.NewReader(b.String())

	nReq, err := http.NewRequest("POST", tmplReq.URL, in)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error4: ", err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, header := range tmplReq.Headers {
		kv := strings.Split(header, ":")
		if len(kv) != 2 {
			continue
		}
		nReq.Header.Add(kv[0], kv[1])
	}
	cli := http.Client{}

	resp, err := cli.Do(nReq)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error5: ", err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctype := resp.Header.Get("Content-Type")

	fmt.Fprintf(w, "Response Content-Type: %s\n", ctype)

	w.Header().Set("Content-Type", ctype)
	io.Copy(w, resp.Body)
	defer resp.Body.Close()

}

func findBy(name string) (string, error) {
	f, err := os.Open(filepath.Join(fileDir, name))
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	fmt.Printf("findBy key=%s val=%s\n", name, string(b))
	return string(b), nil

}

func GetFile(name string, lnum int) (string, error) {
	f, err := os.Open(filepath.Join(fileDir, name))
	if err != nil {
		return "", err
	}
	defer f.Close()
	bio := bufio.NewReader(f)
	for i := 0; i < lnum; i++ {
		_, _ = bio.ReadString('\n')
	}
	fmt.Printf("GetFile key=%s skip=%d\n", name, lnum)

	b, err := io.ReadAll(bio)
	if err != nil {
		return "", err
	}
	if len(b) > 100 {
		fmt.Printf("mes=%s\n", string(b)[0:100])
	}
	return string(b), nil
}

func staticer(fname string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ae := r.Header.Get("Accept-Encoding")

		fmt.Printf("Accept-Encoding: %s\n", ae)

		r.ParseForm()
		param := r.Form
		message := param.Get("message")
		lnum := 0
		if param.Has("l") {
			lnum, _ = strconv.Atoi(param.Get("l"))
		}

		fmt.Printf("message=%s\n", message)

		f, err := os.Open(filepath.Join(fileDir, fname))
		if err != nil {
			fmt.Fprint(os.Stderr, "Error1: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")

		defer f.Close()

		val, err := GetFile(message, lnum)
		if err != nil {
			io.Copy(w, f)

			return

		}
		if len(message) == 0 || lnum == 0 {

			var b strings.Builder
			io.Copy(&b, f)
			w.Write([]byte(strings.ReplaceAll(b.String(), "[message]", val)))
			return
		}
		w.Header().Add("Content-Encoding", "gzip")

		//zf, err := zstd.NewWriter(w)
		zf := gzip.NewWriter(w)
		// if err != nil {
		// 	return
		// }

		defer zf.Close()
		var b strings.Builder
		io.Copy(&b, f)
		zf.Write([]byte(strings.ReplaceAll(b.String(), "[message]", val)))

	}
}

func test(w http.ResponseWriter, r *http.Request) {
	// f, err := os.Open(filepath.Join(fileDir, "test.html"))
	// if err != nil {
	// 	fmt.Fprint(os.Stderr, "Error1: ", err)
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// w.Header().Set("Content-Type", "text/html")
	// io.Copy(w, f)
	// defer f.Close()
	staticer("test.html")(w, r)
}

func run2(w http.ResponseWriter, r *http.Request) {

	in := r.Body
	nReq, err := http.NewRequest("POST", "http://10.2.1.15:8000/v1/audio/speech", in)
	if err != nil {
		fmt.Fprint(os.Stderr, "run2 Error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nReq.Header.Add("Content-Type", "application/json")
	nReq.Header.Add("Accept", "application/json")

	cli := http.Client{}

	resp, err := cli.Do(nReq)
	if err != nil {
		fmt.Fprint(os.Stderr, "run2 Error5: ", err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctype := resp.Header.Get("Content-Type")

	fmt.Fprintf(w, "Response Content-Type: %s\n", ctype)

	w.Header().Set("Content-Type", ctype)

	defer resp.Body.Close()
	io.Copy(w, resp.Body)

	return

}

var fav http.HandlerFunc = staticer("favicon.ico")

func main() {
	server := http.Server{
		Addr:    ":19999",
		Handler: nil,
	}

	http.HandleFunc("/set", set)
	http.HandleFunc("/list", list)
	http.HandleFunc("/run", run)
	http.HandleFunc("/run2", run2)
	http.HandleFunc("/test", test)
	http.HandleFunc("/favicon.ico", fav)

	server.ListenAndServe()
}
