ALTER TABLE users
  ADD COLUMN image_key text,
  ADD COLUMN image_mime text,
  ADD COLUMN image_updated_at timestamptz,
  ADD CONSTRAINT users_image_metadata_consistent CHECK (
    (image_key IS NULL AND image_mime IS NULL AND image_updated_at IS NULL)
    OR (image_key IS NOT NULL AND image_mime IS NOT NULL AND image_updated_at IS NOT NULL)
  );

ALTER TABLE products
  ADD COLUMN image_key text,
  ADD COLUMN image_mime text,
  ADD COLUMN image_updated_at timestamptz,
  ADD CONSTRAINT products_image_metadata_consistent CHECK (
    (image_key IS NULL AND image_mime IS NULL AND image_updated_at IS NULL)
    OR (image_key IS NOT NULL AND image_mime IS NOT NULL AND image_updated_at IS NOT NULL)
  );
