.PHONY: build clean test app-build app-bundle app-run

BINARY := obsidian-daily-macos

build:
	CGO_ENABLED=0 go build -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY)
	rm -rf ObsidianDaily.app
	cd app && swift package clean

app-build:
	cd app && swift build -c release

app-bundle: build app-build
	mkdir -p ObsidianDaily.app/Contents/MacOS
	mkdir -p ObsidianDaily.app/Contents/Resources
	cp app/.build/release/ObsidianDaily ObsidianDaily.app/Contents/MacOS/
	cp $(BINARY) ObsidianDaily.app/Contents/Resources/
	cp app/Sources/ObsidianDaily/Resources/Info.plist ObsidianDaily.app/Contents/

app-run:
	cd app && OBSIDIAN_VAULT_ROOT="$$(fish -c 'echo $$OBSIDIAN_VAULT_ROOT')" swift run ObsidianDaily
