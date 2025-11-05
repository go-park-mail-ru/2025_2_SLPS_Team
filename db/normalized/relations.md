PROFILE:
id PrimaryKey
full_name
email
avatar S3_Path
about_myself
password_hashed_with_salt
created_at
updated_at

PROFILE:
{id} -> {full_name, email, avatar, about_myself, password_hashed_with_salt, created_at, updated_at}
 
FRIEND_RELATIONSHIP:
first_profile_id
second_profile_id
status
created_at
updated_at

FRIEND_RELATIONSHIP:
{first_profile_id, second_profile_id} -> {status, created_at, updated_at}

COMMUNITY:
id PrimaryKey
name
status
avatar S3_Path
description
created_at
updated_at

COMMUNITY:
{id} -> {name, status, avatar, description, created_at, updated_at}

COMMUNITY_AUTHOR:
community_id
author_id
role
created_at
updated_at

COMMUNITY_AUTHOR:
{community_id, author_id} -> {role, created_at, updated_at}

COMMUNITY_SUBSCRIBER:
community_id
subscriber_id
created_at

COMMUNITY_SUBSCRIBER:
{community_id, subscriber_id} -> {created_at}

POST:
id primary_key
community_id
author_id
text
created_at
updated_at

POST:
{id} -> {community_id, author_id, text, created_at, updated_at}

COMMENT:
id PrimaryKey
author_id
obj_id
obj_type
text
created_at
updated_at

COMMENT:
{id} -> {author_id, obj_id, obj_type, text, created_at, updated_at}

CHAT:
id PrimaryKey
name
avatar S3_Path
created_at
updated_at

CHAT:
{id}-> {name, avatar, created_at, updated_at}

CHAT_MEMBER:
chat_id ForeignKey
member_id ForeignKey
role
created_at
updated_at

CHAT_MEMBER:
{chat_id, member_id} -> {role, created_at, updated_at}

MESSAGE:
id PrimaryKey
author_id ForeignKey
chat_id ForeignKey
replayed_message_id ForeignKey
text
created_at
updated_at

MESSAGE:
{id} -> {author_id, chat_id, replayed_message_id, text, created_at, updated_at}

FORWARD_MESSAGE:
main_message_id ForeignKey
minor_message_id ForeignKey
created_at
updated_at

FORWARD_MESSAGE:
{main_message_id, minor_message_id} -> {created_at, updated_at}

ATTACHMENT:
id PrimaryKey
obj_id ForeignKey
obj_type 
file_path S3_Path
created_at
updated_at

ATTACHMENT:
{id} -> {obj_id, obj_type, file_path, created_at, updated_at}

REACTION:
author_id ForeignKey
obj_id ForeignKey
obj_type
created_at
updated_at

REACTION:
{author_id, obj_id} -> {obj_type, created_at, updated_at}
