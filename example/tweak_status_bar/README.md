# Tweak: Status Bar Pulse

A system tweak that runs a heartbeat background process within the SpringBoard.

## What it implements
- Continuous background execution within a host process.
- Real-time logging from inside SpringBoard.
- Template for hooking into UIKit classes (StatusBar).

## How to Apply
1. Navigate to this directory:
   ```bash
   cd example/tweak_status_bar
   ```
2. Build and apply:
   ```bash
   gios run --watch
   ```
   *The `--watch` flag is recommended to see the logs in real-time as you modify the code.*

## Technical Details
This tweak uses a C constructor to bridge into the Go runtime. It serves as a base for developers who want to modify the UI elements of iOS using Go.
