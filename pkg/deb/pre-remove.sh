#!/bin/bash

BIN_DIR=/usr/bin

if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
	deb-systemd-invoke stop atella.service
else
	invoke-rc.d atella stop
fi

# -- OLD CONFIGURATION --

# if [ -x /bin/systemctl ] ; then
#   /bin/systemctl stop atella.service
# fi