//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// After a relaunch the window can be key while the web view never heard it become key, and then
// the web view shows no hover and keeps a stale cursor until the user leaves the app and comes
// back. Telling the web view again what leaving and coming back tells it puts that right.
static void refreshKeyWindow(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *window = [NSApp keyWindow];
		if (window == nil) {
			return;
		}
		NSNotificationCenter *center = [NSNotificationCenter defaultCenter];
		[center postNotificationName:NSWindowDidResignKeyNotification object:window];
		[center postNotificationName:NSWindowDidBecomeKeyNotification object:window];
	});
}
*/
import "C"

func refreshKeyWindow() { C.refreshKeyWindow() }
