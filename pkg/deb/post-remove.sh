#!/bin/bash

function disable_systemd {
    systemctl disable atella
    rm -f $1
}

function disable_update_rcd {
    update-rc.d -f atella remove
    rm -f /etc/init.d/atella
}

function disable_chkconfig {
    chkconfig --del atella
    rm -f /etc/init.d/atella
}

if [ "$1" == "remove" -o "$1" == "purge" ]; then
	# Remove/purge
	rm -f /etc/default/atella

	if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
		disable_systemd /lib/systemd/system/atella.service
	else
		if which update-rc.d &>/dev/null; then
			disable_update_rcd
		else
			disable_chkconfig
		fi
	fi
fi

# -- OLD CONFIGURATION --

# set -e

# if [ -x /bin/systemctl ] ; then
#   /bin/systemctl daemon-reload
# fi
