-- 000011_create_concept_examples.up.sql
-- A single example sentence pair (Japanese + English) belonging to a concept.
-- The Japanese sentence is NOT stored here — it lives in example_segments.

CREATE TABLE IF NOT EXISTS concept_examples (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    concept_id  UUID         NOT NULL,
    -- English translation of the example. Example: "I am a company employee."
    english     VARCHAR(500) NOT NULL,
    -- Position of this example within its concept
    order_index INTEGER      NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_concept_examples_concepts FOREIGN KEY (concept_id) REFERENCES concepts (id) ON DELETE CASCADE
);

CREATE INDEX idx_concept_examples_concept_id ON concept_examples (concept_id);
CREATE INDEX idx_concept_examples_order      ON concept_examples (concept_id, order_index);
