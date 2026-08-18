CREATE TEMP TABLE system_skill_catalog_down_guard(value INTEGER);
CREATE TEMP TRIGGER system_skill_catalog_down_guard_retained
BEFORE INSERT ON system_skill_catalog_down_guard
WHEN EXISTS(SELECT 1 FROM system_skills)
  OR EXISTS(SELECT 1 FROM system_skill_versions)
  OR EXISTS(SELECT 1 FROM system_skill_audit)
BEGIN
    SELECT RAISE(ABORT, 'cannot remove retained Skill catalog');
END;
INSERT INTO system_skill_catalog_down_guard(value) VALUES(1);
DROP TRIGGER system_skill_catalog_down_guard_retained;
DROP TABLE system_skill_catalog_down_guard;
DROP TABLE IF EXISTS system_skill_audit;
DROP TABLE IF EXISTS system_skill_versions;
DROP TABLE IF EXISTS system_skills;
