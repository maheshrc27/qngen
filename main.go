package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/maheshrc27/qngen/components"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func main() {
	http.HandleFunc("/", Index)
	http.HandleFunc("/upload", Upload)
	http.HandleFunc("/qns", Questions)

	fmt.Println("Server is running at port 9001")
	err := http.ListenAndServe(":9001", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func Index(w http.ResponseWriter, r *http.Request) {
	page := components.Uploader("Question Generator from PDF")
	page.Render(context.Background(), w)
}

func Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	workDir, _ := os.Getwd()

	err := r.ParseMultipartForm(15 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	id, err := gonanoid.New(12)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uploadDir := filepath.Join(workDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ext := filepath.Ext(header.Filename)
	dst, err := os.Create(filepath.Join(uploadDir, fmt.Sprintf("%s%s", id, ext)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/qns?id=%s", id), http.StatusSeeOther)
}

func Questions(w http.ResponseWriter, r *http.Request) {
	fileId := r.URL.Query().Get("id")

	text := GenerateText(fileId)
	w.Write([]byte(text))
}

func GenerateText(fileid string) string {
	workDir, _ := os.Getwd()
	filepath := filepath.Join(workDir, "uploads", fileid)

	fmt.Println(filepath)

	cmd := exec.Command("pdftotext", filepath+".pdf", "-q", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(stderr)
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)

	return string(buf.String())
}
