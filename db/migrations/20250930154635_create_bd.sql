-- Создание ENUM типов
CREATE TYPE friendship_status_enum AS ENUM ('pending', 'accepted', 'rejected', 'blocked');
CREATE TYPE community_status_enum AS ENUM ('public', 'private', 'closed');
CREATE TYPE role_enum AS ENUM ('admin', 'moderator', 'member', 'owner');
CREATE TYPE comment_obj_type_enum AS ENUM ('post', 'message', 'comment');
CREATE TYPE attachment_obj_type_enum AS ENUM ('post', 'message', 'comment', 'profile', 'community');
CREATE TYPE reaction_obj_type_enum AS ENUM ('post', 'message', 'comment');
CREATE TYPE reaction_type_enum AS ENUM ('like', 'dislike', 'love', 'laugh', 'angry', 'wow', 'sad');

-- Таблица пользователей
CREATE TABLE PROFILE
(
    id                          INT         PRIMARY KEY,
    full_name                   TEXT        NOT NULL,
                                CONSTRAINT full_name_length CHECK (LENGTH(full_name) BETWEEN 1 AND 64),
    email                       TEXT        UNIQUE,
                                CONSTRAINT email_length CHECK (LENGTH(email) <= 254),
    phone                       TEXT        UNIQUE,
                                CONSTRAINT phone_length CHECK (LENGTH(phone) <= 20),
    avatar                      TEXT        NULL,
                                CONSTRAINT avatar_length CHECK (LENGTH(avatar) <= 512),
    about_myself                TEXT,
                                CONSTRAINT about_myself_length CHECK (LENGTH(about_myself) <= 256),                                                        
    password_hashed_with_salt   TEXT        NOT NULL,
                                CONSTRAINT password_length CHECK (LENGTH(password_hashed_with_salt) >= 8),
    created_at                  DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT email_or_phone_required CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

-- Дружба между пользователями
CREATE TABLE FRIEND_RELATIONSHIP
(
    first_profile_id            INT         NOT NULL,
    second_profile_id           INT         NOT NULL,
    status friendship_status_enum NOT NULL DEFAULT 'pending',
                                CONSTRAINT status_length CHECK ( LENGTH(status::text) <= 64),
    created_at                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (first_profile_id, second_profile_id),
    FOREIGN KEY (first_profile_id) REFERENCES PROFILE (id) ON DELETE CASCADE,
    FOREIGN KEY (second_profile_id) REFERENCES PROFILE (id) ON DELETE CASCADE,
    CONSTRAINT no_self_friendship CHECK (first_profile_id != second_profile_id),
    CONSTRAINT ordered_friendship CHECK (first_profile_id < second_profile_id)
);

-- Сообщества
CREATE TABLE COMMUNITY
(
    id                          INT             PRIMARY KEY,
    name                        TEXT            NOT NULL,
                                CONSTRAINT name_length CHECK (LENGTH(name) BETWEEN 5 AND 64),
    status community_status_enum                NOT NULL DEFAULT 'public',
    avatar                      TEXT,
                                CONSTRAINT avatar_length CHECK (LENGTH(avatar) <= 512),
    description                 TEXT,
                                CONSTRAINT description_length CHECK (LENGTH(description) <= 512),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Авторы/админы сообщества
CREATE TABLE COMMUNITY_AUTHOR
(
    community_id                INT             NOT NULL,
    author_id                   INT             NOT NULL,
    role         role_enum                      NOT NULL DEFAULT 'member',
    created_at                                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (community_id, author_id),
    FOREIGN KEY (community_id) REFERENCES COMMUNITY (id) ON DELETE CASCADE,
    FOREIGN KEY (author_id) REFERENCES PROFILE (id) ON DELETE CASCADE
);

-- Подписчики сообщества
CREATE TABLE COMMUNITY_SUBSCRIBER
(
    community_id                INT             NOT NULL,
    subscriber_id               INT             NOT NULL,
    created_at                                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (community_id, subscriber_id),
    FOREIGN KEY (community_id) REFERENCES COMMUNITY (id) ON DELETE CASCADE,
    FOREIGN KEY (subscriber_id) REFERENCES PROFILE (id) ON DELETE CASCADE
);

-- Посты в сообществах
CREATE TABLE POST
(
    id                          INT             PRIMARY KEY,
    community_id                INT             NOT NULL,
    author_id                   INT             NOT NULL,
    text                        TEXT            NOT NULL,
                                CONSTRAINT text_length CHECK (LENGTH(text) BETWEEN 24 AND 4096),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (community_id) REFERENCES COMMUNITY (id) ON DELETE CASCADE,
    FOREIGN KEY (author_id) REFERENCES PROFILE (id) ON DELETE CASCADE
);

-- Комментарии (вложенные поддерживаются)
CREATE TABLE COMMENT
(
    id                          INT             PRIMARY KEY,
    author_id                   INT             NOT NULL,
    post_id                     INT             NOT NULL,
    obj_id                      INT             NOT NULL,
    obj_type comment_obj_type_enum              NOT NULL,
    text                        TEXT            NOT NULL,
                                CONSTRAINT text_length CHECK (LENGTH(text) BETWEEN 1 AND 1024),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES PROFILE (id),
    FOREIGN KEY (post_id) REFERENCES POST (id) ON DELETE CASCADE
);

-- Чаты
CREATE TABLE CHAT
(
    id                          INT             PRIMARY KEY,
    name                        TEXT            NOT NULL,
                                CONSTRAINT name_length CHECK (LENGTH(name) BETWEEN 5 AND 64),
    avatar                      TEXT,
                                CONSTRAINT avatar_length CHECK (LENGTH(avatar) <= 512),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Участники чатов
CREATE TABLE CHAT_MEMBER
(
    chat_id                     INT             NOT NULL,
    member_id                   INT             NOT NULL,
    role role_enum                              NOT NULL DEFAULT 'member',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chat_id, member_id),
    FOREIGN KEY (chat_id) REFERENCES CHAT (id) ON DELETE CASCADE,
    FOREIGN KEY (member_id) REFERENCES PROFILE (id) ON DELETE CASCADE
);

-- Сообщения в чатах
CREATE TABLE MESSAGE
(
    id                          INT             PRIMARY KEY,
    author_id                   INT             NOT NULL,
    chat_id                     INT             NOT NULL,
    replayed_message_id         INT,
    text                        TEXT            NOT NULL,
                                CONSTRAINT text_length CHECK (LENGTH(text) BETWEEN 1 AND 4096),
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES PROFILE (id),
    FOREIGN KEY (chat_id) REFERENCES CHAT (id) ON DELETE CASCADE,
    FOREIGN KEY (replayed_message_id) REFERENCES MESSAGE (id) ON DELETE SET NULL
);


-- Пересланные сообщения
CREATE TABLE FORWARD_MESSAGE
(
    main_message_id             INT             NOT NULL,
    minor_message_id            INT             NOT NULL,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (main_message_id, minor_message_id),
    FOREIGN KEY (main_message_id) REFERENCES MESSAGE (id) ON DELETE CASCADE,
    FOREIGN KEY (minor_message_id) REFERENCES MESSAGE (id) ON DELETE CASCADE,
    CONSTRAINT no_self_forward CHECK (main_message_id != minor_message_id)
);

-- Вложения
CREATE TABLE ATTACHMENT
(
    id                          INT             PRIMARY KEY,
    obj_id                      INT             NOT NULL,
    obj_type   attachment_obj_type_enum         NOT NULL,
    file_path                   TEXT            NOT NULL,
                                CONSTRAINT file_path_length CHECK (LENGTH(file_path) BETWEEN 24 AND 512),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Реакции (лайки и пр.)
CREATE TABLE REACTION
(
    author_id                   INT             NOT NULL,
    obj_id                      INT             NOT NULL,
    obj_type   reaction_obj_type_enum NOT NULL,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (author_id, obj_id, obj_type),
    FOREIGN KEY (author_id) REFERENCES PROFILE (id) ON DELETE CASCADE
);

-- Триггеры для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Создание триггеров для всех таблиц
CREATE TRIGGER update_profile_updated_at BEFORE UPDATE ON PROFILE FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_friend_relationship_updated_at BEFORE UPDATE ON FRIEND_RELATIONSHIP FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_community_updated_at BEFORE UPDATE ON COMMUNITY FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_community_author_updated_at BEFORE UPDATE ON COMMUNITY_AUTHOR FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_community_subscriber_updated_at BEFORE UPDATE ON COMMUNITY_SUBSCRIBER FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_post_updated_at BEFORE UPDATE ON POST FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_comment_updated_at BEFORE UPDATE ON COMMENT FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_chat_updated_at BEFORE UPDATE ON CHAT FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_chat_member_updated_at BEFORE UPDATE ON CHAT_MEMBER FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_message_updated_at BEFORE UPDATE ON MESSAGE FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_forward_message_updated_at BEFORE UPDATE ON FORWARD_MESSAGE FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_attachment_updated_at BEFORE UPDATE ON ATTACHMENT FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_reaction_updated_at BEFORE UPDATE ON REACTION FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();