ALTER TABLE public.test_runs

ADD COLUMN git_branch VARCHAR(100),
ADD COLUMN git_sha VARCHAR(50),
ADD COLUMN build_trigger_actor VARCHAR(50),
ADD COLUMN build_url VARCHAR(250)