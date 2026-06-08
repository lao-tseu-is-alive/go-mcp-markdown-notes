#!/bin/bash
## serverSystemdInstall.sh
## version : 1.0.0
## script to prepare systemd unit to install program name received as first argument
echo "NUM ARGS : " $#
if [ $# -eq 1 ]; then
  APP_NAME=${1,,}
else
  echo "## 💥💥 ERROR: Did not receive unit name as first argument, will exit..."
  exit 1
fi
cd /tmp || exit 1
echo "## Converting App name Upper Case to underscore"
#GONAME=$(echo "$APP_NAME" | sed --expression 's/\([A-Z]\)/_\L\1/g' --expression 's/^_//')
GONAME=$APP_NAME
echo "creating group $GONAME"
groupadd --system "$GONAME"
grep "$GONAME" /etc/group
echo "creating user $GONAME and adding it to group $GONAME"
useradd -M -r -s /sbin/nologin "$GONAME" -g "$GONAME"
id "$GONAME"
chown "$GONAME":"$GONAME" "/usr/local/bin/${GONAME}Server"
lsa /usr/local/bin/
cp "${GONAME}.service" /etc/systemd/system/
mkdir "/var/lib/${GONAME}"
chown -R "$GONAME":"$GONAME" "/var/lib/${GONAME}"
echo "you can edit and check with : vim /etc/systemd/system/${GONAME}.service"
mkdir "/var/log/${GONAME}"
chown -R "$GONAME":"$GONAME" "/var/log/${GONAME}"
chmod -R 775 "/var/log/${GONAME}"
chmod -R 775 "/var/lib/${GONAME}"
systemctl status "${GONAME}.service"
systemctl enable "${GONAME}.service"
systemctl start "${GONAME}.service"
systemctl status "${GONAME}.service"

