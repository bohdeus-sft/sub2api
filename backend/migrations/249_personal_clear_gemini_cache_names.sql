-- This personal fork retains provider cache bindings, not client-supplied labels.
-- Include soft-deleted records, whose display names otherwise remain in the DB.
UPDATE gemini_cached_contents
SET display_name = ''
WHERE display_name <> '';
