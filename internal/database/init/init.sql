CREATE TABLE IF NOT EXISTS test_run (
  id INT PRIMARY KEY,
  start_time TIMESTAMP,
  end_time TIMESTAMP
)

CREATE TABLE IF NOT EXISTS suite_run (
  id INT PRIMARY KEY,
  test_run_id INT 
  start_time TIMESTAMP,
  end_time TIMESTAMP,
  FOREIGN KEY(test_run_id) 
    REFERENCES test_run (id)
)

CREATE TABLE IF NOT EXISTS spec_run (
  id INT PRIMARY KEY,
  suite_id INT,
  spec_description VARCHAR(100),
  status VARCHAR(10)
  Message VARCHAR(255)
  start_time TIMESTAMP,
  end_time TIMESTAMP,
  FOREIGN KEY (suite_id)
    REFERENCES suite_run(id)
)
