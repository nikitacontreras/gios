# System Notification Proxy (Darwin)

A daemon that listens for global system events (Darwin Notifications) and logs them for analysis.

## What it implements
- **Darwin Notifications**: Intercepts events like screen locking, battery changes, and network shifts.
- **CGO Integration**: Wraps `<notify.h>` to register system-wide callbacks.
- **Async Events**: Bridges C callbacks back into Go channels for clean handling.

## How to use
1. Navigate to this directory:
   ```bash
   cd example/sys_notify_proxy
   ```
2. Build and install:
   ```bash
   gios install
   ```
3. Monitor the events:
   ```bash
   gios logs
   ```
4. Try locking/unlocking your iPad to see the notifications being captured in the log.

## Importance
This is a foundational example for security research and system automation. It allows your Go code to react to any native system event in real-time.
