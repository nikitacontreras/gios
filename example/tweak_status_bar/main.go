package main

/*
extern void TweakEntry();

__attribute__((constructor)) static void init_tweak() {
    TweakEntry();
}
*/
import "C"

import (
	"fmt"
	"time"
)

//export TweakEntry
func TweakEntry() {
	go func() {
		// Just a simple log to demonstrate it's in the process context
		for {
			fmt.Printf("[Gios/StatusBar] Tweak active! Heartbeat from process...\n")
			time.Sleep(30 * time.Second)
		}
	}()
}

func init() {}

func main() {}
