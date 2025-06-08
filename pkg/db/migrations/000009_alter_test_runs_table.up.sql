-- Step 1: Create the ENUM type
CREATE TYPE test_run_status AS ENUM ('FAILED', 'SKIPPED', 'PASSED');

-- Step 2: Add a new column (no TYPE keyword needed)
ALTER TABLE public.test_runs
ADD COLUMN status test_run_status;