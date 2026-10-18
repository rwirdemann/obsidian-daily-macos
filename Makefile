.PHONY: build clean test app-build app-bundle app-run dmg icon install

BINARY := obsidian-daily-macos
VERSION := $(shell plutil -extract CFBundleShortVersionString raw \
	app/Sources/ObsidianDaily/Resources/Info.plist)
ICNS := app/Sources/ObsidianDaily/Resources/AppIcon.icns
DMG := dist/ObsidianDaily-$(VERSION)-arm64.dmg

build:
	CGO_ENABLED=0 go build -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY)
	rm -rf ObsidianDaily.app dist
	cd app && swift package clean

app-build:
	cd app && swift build -c release

app-bundle: build app-build
	mkdir -p ObsidianDaily.app/Contents/MacOS
	mkdir -p ObsidianDaily.app/Contents/Resources
	cp app/.build/release/ObsidianDaily ObsidianDaily.app/Contents/MacOS/
	cp $(BINARY) ObsidianDaily.app/Contents/Resources/
	cp app/Sources/ObsidianDaily/Resources/Info.plist ObsidianDaily.app/Contents/
	cp $(ICNS) ObsidianDaily.app/Contents/Resources/
	codesign --force --sign - ObsidianDaily.app/Contents/Resources/$(BINARY)
	codesign --force --sign - ObsidianDaily.app

install: app-bundle
	pkill -x ObsidianDaily || true
	while pgrep -x ObsidianDaily >/dev/null; do sleep 0.1; done
	rm -rf /Applications/ObsidianDaily.app
	ditto ObsidianDaily.app /Applications/ObsidianDaily.app
	open /Applications/ObsidianDaily.app

dmg: app-bundle
	rm -rf dist/dmg $(DMG)
	mkdir -p dist/dmg
	ditto ObsidianDaily.app dist/dmg/ObsidianDaily.app
	ln -s /Applications dist/dmg/Applications
	hdiutil create -volname "Obsidian Daily" -srcfolder dist/dmg \
		-format UDZO $(DMG)
	rm -rf dist/dmg

app-run:
	cd app && OBSIDIAN_VAULT_ROOT="$$(fish -c 'echo $$OBSIDIAN_VAULT_ROOT')" swift run ObsidianDaily

icon:
	rm -rf dist/AppIcon.iconset
	mkdir -p dist/AppIcon.iconset
	swift scripts/make-icon.swift dist/icon.png
	for s in 16 32 128 256 512; do \
		sips -z $$s $$s dist/icon.png \
			--out dist/AppIcon.iconset/icon_$${s}x$${s}.png >/dev/null; \
		sips -z $$((s*2)) $$((s*2)) dist/icon.png \
			--out dist/AppIcon.iconset/icon_$${s}x$${s}@2x.png >/dev/null; \
	done
	iconutil -c icns dist/AppIcon.iconset -o $(ICNS)
	rm -rf dist/AppIcon.iconset dist/icon.png
