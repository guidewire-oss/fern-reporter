CREATE TABLE IF NOT EXISTS suite_runs (
  id SERIAL PRIMARY KEY,
  test_run_id INT,
  start_time TIMESTAMP,
  end_time TIMESTAMP,
  FOREIGN KEY(test_run_id)
    REFERENCES test_runs (id)
)
