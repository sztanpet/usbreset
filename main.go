// Reset USB device, point is to not have to physically re-plug the device
// Reimplementation of code from https://askubuntu.com/questions/645/how-do-you-reset-a-usb-device-from-the-command-line
// Use lsusb to identify device on the bus
// Example: ./usbreset -p /dev/bus/usb/003/004
package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

var pathToReset = flag.String("pathToReset", "", "The device-filename under /dev/bus/usb/ that shall be reset")

func init() {
	flag.StringVar(pathToReset, "p", "", "(shorthand for --pathToReset)")
}

func main() {
	flag.Parse()

	if *pathToReset == "" {
		flag.PrintDefaults()
		fail("No path to reset specified!", nil)
	}

	h, err := os.OpenFile(*pathToReset, os.O_WRONLY, 666)
	if err != nil {
		fail("Could not open file", err)
	}

	const USBDEVFS_RESET = 'U'<<(4*2) | 20
	if err = unix.IoctlSetInt(int(h.Fd()), USBDEVFS_RESET, 0); err != nil {
		fail("Could not reset", err)
	}

	_ = h.Close()
	os.Exit(0)
}

func fail(msg string, err error) {
	fmt.Print(msg)
	if err != nil {
		fmt.Println(": " + err.Error())
	} else {
		fmt.Println()
	}

	os.Exit(1)
}
