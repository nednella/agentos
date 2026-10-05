// Command desktop is the agentos window.
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
	if addr := os.Getenv("AGENTOS_HTTP"); addr != "" {
		hub := &eventHub{subs: map[chan []byte]struct{}{}}
		app := newApp(cfg, host{emit: hub.emit, clipboard: func(string) {}}, execRunner)
		quitOnSignal(app)
		return serveHTTP(app, hub, assets, addr)
	}

	var window atomic.Pointer[context.Context]
	app := newApp(cfg, host{
		emit: func(event string, payload any) {
			if c := window.Load(); c != nil {
				runtime.EventsEmit(*c, event, payload)
			}
		},
		clipboard: func(text string) {
			if c := window.Load(); c != nil {
				_ = runtime.ClipboardSetText(*c, text)
			}
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
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup: func(c context.Context) {
			window.Store(&c)
			stripMenuShortcuts()
			if err := app.start(c); err != nil {
				log.Printf("agentos: %v", err)
			}
		},
		OnShutdown: func(context.Context) { app.stop() },
		Bind:       []any{app},
		// The Edit menu is what makes ⌘C, ⌘V and ⌘X reach the web view on macOS. Select All loses its key in stripMenuShortcuts.
		Menu: menu.NewMenuFromItems(menu.AppMenu(), menu.EditMenu(), menu.WindowMenu()),
		Mac: &mac.Options{
			TitleBar:   mac.TitleBarHiddenInset(),
			Appearance: mac.NSAppearanceNameDarkAqua,
		},
	})
}

// quitOnSignal stops the app's terminals before the process ends.
func quitOnSignal(app *App) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-quit
		app.stop()
		os.Exit(0)
	}()
}
