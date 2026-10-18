-- +goose Up
-- The avatar shown is the first one set of: profile_image (an upload's R2
-- key), avatar_preset (one of the built-in avatars), oauth_image_url (the
-- Google/X picture from the last social login), then the default preset.
ALTER TABLE users_profile
    ADD COLUMN avatar_preset   text,
    ADD COLUMN oauth_image_url text,
    ADD COLUMN city            text,
    ADD CONSTRAINT chk_users_profile_city_len CHECK (char_length(city) <= 100),
    ADD CONSTRAINT chk_users_profile_oauth_image_url CHECK (char_length(oauth_image_url) <= 1024 AND oauth_image_url LIKE 'https://%');

-- bio and country_code already hold data from before these rules. NOT VALID
-- checks new writes without failing the migration on an old row.
ALTER TABLE users_profile
    ADD CONSTRAINT chk_users_profile_bio_len CHECK (char_length(bio) <= 300) NOT VALID,
    ADD CONSTRAINT chk_users_profile_country_code CHECK (country_code ~ '^[A-Z]{2}$') NOT VALID;

-- +goose Down
ALTER TABLE users_profile
    DROP CONSTRAINT IF EXISTS chk_users_profile_country_code,
    DROP CONSTRAINT IF EXISTS chk_users_profile_oauth_image_url,
    DROP CONSTRAINT IF EXISTS chk_users_profile_bio_len,
    DROP CONSTRAINT IF EXISTS chk_users_profile_city_len,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS oauth_image_url,
    DROP COLUMN IF EXISTS avatar_preset;
