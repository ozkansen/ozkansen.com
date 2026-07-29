.PHONY: dev templ tailwind air build lint

install:
	npm install tailwindcss @tailwindcss/cli
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/air-verse/air@latest

# Tüm izleyicileri aynı anda başlatır
dev:
	make -j3 templ tailwind air

# Templ dosyalarını izler ve otomatik derler
templ:
	templ generate --watch

# Tailwind CSS dosyasını izler ve derler
tailwind:
	npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --watch

# Air ile Go sunucusunu canlı tutar
air:
	air

# Production için optimize edilmiş build alır
build:
	templ generate
	npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify
	go build -o ./bin/main ./cmd/web

lint:
	golangci-lint run ./...