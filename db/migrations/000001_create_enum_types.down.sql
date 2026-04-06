-- 000001_create_enum_types.down.sql
-- Drop all enum types. Tables must be dropped before this runs.

DROP TYPE IF EXISTS admin_permission_level;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS xp_source_type;
DROP TYPE IF EXISTS shop_item_type;
DROP TYPE IF EXISTS character_script;
DROP TYPE IF EXISTS segment_type;
DROP TYPE IF EXISTS srs_card_face;
DROP TYPE IF EXISTS srs_card_type;
DROP TYPE IF EXISTS quiz_type;
DROP TYPE IF EXISTS progress_status;
DROP TYPE IF EXISTS jlpt_level;
