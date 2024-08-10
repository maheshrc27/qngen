package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/maheshrc27/qngen/components"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"google.golang.org/api/option"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	dir := http.Dir("./static")
	fileServer := http.FileServer(dir)
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))

	http.HandleFunc("/", Index)
	http.HandleFunc("/upload", Upload)
	http.HandleFunc("/qns", Questions)

	fmt.Println("Server is running at port 9001")
	err = http.ListenAndServe(":9001", nil)
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

	prompt := fmt.Sprintf(`"You are a professor in the field of Computer science and physics. You have to provide notes for the students as questions and answer form."
	"Using the provided text, write the questions and answer which becomes easy for students to study. Create 30 realistic exam questions covering the entire content. Provide the output in JSON format."
	"The JSON should have the structure: [{"question": "...", "explanation": "..."}, ...]. Ensure the JSON is valid and properly formatted and can be unmarshalled into go structs."
	"The text is : %s"`, text)

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	model.ResponseMIMEType = "application/json"

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatal(err)
	}
	var out string
	var questions []components.Question

	for _, c := range resp.Candidates {
		if c.Content != nil {
			out = fmt.Sprintf("%s", *c.Content)
		}
	}
	out = strings.TrimPrefix(out, "{[")
	out = strings.TrimSuffix(out, "] model}")

	err = json.Unmarshal([]byte(out), &questions)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	page := components.Viewer("Question Generator from PDF", questions)
	page.Render(context.Background(), w)
}

func GenerateText(fileid string) string {
	workDir, _ := os.Getwd()
	filepath := filepath.Join(workDir, "uploads", fileid)

	cmd := exec.Command("pdftotext", filepath+".pdf", "-q", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)

	return string(buf.String())
}
