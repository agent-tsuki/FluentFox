-- +goose Up
-- The voice behind each key ("main", or a listening cast key such as f1): an
-- engine style, chosen by a super-admin. A key with no row keeps the voice
-- from the server's env (VOICEVOX_SPEAKER_ID for main). Clips live in a
-- folder named after the engine and style, so choosing another style never
-- overwrites clips: the new voice simply has none until they are made.
CREATE TABLE audio_voice (
    key          text PRIMARY KEY,
    engine       text NOT NULL,
    style_id     bigint NOT NULL,
    speaker_name text NOT NULL,
    style_name   text NOT NULL,
    -- The AivisHub model the style comes from, so a fresh engine can install
    -- it; null for a VOICEVOX voice (built into the engine).
    model_uuid   uuid,
    credit       text NOT NULL,
    updated_by   uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_audio_voice_key CHECK (key ~ '^[a-z][a-z0-9]{0,15}$' AND key <> 'alt'),
    CONSTRAINT chk_audio_voice_engine CHECK (engine ~ '^[a-z][a-z0-9-]{0,31}$')
);

-- Every clip stored in R2, so finding one is a database lookup instead of a
-- storage request. Clips made before this table are added the first time
-- they are found in storage.
CREATE TABLE audio_clip (
    object_key text PRIMARY KEY,
    profile    text NOT NULL,
    text       text NOT NULL,
    bytes      int NOT NULL DEFAULT 0,
    created_by uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audio_clip_profile ON audio_clip (profile);

-- Work for the audio worker (cmd/audioworker, next to the voice engine):
-- generate a voice's missing clips, or install an AivisHub model.
CREATE TYPE audio_job_kind AS ENUM ('generate', 'install_model');
CREATE TYPE audio_job_status AS ENUM ('queued', 'running', 'done', 'failed', 'cancelled');

CREATE TABLE audio_job (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind         audio_job_kind NOT NULL,
    status       audio_job_status NOT NULL DEFAULT 'queued',
    -- generate: one voice key, or null for every voice.
    voice_key    text,
    -- install_model: the AivisHub model.
    model_uuid   uuid,
    total        int NOT NULL DEFAULT 0,
    done         int NOT NULL DEFAULT 0,
    failed       int NOT NULL DEFAULT 0,
    error        text,
    created_by   uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    started_at   timestamptz,
    heartbeat_at timestamptz,
    finished_at  timestamptz,
    CONSTRAINT chk_audio_job_model CHECK (kind <> 'install_model' OR model_uuid IS NOT NULL)
);

CREATE INDEX idx_audio_job_queue ON audio_job (status, created_at);

-- The listening cast, as chosen on 2026-11-09 (AivisSpeech, ACML 1.0 models).
INSERT INTO audio_voice (key, engine, style_id, speaker_name, style_name, model_uuid, credit) VALUES
    ('f1', 'aivisspeech', 888753760,  'まお',     'ノーマル', 'a59cb814-0083-4369-8542-f51a29e72af7', 'AivisSpeech:まお'),
    ('f2', 'aivisspeech', 497929760,  'morioki',  'ノーマル', 'baaae3c0-7b22-4605-8ba5-80c959b41a48', 'AivisSpeech:morioki'),
    ('f3', 'aivisspeech', 1325133120, '花音',     'ノーマル', 'a670e6b8-0852-45b2-8704-1bc9862f2fe6', 'AivisSpeech:花音'),
    ('f4', 'aivisspeech', 1431611904, 'まい',     'ノーマル', 'e9339137-2ae3-4d41-9394-fb757a7e61e6', 'AivisSpeech:まい'),
    ('m1', 'aivisspeech', 606865152,  'fumifumi', 'ノーマル', '71e72188-2726-4739-9aa9-39567396fb2a', 'AivisSpeech:fumifumi'),
    ('m2', 'aivisspeech', 1937616896, 'にせ',     'ノーマル', '6d11c6c2-f4a4-4435-887e-23dd60f8b8dd', 'AivisSpeech:にせ'),
    ('m3', 'aivisspeech', 1310138977, '阿井田 茂', 'Calm',    '47e53151-a378-46f3-abee-ce13aa07feb1', 'AivisSpeech:阿井田 茂'),
    ('m4', 'aivisspeech', 1628969216, 'TANAKA',   'ノーマル', '470a41e2-863f-4bf2-8a20-bf7cde0d8f7b', 'AivisSpeech:TANAKA'),
    ('m5', 'aivisspeech', 1099751712, 'yukyu',    'デフォルト', '1f04a608-916c-4d9e-bf6e-b9da5c176f85', 'AivisSpeech:yukyu');

-- +goose Down
DROP TABLE IF EXISTS audio_job;
DROP TYPE IF EXISTS audio_job_status;
DROP TYPE IF EXISTS audio_job_kind;
DROP TABLE IF EXISTS audio_clip;
DROP TABLE IF EXISTS audio_voice;
