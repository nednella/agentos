// Command desktop is the agentos window: the terminal UI's sessions, on a web front end.
package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("agentos: %v", err)
	}
}

func run() error {
	fixLocale()
	if err := fixPath(); err != nil {
		log.Printf("agentos: %v", err)
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	assets, err := fs.Sub(assetsFS, assetsRoot)
	if err != nil {
		return err
	}

	var window atomic.Pointer[context.Context]
	ctx := func() context.Context {
		if c := window.Load(); c != nil {
			return *c
		}
		return nil
	}
	if addr := os.Getenv("AGENTOS_HTTP"); addr != "" {
		hub := &eventHub{subs: map[chan []byte]struct{}{}}
		noop := func(string) {}
		noPicker := func() (string, error) { return "", nil }
		app := newApp(cfg, host{emit: hub.emit, clipboard: noop, openURL: noop, pickDir: noPicker}, execRunner)
		quitOnSignal(app)
		return serveHTTP(app, hub, assets, addr)
	}

	app := newApp(cfg, host{
		emit: func(event string, payload any) {
			if c := ctx(); c != nil {
				runtime.EventsEmit(c, event, payload)
			}
		},
		clipboard: func(text string) {
			if c := ctx(); c != nil {
				_ = runtime.ClipboardSetText(c, text)
			}
		},
		openURL: func(url string) {
			if c := ctx(); c != nil {
				runtime.BrowserOpenURL(c, url)
			}
		},
		pickDir: func() (string, error) {
			return runtime.OpenDirectoryDialog(*window.Load(), runtime.OpenDialogOptions{Title: "Add a project folder", CanCreateDirectories: true})
		},
	}, execRunner)

	quitOnSignal(app)

	return wails.Run(&options.App{
		Title:            "agentos",
		Width:            1500,
		Height:           950,
		MinWidth:         420,
		MinHeight:        360,
		BackgroundColour: &options.RGBA{R: 7, G: 9, B: 13, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets, Handler: app.mediaHandler()},
		OnStartup: func(c context.Context) {
			window.Store(&c)
			if err := app.start(c); err != nil {
				log.Printf("agentos: %v", err)
			}
		},
		OnShutdown: func(context.Context) { app.stop() },
		Bind:       []any{app},
		// The Edit menu is what makes ⌘C, ⌘V and ⌘A reach the web view on macOS.
		Menu: menu.NewMenuFromItems(menu.AppMenu(), menu.EditMenu(), menu.WindowMenu()),
		Mac: &mac.Options{
			TitleBar:   mac.TitleBarHiddenInset(),
			Appearance: mac.NSAppearanceNameDarkAqua,
		},
	})
}

// quitOnSignal stops the app's browsers and terminals before the process ends: a stopped app leaves no browser behind.
func quitOnSignal(app *App) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-quit
		app.stop()
		os.Exit(0)
	}()
}
