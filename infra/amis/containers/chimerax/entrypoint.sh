#!/bin/bash
# UCSF ChimeraX container entrypoint (#290). DCV/X11 are on the host; this runs in
# the container against the bind-mounted X socket with the session's DISPLAY
# (passed in by spore-app-run, #263). Starts a minimal WM, launches ChimeraX
# fullscreen, and re-maximizes once DCV resizes the virtual display to the browser
# viewport. Blocks until ChimeraX exits, which ends the DCV session.
set -u

xsetroot -solid black 2>/dev/null || true
metacity --sm-disable &
sleep 1

# ChimeraX has no --fullscreen flag; the WM (metacity) + the re-maximize loop
# below own fullscreen. Start it plain and capture the PID.
/usr/bin/chimerax &
CX_PID=$!

# After DCV resizes the display to the browser viewport, re-maximize the main
# ChimeraX window (skip transient dialogs).
INIT_SIZE=$(xdpyinfo 2>/dev/null | grep dimensions | grep -oP '[0-9]+x[0-9]+')
for _ in $(seq 1 60); do
  sleep 2
  SIZE=$(xdpyinfo 2>/dev/null | grep dimensions | grep -oP '[0-9]+x[0-9]+')
  if [ "$SIZE" != "$INIT_SIZE" ] && [ -n "$SIZE" ]; then
    sleep 1
    for w in $(xprop -root 2>/dev/null | grep -oP '0x[0-9a-f]{5,}'); do
      xprop -id "$w" WM_CLASS 2>/dev/null | grep -qi chimerax || continue
      xprop -id "$w" WM_TRANSIENT_FOR 2>/dev/null | grep -q 'window id' && continue
      xprop -id "$w" -format _NET_WM_STATE 32a -set _NET_WM_STATE \
        '_NET_WM_STATE_MAXIMIZED_VERT,_NET_WM_STATE_MAXIMIZED_HORZ' 2>/dev/null
    done
    break
  fi
done &

wait $CX_PID
