-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Custom ENUM for difficulty perception
DO $$ BEGIN
    CREATE TYPE difficulty_level AS ENUM ('easy', 'ok', 'hard');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
