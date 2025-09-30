PROFILE:
id primary key
full_name
email
avatar S3_Path
about_myself
password
profiles:
id ->  full_name, email, avatar, password, about_myself

FRIEND_RELATIONSHIP:
first_profile_id
second_profile_id
created_at
status

FRIEND_RELATIONSHIP:
{first_profile_id, second_profile_id} -> created_at, status



COMMUNITY:
id primary key
name
status
description
avatar S3_Path
Community:
community_id -> name, status, avatar, description

COMMUNITY_AUTHOR:
community_id
author_id
role
community_author:
{community_id, author_id} -> role

COMMUNITY_SUBSCRIBER:
community_id
subscriber_id
created_at
community_subscriber:
{community_id, subscriber_id} -> created_at

POST:
id primary_key
text
community_id
author_id
post:
id -> text, community_id, author_id


COMMENT:
id primary key
text
author_id
obj_id ?? Какой референс ?
comment:
id -> text, author_id, obj_id

MESSAGE:
id primary key
author_id
chat_id
replayed_message_id
message:
id-> author_id, chat_id, replayed_message_id


FORWARD_MESSAGE:
main_message_id
minor_message_id
forward_message:
{main_message_id, minor_message_id} ->
CHAT:
id primary key
avatar S3_Path
description
chat:
id-> avatar, description
CHAT_MEMBER:
chat_id
member_id
role
chat_member:
{chat_id, member_id} -> role

ATTACHMENT:
id primary key
binary_file S3_Path
obj_id
attachment:
id -> binary_file, obj_id

REACTION:
author_id
obj_id
type
reaction:
{author_id, obj_id} -> type
