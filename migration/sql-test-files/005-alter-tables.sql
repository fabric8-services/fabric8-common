ALTER TABLE ONLY environments ADD COLUMN cluster text;
UPDATE environments SET cluster = '{{ index . 0}}'  WHERE cluster is null or cluster = '';
ALTER TABLE environments ALTER COLUMN cluster set NOT NULL ,ADD CHECK (cluster <> '');
