#!/bin/sh

CERT_DIR="/etc/nginx/certs/live/unitesm.ru"

# Следим за изменениями сертификатов
inotifywait -m -e close_write "$CERT_DIR" |
while read -r path action file; do
    if [ "$file" = "fullchain.pem" ] || [ "$file" = "privkey.pem" ]; then
        echo "Certificate $file changed, reloading Nginx..."
        nginx -s reload
    fi
done
