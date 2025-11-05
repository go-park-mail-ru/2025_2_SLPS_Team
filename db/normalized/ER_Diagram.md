erDiagram
    PROFILE {
        int id PK
        text full_name
        text email
        text avatar
        text about_myself
        text password_hashed_with_salt
        timestamp created_at
        timestamp updated_at
    }

    FRIEND_RELATIONSHIP {
        int first_profile_id FK
        int second_profile_id FK
        friendship_status_enum status
        timestamp created_at
        timestamp updated_at
    }

    COMMUNITY {
        int id PK
        text name
        community_status_enum status
        text avatar
        text description
        timestamp created_at
        timestamp updated_at
    }

    COMMUNITY_AUTHOR {
        int community_id FK
        int author_id FK
        role_enum role
        timestamp created_at
        timestamp updated_at
    }

    COMMUNITY_SUBSCRIBER {
        int community_id FK
        int subscriber_id FK
        timestamp created_at
    }

    POST {
        int id PK
        int community_id FK
        int author_id FK
        text text
        timestamp created_at
        timestamp updated_at
    }

    COMMENT {
        int id PK
        int author_id FK
        int obj_id
        comment_obj_type_enum obj_type
        text text
        timestamp created_at
        timestamp updated_at
    }

    CHAT {
        int id PK
        text name
        text avatar
        timestamp created_at
        timestamp updated_at
    }

    CHAT_MEMBER {
        int chat_id FK
        int member_id FK
        role_enum role
        timestamp created_at
        timestamp updated_at
    }

    MESSAGE {
        int id PK
        int author_id FK
        int chat_id FK
        int replayed_message_id FK
        text text
        timestamp created_at
        timestamp updated_at
    }

    FORWARD_MESSAGE {
        int main_message_id FK
        int minor_message_id FK
        timestamp created_at
        timestamp updated_at
    }

    ATTACHMENT {
        int id PK
        int obj_id
        attachment_obj_type_enum obj_type
        text file_path
        timestamp created_at
        timestamp updated_at
    }

    REACTION {
        int author_id FK
        int obj_id
        reaction_obj_type_enum obj_type
        timestamp created_at
        timestamp updated_at
    }

    PROFILE ||--o{ FRIEND_RELATIONSHIP : "first_profile"
    PROFILE ||--o{ FRIEND_RELATIONSHIP : "second_profile"
    PROFILE ||--o{ COMMUNITY_AUTHOR : "author"
    PROFILE ||--o{ COMMUNITY_SUBSCRIBER : "subscriber"
    PROFILE ||--o{ POST : "author"
    PROFILE ||--o{ COMMENT : "author"
    PROFILE ||--o{ CHAT_MEMBER : "member"
    PROFILE ||--o{ MESSAGE : "author"
    PROFILE ||--o{ REACTION : "author"
    
    COMMUNITY ||--o{ COMMUNITY_AUTHOR : "authors"
    COMMUNITY ||--o{ COMMUNITY_SUBSCRIBER : "subscribers"
    COMMUNITY ||--o{ POST : "posts"
    
    CHAT ||--o{ CHAT_MEMBER : "members"
    CHAT ||--o{ MESSAGE : "messages"
    
    MESSAGE ||--o{ MESSAGE : "replies"
    MESSAGE ||--o{ FORWARD_MESSAGE : "main_forward"
    MESSAGE ||--o{ FORWARD_MESSAGE : "minor_forward"

    MESSAGE ||--o{ ATTACHMENT : "has"
    POST ||--o{ ATTACHMENT : "has"
    COMMENT ||--o{ ATTACHMENT : "has"
    COMMUNITY ||--o{ ATTACHMENT : "has" 