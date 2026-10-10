#!/bin/sh
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
mount -t proc proc /proc
mount -t sysfs sysfs /sys
ifconfig lo 127.0.0.1 up
mount -t devtmpfs devtmpfs /dev
mkdir -p /dev/pts /tmp /home/fnterminal
mount -t devpts devpts /dev/pts
chmod 1777 /tmp
printf 'fnterminal:x:700:700:FnTerminal:/home/fnterminal:/bin/bash\n' >> /etc/passwd
printf 'fnterminal:x:700:\n' >> /etc/group
chown 700:700 /home/fnterminal
mkdir -p /run/sshd /test-keys /home/nasdemo/.ssh /home/10001/.ssh
printf 'sshd:x:22:22:SSH:/var/empty:/sbin/nologin\n' >> /etc/passwd
printf 'sshd:x:22:\n' >> /etc/group
printf 'nasdemo:x:701:701:SSH test:/home/nasdemo:/bin/sh\n' >> /etc/passwd
printf 'nasdemo:x:701:\n' >> /etc/group
printf 'nasdemo::20000:0:99999:7:::\n' >> /etc/shadow
printf '10001:x:702:702:Numeric SSH test:/home/10001:/bin/sh\n' >> /etc/passwd
printf '10001:x:702:\n' >> /etc/group
printf '10001::20000:0:99999:7:::\n' >> /etc/shadow
ssh-keygen -A >/dev/null 2>&1
ssh-keygen -t ed25519 -N '' -f /test-keys/id_ed25519 >/dev/null 2>&1
cp /test-keys/id_ed25519.pub /home/nasdemo/.ssh/authorized_keys
chmod 700 /home/nasdemo/.ssh; chmod 600 /home/nasdemo/.ssh/authorized_keys
chown -R 701:701 /home/nasdemo
cp /test-keys/id_ed25519.pub /home/10001/.ssh/authorized_keys
chmod 700 /home/10001/.ssh; chmod 600 /home/10001/.ssh/authorized_keys
chown -R 702:702 /home/10001
/usr/sbin/sshd -p 22 -p 10022 -o PasswordAuthentication=no -o PermitEmptyPasswords=no -o "AllowUsers=nasdemo 10001"
printf '\nLINUX_TEST_START\n'
uname -a
su fnterminal -s /bin/sh -c '/fttests -test.v -test.timeout=60s'
result=$?
if [ "$result" = 0 ]; then FNTERMINAL_TEST_SSH=1 FNTERMINAL_PRODUCTION_BIN=/production-service /fttests -test.v -test.run=TestLinuxProductionGateway -test.timeout=30s; result=$?; fi
if [ "$result" = 0 ]; then /bin/sh /lifecycle.sh; result=$?; fi
printf '\nLINUX_TEST_RESULT=%s\n' "$result"
sync
poweroff -f
