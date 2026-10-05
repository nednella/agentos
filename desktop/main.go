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

	"github.com/nednella/agentos/desktop/devhttp"
	"github.com/nednella/agentos/desktop/internal/app"
	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func main() {
	if err := launch(); err != nil {
		log.Fatalf("agentos: %v", err)
	}
}

func launch() error {
	fixLocale()
	if err := fixPath(); err != nil {
		log.Printf("agentos: %v", err)
	}
	cfg, err := app.Load()
	if err != nil {
		return err
	}
	assets, err := fs.Sub(assetsFS, assetsRoot)
	if err != nil {
		return err
	}
	if addr := os.Getenv("AGENTOS_HTTP"); addr != "" {
		hub := devhttp.NewHub()
		a := app.New(cfg, app.Host{Emit: hub.Emit, Clipboard: func(string) {}, PickDir: func() (string, error) { return "", nil }}, run.Exec, run.ExecEnv)
		quitOnSignal(a)
		return devhttp.Serve(devhttp.Options{Services: a.Services(), Start: a.Start, Stop: a.Stop, Hub: hub, Assets: assets, Media: a.Media(), Addr: addr})
	}

	var window atomic.Pointer[context.Context]
	a := app.New(cfg, app.Host{
		Emit: func(event string, payload any) {
			if c := window.Load(); c != nil {
				runtime.EventsEmit(*c, event, payload)
			}
		},
		Clipboard: func(text string) {
			if c := window.Load(); c != nil {
				_ = runtime.ClipboardSetText(*c, text)
			}
		},
		PickDir: func() (string, error) {
			return runtime.OpenDirectoryDialog(*window.Load(), runtime.OpenDialogOptions{Title: "Add a project folder", CanCreateDirectories: true})
		},
	}, run.Exec, run.ExecEnv)
	quitOnSignal(a)

	return wails.Run(&options.App{
		Title:            "agentos",
		Width:            1500,
		Height:           950,
		MinWidth:         420,
		MinHeight:        360,
		BackgroundColour: &options.RGBA{R: 7, G: 9, B: 13, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets, Handler: a.Media()},
		OnStartup: func(c context.Context) {
			window.Store(&c)
			stripMenuShortcuts()
			if err := a.Start(c); err != nil {
				log.Printf("agentos: %v", err)
			}
		},
		OnShutdown: func(context.Context) { a.Stop() },
		Bind:       a.Services(),
		// The Edit menu is what makes ⌘C, ⌘V and ⌘X reach the web view on macOS. Select All loses its key in stripMenuShortcuts.
		Menu: menu.NewMenuFromItems(menu.AppMenu(), menu.EditMenu(), menu.WindowMenu()),
		Mac: &mac.Options{
			TitleBar:   mac.TitleBarHiddenInset(),
			Appearance: mac.NSAppearanceNameDarkAqua,
		},
	})
}

// quitOnSignal stops the app's terminals before the process ends.
func quitOnSignal(a *app.App) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-quit
		a.Stop()
		os.Exit(0)
	}()
}
