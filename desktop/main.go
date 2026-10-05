// Command desktop is the agentos window.
package main

import (
	"context"
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
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
	app := newApp(cfg)

	return wails.Run(&options.App{
		Title:            "agentos",
		Width:            1500,
		Height:           950,
		MinWidth:         420,
		MinHeight:        360,
		BackgroundColour: &options.RGBA{R: 7, G: 9, B: 13, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup: func(c context.Context) {
			stripMenuShortcuts()
			app.start(c)
		},
		Bind: []any{app},
		// The Edit menu is what makes ⌘C, ⌘V and ⌘X reach the web view on macOS. Select All loses its key in stripMenuShortcuts.
		Menu: menu.NewMenuFromItems(menu.AppMenu(), menu.EditMenu(), menu.WindowMenu()),
		Mac: &mac.Options{
			TitleBar:   mac.TitleBarHiddenInset(),
			Appearance: mac.NSAppearanceNameDarkAqua,
		},
	})
}
