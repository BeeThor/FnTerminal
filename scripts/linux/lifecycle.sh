#!/bin/sh
set -eu
# Only used inside the disposable test VM.
base=/tmp/fnterminal-lifecycle
mkdir -p "$base/target/bin" "$base/target/web" "$base/home" "$base/var" "$base/etc"
cp /production-service "$base/target/bin/fnterminal"
printf '<html>test</html>\n' > "$base/target/web/index.html"
chown -R 700:700 "$base"
chmod 775 "$base/target"
export TRIM_APPDEST="$base/target" TRIM_PKGHOME="$base/home" TRIM_PKGVAR="$base/var" TRIM_PKGETC="$base/etc" TRIM_USERNAME=fnterminal TRIM_TEMP_LOGFILE="$base/error.log"
# Real root lifecycle invocation must drop to fnterminal and leave no root service.
/bin/bash /package-main start
/bin/bash /package-main status
read -r pid born < "$base/var/fnterminal.pid"
actual=$(awk '/^Uid:/{print $2}' "/proc/$pid/status")
[ "$actual" = 700 ] || { echo "Unexpected service UID $actual"; exit 1; }
/bin/bash /package-main stop
if /bin/bash /package-main status; then echo 'Service remained running'; exit 1; else [ "$?" = 3 ]; fi
[ ! -S "$base/target/app.sock" ]
echo 'LINUX_LIFECYCLE_PASS start/status/stop UID700 dedicated0775'
