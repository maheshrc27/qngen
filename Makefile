.PHONY: run
run:
	@go run main.go

.PHONY: templ/generate
templ/generate:
	@go run github.com/a-h/templ/cmd/templ@latest generate

.PHONY: tailwind/watch
tailwind/watch:
	./tailwindcss -i ./static/css/main.css -o ./static/css/tailwind.css --watch

.PHONY: tailwind/build
tailwind/build:
	./tailwindcss -i ./static/css/main.css -o ./static/css/tailwind.css --minify