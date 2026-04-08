CREATE TABLE Vocabulary (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kanji    UUID NOT NULL REFERENCES users(id),

    -- vocab data
    word    VARCHAR NOT NULL,
    meaning VARCHAR NOT NULL,
    hiragana VARCHAR NULL,
    romaji VARCHAR NULL,

    -- vocab info
    img_url VARCHAR NULL,
    audio_file VARCHAR NULL,

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    CONSTRAINT uq_user_verification_user_id UNIQUE (user_id)
);

CREATE TABLE kanji (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- vocab data
    word    VARCHAR NOT NULL,
    onyumi  VARCHAR NULL,
    kunyomi  VARCHAR NULL,
    meaning VARCHAR NOT NULL,
    hiragana VARCHAR NULL,
    romaji VARCHAR NULL,

    -- vocab info
    img_url VARCHAR NULL,
    audio_file VARCHAR NULL, 
    

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);


CREATE TABLE kanas (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- vocab data
    character VARCHAR NOT NULL;
    romanji   VARCHAR NOT NULL:
    
    
    -- vocab info
    img_url VARCHAR NULL,
    audio_file VARCHAR NULL,

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    CONSTRAINT uq_user_verification_user_id UNIQUE (user_id)
);

