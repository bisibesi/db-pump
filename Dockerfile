FROM php:8.2-apache

USER root

# 1. 한국 미러 서버로 변경 (속도 및 연결 안정성 확보)
RUN sed -i 's/deb.debian.org/ftp.kr.debian.org/g' /etc/apt/sources.list.d/debian.sources || true \
    && sed -i 's/deb.debian.org/ftp.kr.debian.org/g' /etc/apt/sources.list || true

# 2. APT 재시도 및 시스템 의존성 설치
RUN apt-get clean \
    && apt-get update --fix-missing \
    && apt-get install -y --no-install-recommends \
    libaio1t64 wget unzip build-essential \
    && ln -s /usr/lib/x86_64-linux-gnu/libaio.so.1t64 /usr/lib/x86_64-linux-gnu/libaio.so.1 \
    && rm -rf /var/lib/apt/lists/*

# 3. Oracle Instant Client 설치
RUN mkdir /opt/oracle && cd /opt/oracle \
    && wget -q https://download.oracle.com/otn_software/linux/instantclient/instantclient-basiclite-linuxx64.zip \
    && wget -q https://download.oracle.com/otn_software/linux/instantclient/instantclient-sdk-linuxx64.zip \
    && unzip -o instantclient-basiclite-linuxx64.zip || echo "Unzip basiclite returned $?" \
    && unzip -o instantclient-sdk-linuxx64.zip || echo "Unzip sdk returned $?" \
    && rm -f *.zip \
    && mv instantclient_* instantclient \
    && echo /opt/oracle/instantclient > /etc/ld.so.conf.d/oracle-instantclient.conf \
    && ldconfig

# 4. PHP 확장 빌드 (OCI8, PDO_OCI)
RUN echo "instantclient,/opt/oracle/instantclient" | pecl install oci8 \
    && docker-php-ext-enable oci8 \
    && docker-php-ext-configure pdo_oci --with-pdo-oci=instantclient,/opt/oracle/instantclient \
    && docker-php-ext-install pdo_oci

# 5. Adminer 설치
RUN wget -q https://www.adminer.org/latest.php -O /var/www/html/index.php \
    && chown -R www-data:www-data /var/www/html