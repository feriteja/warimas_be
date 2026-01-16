-- +migrate Up

ALTER TABLE category
ADD COLUMN slug TEXT;

UPDATE category
SET slug = TRIM(BOTH '-' FROM LOWER(
  REGEXP_REPLACE(name, '[^a-zA-Z0-9]+', '-', 'g')
))
WHERE slug IS NULL;

ALTER TABLE category
ALTER COLUMN slug SET NOT NULL;

CREATE UNIQUE INDEX idx_category_slug_unique
ON category(slug);

CREATE OR REPLACE FUNCTION generate_category_slug()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.slug IS NULL OR NEW.slug = '' THEN
    NEW.slug := TRIM(BOTH '-' FROM LOWER(
      REGEXP_REPLACE(NEW.name, '[^a-zA-Z0-9]+', '-', 'g')
    ));
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_generate_category_slug
BEFORE INSERT OR UPDATE OF name ON category
FOR EACH ROW
EXECUTE FUNCTION generate_category_slug();


-- +migrate Down

-- +migrate Down

DROP TRIGGER IF EXISTS trg_generate_category_slug ON category;
DROP FUNCTION IF EXISTS generate_category_slug;

DROP INDEX IF EXISTS idx_category_slug_unique;
ALTER TABLE category DROP COLUMN slug;
