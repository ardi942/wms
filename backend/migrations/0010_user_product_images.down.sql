ALTER TABLE users DROP CONSTRAINT IF EXISTS users_image_metadata_consistent;
ALTER TABLE users
  DROP COLUMN IF EXISTS image_key,
  DROP COLUMN IF EXISTS image_mime,
  DROP COLUMN IF EXISTS image_updated_at;

ALTER TABLE products DROP CONSTRAINT IF EXISTS products_image_metadata_consistent;
ALTER TABLE products
  DROP COLUMN IF EXISTS image_key,
  DROP COLUMN IF EXISTS image_mime,
  DROP COLUMN IF EXISTS image_updated_at;
