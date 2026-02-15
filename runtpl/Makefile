CONTAINER_ID=$(shell cat /etc/hostname)
HOST_PWD=$(shell docker inspect $(CONTAINER_ID) -f "{{range  .Mounts}}{{if eq .Destination \"/workspaces\"  }}{{.Source}}/cmd{{end}}{{end}}")

ESBUILD := npx esbuild
TS_SRC := openai_tts_realtime_t.ts
OUT := openai_tts_realtime_t.js

EPOCH=$(shell date +%s)

.DEFAULT_GOAL := build
.PHONY: build serve install clean prune echo embed inline
EMBED_SCRIPT := scripts/embed_ts.js

openai_tts_realtime_t.js: *.ts
	@echo "Type-checking with npx tsc --noEmit"
	@npx tsc --noEmit
	$(ESBUILD) $(TS_SRC) --bundle --format=esm --target=es2020 --outfile=./$(OUT) --sourcemap

build: embed

serve:
	npx http-server . -p 8000

install:
	npm ci

# Build the TS and inline the bundled JS into the HTML.
# Usage: `make embed` or `make inline` (depends on `scripts/embed_ts.js`).
.PHONY: embed inline
embed: openai_tts_realtime_t.js
	@if [ -f $(EMBED_SCRIPT) ]; then \
		echo "Inlining $(OUT) into openai_tts_realtime_t.html using $(EMBED_SCRIPT)"; \
		if node $(EMBED_SCRIPT) openai_tts_realtime_t.html openai_tts_realtime_t.ts; then \
			echo "Embed succeeded — removing backup openai_tts_realtime_t.html.bak"; \
			rm -f openai_tts_realtime_t.html.bak; \
		else \
			echo "Embed failed — backup kept as openai_tts_realtime_t.html.bak"; \
			exit 1; \
		fi; \
	else \
		echo "$(EMBED_SCRIPT) が見つかりません。まず scripts/embed_ts.js を作成してください。"; \
		exit 1; \
	fi

inline: embed
clean:
	rm -f $(OUT) $(OUT).map
	@if [ -d node_modules ]; then echo "node_modules exists. Run 'make prune' to remove it."; fi

prune:
	# Remove node_modules and package-lock if you really want to prune dependencies
	rm -rf node_modules package-lock.json
	-git rm -r --cached node_modules || true

build_docker:
	docker build .

install_to_docker: openai_tts_realtime_t.js
	cp tmpl/test.html tmpl/${EPOCH}.test.html
	cp openai_tts_realtime_t.js tmpl/.
	cp openai_tts_realtime_t.html tmpl/.
	cd tmpl/ && ln -sf openai_tts_realtime_t.html test.html

echo:
	echo ${HOST_PWD}

# openai_tts_realtime_t.html  
# openai_tts_realtime_t.js