CREATE TABLE community_roles (
    id BIGINT unsigned NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME,
    object_salt VARCHAR(255) NOT NULL,
    community_id BIGINT unsigned NOT NULL,
    name VARCHAR(255) NOT NULL,
    show_online_differently BOOLEAN NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX community_roles_community_id_idx ON community_roles (community_id);
CREATE INDEX community_roles_show_online_differently_idx ON community_roles (show_online_differently);

CREATE TABLE community_roles_users (
    created_at DATETIME NOT NULL,
    community_role_id BIGINT unsigned NOT NULL,
    user_id BIGINT unsigned NOT NULL
);

CREATE INDEX community_roles_users_community_role_id_idx ON community_roles_users (community_role_id);
CREATE INDEX community_roles_users_user_id_idx ON community_roles_users (user_id);
