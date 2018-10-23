INSERT INTO
   users(created_at, updated_at, id, name, allocated_storage)
VALUES
   (now(), now(), '01b291cd-9399-4f1a-8bbc-d1de66d76192', 'osio-stage', 32767),
-- value for allocated_storage=32768 is out of range for smallint datatype
   (now(), now(), '01b291cd-9399-4f1a-8bbc-d1de66d76192', 'osio-stage', 32768);
