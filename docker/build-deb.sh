#!/bin/bash
fpm -t deb \
	-s "tar" \
	--description "${DESCRIPTION}" \
	--vendor "${VENDOR}" \
	--url "${URL}" \
	--license "${LICENSE}" \
	--name "${SERVICE}" \
	--version "${VERSION_RELEASE}" \
	--iteration "1" \
	--config-files "/etc/${SERVICE}/${SERVICE}.conf" \
	--after-install "/pkg/postinst" \
	-p /pkg/deb \
	/pkg/tar/${SERVICE}-${VERSION_RELEASE}.tar.gz