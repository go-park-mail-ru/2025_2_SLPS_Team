-- Таблица пользователей
CREATE TABLE PROFILE
(
    id                        INT PRIMARY KEY,
    username                  TEXT        NOT NULL,
    CONSTRAINT username_length CHECK (LENGTH(username) <= 64),
    email                     TEXT UNIQUE NOT NULL,
    CONSTRAINT email_length CHECK (LENGTH(email) <= 64),
    password_hashed_with_salt TEXT        NOT NULL,
    avatar_path               TEXT NULL,
    CONSTRAINT avatar_path_length CHECK ( LENGTH(avatar_path) <= 512),
    about_myself              TEXT,
    CONSTRAINT about_myself_length CHECK ( LENGTH(about_myself) <= 256),
    created_at                DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Дружба между пользователями
CREATE TABLE FRIEND_RELATIONSHIP
(
    first_friend_id  INT,
    second_friend_id INT,
    status           TEXT,
    CONSTRAINT status_length CHECK ( LENGTH(status) <= 64),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (first_friend_id, second_friend_id),
    FOREIGN KEY (first_friend_id) REFERENCES PROFILE (id),
    FOREIGN KEY (second_friend_id) REFERENCES PROFILE (id)
);

-- Сообщества
CREATE TABLE COMMUNITY
(
    id          INT PRIMARY KEY,
    name        TEXT,
    status      TEXT,
    description TEXT,
    avatar_path TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Авторы/админы сообщества
CREATE TABLE COMMUNITY_AUTHOR
(
    community_id INT,
    author_id    INT,
    role         TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (community_id, author_id),
    FOREIGN KEY (community_id) REFERENCES COMMUNITY (id),
    FOREIGN KEY (author_id) REFERENCES PROFILE (id)
);

-- Подписчики сообщества
CREATE TABLE COMMUNITY_SUBSCRIBER
(
    community_id  INT,
    subscriber_id INT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (community_id, subscriber_id),
    FOREIGN KEY (community_id) REFERENCES COMMUNITY (id),
    FOREIGN KEY (subscriber_id) REFERENCES PROFILE (id)
);

-- Посты в сообществах
CREATE TABLE POST
(
    id           INT PRIMARY KEY,
    community_id INT,
    author_id    INT,
    text         TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (community_id) REFERENCES COMMUNITY (id),
    FOREIGN KEY (author_id) REFERENCES PROFILE (id)
);

-- Комментарии (вложенные поддерживаются)
CREATE TABLE COMMENT
(
    id                INT PRIMARY KEY,
    author_id         INT,
    post_id           INT,
    parent_comment_id INT,
    text              TEXT,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES PROFILE (id),
    FOREIGN KEY (post_id) REFERENCES POST (id),
    FOREIGN KEY (parent_comment_id) REFERENCES COMMENT (id)
);

-- Чаты
CREATE TABLE CHAT
(
    id            INT PRIMARY KEY,
    avatar_path   TEXT,
    description   TEXT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Участники чатов
CREATE TABLE CHAT_MEMBER
(
    chat_id    INT,
    member_id  INT,
    role       TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chat_id, member_id),
    FOREIGN KEY (chat_id) REFERENCES CHAT (id),
    FOREIGN KEY (member_id) REFERENCES PROFILE (id)
);

-- Сообщения в чатах
CREATE TABLE MESSAGE
(
    id                 INT PRIMARY KEY,
    author_id          INT,
    chat_id            INT,
    replied_message_id INT,
    text               TEXT,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES PROFILE (id),
    FOREIGN KEY (chat_id) REFERENCES CHAT (id),
    FOREIGN KEY (replied_message_id) REFERENCES MESSAGE (id)
);

-- Пересланные сообщения
CREATE TABLE FORWARD_MESSAGE
(
    main_message_id  INT,
    minor_message_id INT,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (main_message_id, minor_message_id),
    FOREIGN KEY (main_message_id) REFERENCES MESSAGE (id),
    FOREIGN KEY (minor_message_id) REFERENCES MESSAGE (id)
);

CREATE TYPE obj_type_enum AS ENUM ('POST', 'COMMENT', 'MESSAGE');

-- Вложения
CREATE TABLE ATTACHMENT
(
    id         INT PRIMARY KEY,
    file_path  TEXT,
    file_type  TEXT,
    obj_id     INT,
    obj_type   obj_type_enum,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
);

CREATE TYPE reaction_type AS ENUM ('LIKE', 'DISLIKE', 'LOVE', 'LAUGH', 'ANGRY');

-- Реакции (лайки и пр.)
CREATE TABLE REACTION
(
    id         INT PRIMARY KEY,
    author_id  INT,
    obj_id     INT,
    obj_type   obj_type_enum,
    type       reaction_type NOT NULL,
    created_at DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES PROFILE (id)
);
