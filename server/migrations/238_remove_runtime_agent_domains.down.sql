DO $$
BEGIN
    RAISE EXCEPTION 'migration 238 removes legacy runtime and agent domains and cannot be reversed';
END
$$;
