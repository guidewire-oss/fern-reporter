-- Step 1: Remove the column
ALTER TABLE public.test_runs
DROP COLUMN status;

-- Step 2: Drop the ENUM type
DROP TYPE test_run_status;