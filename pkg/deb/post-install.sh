#!/bin/bash


BIN_DIR=/usr/bin
LOG_DIR=/var/log/atella
SCRIPT_DIR=/usr/lib/atella/scripts
LOGROTATE_DIR=/etc/logrotate.d

function install_init {
    cp -f $SCRIPT_DIR/init.sh /etc/init.d/atella
    chmod +x /etc/init.d/atella
}

function install_systemd {
    cp -f $SCRIPT_DIR/atella.service $1
    systemctl daemon-reload || true
    systemctl enable atella || true
}

function install_update_rcd {
    update-rc.d atella defaults
}

function install_chkconfig {
    chkconfig --add atella
}

if [[ ! -f /etc/default/atella ]]; then
    touch /etc/default/atella
fi

if [[ ! -d /etc/atella/conf.d ]]; then
    mkdir -p /etc/atella/conf.d
fi

if [[ ! -f /etc/atella/atella.conf ]] && [[ -f /etc/atella/atella.conf.tpl ]]; then
   cp /etc/atella/atella.conf.tpl /etc/atella/atella.conf
fi

if [[ ! -f $LOGROTATE_DIR/atella ]] ; then
   cp $SCRIPT_DIR/atella.logrotate $LOGROTATE_DIR/atella
fi

test -d $LOG_DIR || mkdir -p $LOG_DIR
chown -R -L atella:atella $LOG_DIR
chmod 755 $LOG_DIR

if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
	install_systemd /lib/systemd/system/atella.service
	systemctl restart atella.service || echo "WARN: systemd not running."
else
	install_init

	if which update-rc.d &>/dev/null; then
		install_update_rcd
	else
		install_chkconfig
	fi
	invoke-rc.d atella restart
fi


# -- OLD CONFIGURATION --

# if [ -x /bin/systemctl ] ; then
#   /bin/systemctl daemon-reload
#   /bin/systemctl enable atella.service
#   if ! /bin/systemctl status atella.service > /dev/null 2>&1 ; then
#       /bin/systemctl restart atella.service
#   else
#       /bin/systemctl start atella.service
#   fi
# elif [ -x /sbin/chkconfig ] ; then
#   /sbin/chkconfig --add atella
# fi