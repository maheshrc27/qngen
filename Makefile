.PHONY: run
run:
	@go run main.go

.PHONY: templ/generate
templ/generate:
	@go run github.com/a-h/templ/cmd/templ@latest generate