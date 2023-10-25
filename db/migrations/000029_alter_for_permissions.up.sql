ALTER TABLE community_roles
    ADD COLUMN view_channels BOOLEAN DEFAULT 0,
    ADD COLUMN manage_channels BOOLEAN DEFAULT 0,
    ADD COLUMN manage_community BOOLEAN DEFAULT 0,
    ADD COLUMN create_invite BOOLEAN DEFAULT 0,
    ADD COLUMN kick_members BOOLEAN DEFAULT 0,
    ADD COLUMN ban_members BOOLEAN DEFAULT 0,
    ADD COLUMN send_messages BOOLEAN DEFAULT 0,
    ADD COLUMN attach_media BOOLEAN DEFAULT 0;

ALTER TABLE communities_users
    ADD COLUMN view_channels BOOLEAN DEFAULT 0,
    ADD COLUMN manage_channels BOOLEAN DEFAULT 0,
    ADD COLUMN manage_community BOOLEAN DEFAULT 0,
    ADD COLUMN create_invite BOOLEAN DEFAULT 0,
    ADD COLUMN kick_members BOOLEAN DEFAULT 0,
    ADD COLUMN ban_members BOOLEAN DEFAULT 0,
    ADD COLUMN send_messages BOOLEAN DEFAULT 0,
    ADD COLUMN attach_media BOOLEAN DEFAULT 0;
