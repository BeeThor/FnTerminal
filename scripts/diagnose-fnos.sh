#!/bin/bash
# Read-only diagnostics: no environment dump, credentials, or terminal transcript.
set -uo pipefail
printf 'FnTerminal diagnostic — %s\n' "$(date -Iseconds)"
printf 'Architecture: %s\n' "$(uname -m)"
printf 'Runtime identity: '; id
printf 'Package account: '; id fnterminal 2>&1 || true
for file in /var/apps/fnterminal/target /var/apps/fnterminal/target/bin/fnterminal /var/apps/fnterminal/target/app.sock /var/apps/fnterminal/home /var/apps/fnterminal/var /var/apps/fnterminal/etc; do
 stat -Lc '%A %a %U:%G %n' "$file" 2>&1 || true
done
printf '\nApplication manifest:\n'
sed -n '1,30p' /var/apps/fnterminal/manifest 2>/dev/null || true
printf '\nService log (last 30 lines; audit metadata only):\n'
tail -n 30 /var/apps/fnterminal/var/fnterminal.log 2>/dev/null || true
printf '\nPackage processes (no arguments or environment):\n'
ps -u fnterminal -o pid,ppid,sid,stat,comm 2>/dev/null || true
