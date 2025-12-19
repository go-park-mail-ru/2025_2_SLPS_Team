erDiagram
    PROFILE {
        int32 id PK
        text full_name
        text email
        text avatar
        text about_myself
        text password_hashed_with_salt
        timestamp created_at
        timestamp updated_at
    }

    FRIEND_RELATIONSHIP {
        int32 first_profile_id FK
        int32 second_profile_id FK
        friendship_status_enum status
        timestamp created_at
        timestamp updated_at
    }

    COMMUNITY {
        int32 id PK
        text name
        community_status_enum status
        text avatar
        text description
        timestamp created_at
        timestamp updated_at
    }

    COMMUNITY_AUTHOR {
        int32 community_id FK
        int32 author_id FK
        role_enum role
        timestamp created_at
        timestamp updated_at
    }

    COMMUNITY_SUBSCRIBER {
        int32 community_id FK
        int32 subscriber_id FK
        timestamp created_at
    }

    POST {
        int32 id PK
        int32 community_id FK
        int32 author_id FK
        text text
        timestamp created_at
        timestamp updated_at
    }

    COMMENT {
        int32 id PK
        int32 author_id FK
        int32 obj_id
        comment_obj_type_enum obj_type
        text text
        timestamp created_at
        timestamp updated_at
    }

    CHAT {
        int32 id PK
        text name
        text avatar
        timestamp created_at
        timestamp updated_at
    }

    CHAT_MEMBER {
        int32 chat_id FK
        int32 member_id FK
        role_enum role
        timestamp created_at
        timestamp updated_at
    }

    MESSAGE {
        int32 id PK
        int32 author_id FK
        int32 chat_id FK
        int32 replayed_message_id FK
        text text
        timestamp created_at
        timestamp updated_at
    }

    FORWARD_MESSAGE {
        int32 main_message_id FK
        int32 minor_message_id FK
        timestamp created_at
        timestamp updated_at
    }

    ATTACHMENT {
        int32 id PK
        int32 obj_id
        attachment_obj_type_enum obj_type
        text file_path
        timestamp created_at
        timestamp updated_at
    }

    REACTION {
        int32 author_id FK
        int32 obj_id
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
