#!/bin/bash
# ParaView container entrypoint (#290). DCV/X11 are on the host; this runs in the
# container against the bind-mounted X socket (DISPLAY=:0). Starts a minimal WM,
# launches ParaView maximized, and re-maximizes once DCV resizes the virtual
# display to the browser viewport (ported from the old per-AMI start-paraview-dcv
# wrapper). Blocks until ParaView exits, which ends the DCV session.
set -u

# Suppress the welcome dialog (it steals focus on startup).
mkdir -p "$HOME/.config/ParaView"
printf '[GeneralSettings]\nShowWelcomeDialog=0\n' > "$HOME/.config/ParaView/ParaView.ini"

xsetroot -solid black 2>/dev/null || true
metacity --sm-disable &
sleep 1

/usr/local/bin/paraview --maximize &
PV_PID=$!

# After DCV resizes the display to the browser viewport, re-maximize the main
# ParaView window (transient dialogs are skipped).
INIT_SIZE=$(xdpyinfo 2>/dev/null | grep dimensions | grep -oP '[0-9]+x[0-9]+')
for _ in $(seq 1 60); do
  sleep 2
  SIZE=$(xdpyinfo 2>/dev/null | grep dimensions | grep -oP '[0-9]+x[0-9]+')
  if [ "$SIZE" != "$INIT_SIZE" ] && [ -n "$SIZE" ]; then
    sleep 1
    for w in $(xprop -root 2>/dev/null | grep -oP '0x[0-9a-f]{5,}'); do
      xprop -id "$w" WM_CLASS 2>/dev/null | grep -qi paraview || continue
      xprop -id "$w" WM_TRANSIENT_FOR 2>/dev/null | grep -q 'window id' && continue
      xprop -id "$w" -format _NET_WM_STATE 32a -set _NET_WM_STATE \
        '_NET_WM_STATE_MAXIMIZED_VERT,_NET_WM_STATE_MAXIMIZED_HORZ' 2>/dev/null
    done
    break
  fi
done &

wait $PV_PID
