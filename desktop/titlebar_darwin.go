//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// Wails gives the hidden-inset title bar the default toolbar style, which centres the
// window buttons 26pt down. The compact style centres them at 20pt, the middle of the top bar.
static void compactTitleBar(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		for (NSWindow *window in [NSApp windows]) {
			if ([window toolbar] != nil) {
				[window setToolbarStyle:NSWindowToolbarStyleUnifiedCompact];
			}
		}
	});
}
*/
import "C"

func compactTitleBar() { C.compactTitleBar() }
