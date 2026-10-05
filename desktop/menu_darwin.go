//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// Wails builds the app and edit menus natively and binds Quit to Cmd+Q and Select All
// to Cmd+A. The app uses both keys itself, so take them off those two items.
static void stripMenuShortcuts(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		for (NSMenuItem *top in [[NSApp mainMenu] itemArray]) {
			for (NSMenuItem *item in [[top submenu] itemArray]) {
				NSString *title = [item title];
				if ([title isEqualToString:@"Select All"] || [title hasPrefix:@"Quit "]) {
					[item setKeyEquivalent:@""];
				}
			}
		}
	});
}
*/
import "C"

func stripMenuShortcuts() { C.stripMenuShortcuts() }
