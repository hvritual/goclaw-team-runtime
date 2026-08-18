CREATE TEMP TABLE space_skill_objects_down_guard(value INTEGER);
CREATE TEMP TRIGGER space_skill_objects_down_guard_retained
BEFORE INSERT ON space_skill_objects_down_guard
WHEN EXISTS(SELECT 1 FROM space_skill_objects)
BEGIN
    SELECT RAISE(ABORT, 'cannot remove retained Skill objects');
END;
INSERT INTO space_skill_objects_down_guard(value) VALUES(1);
DROP TRIGGER space_skill_objects_down_guard_retained;
DROP TABLE space_skill_objects_down_guard;
DROP TABLE IF EXISTS space_skill_objects;
