CREATE TEMP TABLE system_skill_files_down_guard(value INTEGER);
CREATE TEMP TRIGGER system_skill_files_down_guard_retained
BEFORE INSERT ON system_skill_files_down_guard
WHEN EXISTS(SELECT 1 FROM system_skill_file_manifests)
  OR EXISTS(SELECT 1 FROM system_skill_import_previews)
  OR EXISTS(SELECT 1 FROM system_skill_import_idempotency)
BEGIN
    SELECT RAISE(ABORT, 'cannot remove retained Skill file/import data');
END;
INSERT INTO system_skill_files_down_guard(value) VALUES(1);
DROP TRIGGER system_skill_files_down_guard_retained;
DROP TABLE system_skill_files_down_guard;
DROP TABLE IF EXISTS system_skill_import_idempotency;
DROP TABLE IF EXISTS system_skill_import_previews;
DROP TABLE IF EXISTS system_skill_file_manifests;
