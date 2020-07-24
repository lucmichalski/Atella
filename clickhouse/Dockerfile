FROM ubuntu:bionic
LABEL Author="Alexey Romanuta R9ODT"

ARG IMAGE_VERSION
ENV CLICKHOUSE_HOME=/var/lib/clickhouse \ 
    CLICKHOUSE_CONFIG=/etc/clickhouse-server/config.xml \
    IMAGE_USER=clickhouse \
    GOSU_VERSION=1.12

# ENTRYPOINT ["/bin/bash"]

RUN groupadd -g 1000 ${IMAGE_USER} && \
    useradd -m -d ${CLICKHOUSE_HOME} -s /usr/sbin/nologin -g 1000 -u 1000 ${IMAGE_USER}
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
    gnupg \
    dirmngr \
    ca-certificates \
    wget \
    procps ; \
    rm -rf /var/lib/apt/lists/* /var/cache/apt
RUN dpkgArch="$(dpkg --print-architecture | awk -F- '{ print $NF }')"; \
	wget --no-verbose -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$dpkgArch"; \
	chmod +x /usr/local/bin/gosu; \
	gosu --version; \
	gosu nobody true
RUN echo "deb http://repo.yandex.ru/clickhouse/deb/stable/ main/" >> \
    /etc/apt/sources.list && \
    apt-key adv --keyserver keyserver.ubuntu.com --recv E0C56BD4
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    clickhouse-client=${IMAGE_VERSION} \
    clickhouse-server=${IMAGE_VERSION} ; \
    rm -rf /var/lib/apt/lists/* /var/cache/apt

EXPOSE 9000 8123 9009

WORKDIR ${CLICKHOUSE_HOME}

COPY ./docker-entrypoint.sh /bin/docker-entrypoint.sh
ENTRYPOINT ["/bin/docker-entrypoint.sh"]

COPY ./start.sh /bin/start.sh
CMD ["/bin/start.sh"]
