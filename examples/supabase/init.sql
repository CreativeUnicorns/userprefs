-- init.sql - Initialize local PostgreSQL for Supabase simulation
-- This script sets up the database similar to how Supabase would structure it

-- Enable necessary extensions (similar to Supabase)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create a schema for better organization (optional)
CREATE SCHEMA IF NOT EXISTS userprefs;

-- Set search path to include our schema
SET search_path TO userprefs, public;

-- The userprefs library will automatically create these tables,
-- but we can pre-create them with additional Supabase-like features

-- Preference definitions table with additional metadata
CREATE TABLE IF NOT EXISTS preference_definitions (
    key TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('string', 'number', 'boolean', 'json', 'enum')),
    category TEXT NOT NULL,
    default_value JSONB,
    allowed_values JSONB,
    description TEXT,
    is_required BOOLEAN DEFAULT false,
    is_sensitive BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User preferences table with additional audit fields
CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id TEXT NOT NULL,
    preference_key TEXT NOT NULL,
    value JSONB NOT NULL,
    previous_value JSONB,
    source TEXT DEFAULT 'api', -- 'api', 'import', 'default', etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, preference_key),
    FOREIGN KEY (preference_key) REFERENCES preference_definitions(key) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_user_preferences_preference_key ON user_preferences(preference_key);
CREATE INDEX IF NOT EXISTS idx_user_preferences_category ON user_preferences(preference_key) 
    WHERE EXISTS (
        SELECT 1 FROM preference_definitions pd 
        WHERE pd.key = user_preferences.preference_key
    );

-- Create GIN index for JSONB value searches
CREATE INDEX IF NOT EXISTS idx_user_preferences_value_gin ON user_preferences USING GIN(value);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update timestamps
DROP TRIGGER IF EXISTS update_preference_definitions_updated_at ON preference_definitions;
CREATE TRIGGER update_preference_definitions_updated_at
    BEFORE UPDATE ON preference_definitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_preferences_updated_at ON user_preferences;
CREATE TRIGGER update_user_preferences_updated_at
    BEFORE UPDATE ON user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to track preference value changes
CREATE OR REPLACE FUNCTION track_preference_changes()
RETURNS TRIGGER AS $$
BEGIN
    -- Store the previous value when updating
    IF TG_OP = 'UPDATE' AND OLD.value IS DISTINCT FROM NEW.value THEN
        NEW.previous_value = OLD.value;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to track changes
DROP TRIGGER IF EXISTS track_user_preference_changes ON user_preferences;
CREATE TRIGGER track_user_preference_changes
    BEFORE UPDATE ON user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION track_preference_changes();

-- Optional: Create a view for easier preference analytics
CREATE OR REPLACE VIEW preference_analytics AS
SELECT 
    pd.key,
    pd.type,
    pd.category,
    pd.description,
    COUNT(up.user_id) as user_count,
    COUNT(DISTINCT up.user_id) as unique_users,
    pd.created_at as definition_created,
    MAX(up.updated_at) as last_updated
FROM preference_definitions pd
LEFT JOIN user_preferences up ON pd.key = up.preference_key
GROUP BY pd.key, pd.type, pd.category, pd.description, pd.created_at;

-- Grant permissions (adjust as needed for your security model)
-- In Supabase, you would use Row Level Security (RLS) instead
GRANT ALL ON userprefs.preference_definitions TO postgres;
GRANT ALL ON userprefs.user_preferences TO postgres;
GRANT SELECT ON userprefs.preference_analytics TO postgres;

-- Insert some sample data for testing (optional)
INSERT INTO preference_definitions (key, type, category, description, default_value) VALUES
    ('theme', 'enum', 'appearance', 'User interface theme', '"dark"'::jsonb),
    ('language', 'string', 'localization', 'User preferred language', '"en"'::jsonb),
    ('notifications_enabled', 'boolean', 'notifications', 'Enable push notifications', 'true'::jsonb),
    ('max_file_size', 'number', 'limits', 'Maximum file upload size in bytes', '10485760'::jsonb)
ON CONFLICT (key) DO NOTHING;

-- Create a function to safely reset all demo data (useful for testing)
CREATE OR REPLACE FUNCTION reset_demo_data()
RETURNS void AS $$
BEGIN
    DELETE FROM user_preferences WHERE user_id LIKE 'demo_%' OR user_id LIKE 'user_%' OR user_id LIKE 'alice_%' OR user_id LIKE 'bob_%' OR user_id LIKE 'charlie_%' OR user_id LIKE 'concurrent_%';
    RAISE NOTICE 'Demo data cleared successfully';
END;
$$ language 'plpgsql';

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE 'Supabase simulation database initialized successfully!';
    RAISE NOTICE 'Schema: userprefs';
    RAISE NOTICE 'Tables: preference_definitions, user_preferences';
    RAISE NOTICE 'Views: preference_analytics';
    RAISE NOTICE 'Functions: update_updated_at_column, track_preference_changes, reset_demo_data';
END $$;
