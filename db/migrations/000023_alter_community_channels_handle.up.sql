ALTER TABLE channels ADD COLUMN handle VARCHAR(255) NOT NULL;
ALTER TABLE `channels` ADD UNIQUE `channels_handle_uq`(`handle`, `community_id`);
