//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// Wails builds the edit and window menus natively and binds Select All to Cmd+A and
// Minimize to Cmd+M. The app uses those keys itself, so take them off the items.
static void stripMenuShortcuts(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		for (NSMenuItem *top in [[NSApp mainMenu] itemArray]) {
			for (NSMenuItem *item in [[top submenu] itemArray]) {
				NSString *title = [item title];
				if ([title isEqualToString:@"Select All"] || [title isEqualToString:@"Minimize"]) {
					[item setKeyEquivalent:@""];
				}
			}
		}
	});
}
*/
import "C"

func stripMenuShortcuts() { C.stripMenuShortcuts() }
