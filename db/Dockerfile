# Используем официальный образ PostgreSQL
FROM postgres:16

# Копируем кастомные конфиги внутрь образа
COPY ./db/config/postgresql.conf /etc/postgresql/postgresql.conf
COPY ./db/config/pg_hba.conf /etc/postgresql/pg_hba.conf

# Создаём директорию для логов и выставляем владельца postgres
RUN mkdir -p /var/lib/postgresql/log \
    && chown -R postgres:postgres /var/lib/postgresql/log \
    && chmod 700 /var/lib/postgresql/log

# По умолчанию запускаем PostgreSQL с кастомным конфигом
CMD ["postgres", "-c", "config_file=/etc/postgresql/postgresql.conf"]
