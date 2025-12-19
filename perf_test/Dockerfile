# Используем официальный образ PostgreSQL
FROM postgres:15

# Копируем кастомные конфигурационные файлы в контейнер
COPY ./db/config/postgresql.conf /etc/postgresql/postgresql.conf
COPY ./db/config/pg_hba.conf /etc/postgresql/pg_hba.conf




