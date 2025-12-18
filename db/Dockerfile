FROM postgres:16

# Копируем шаблоны и entrypoint внутрь образа
COPY config/postgresql.conf /etc/postgresql/postgresql.conf.template
COPY ./init/init.sql.template /docker-entrypoint-initdb.d/init.sql.template
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["postgres"]
