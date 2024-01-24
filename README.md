[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/Guidewire/fern-reporter/badge)](https://securityscorecards.dev/viewer/?uri=github.com/Guidewire/fern-reporter)
![Coverage](https://img.shields.io/badge/Coverage-23.0%25-red)

![Fern](https://github.com/guidewire/fern-reporter/raw/main/docs/images/logo-no-background.png)


## Introduction

Welcome to the Fern Project, an innovative open-source solution designed to enhance Ginkgo test reports. This project is focused on capturing, storing, and analyzing test data to provide insights into test performance and trends. The Fern Project is ideal for teams using Ginkgo, a popular BDD-style Go testing framework, offering a comprehensive overview of test executions and performance metrics.

Key Features:

1. **Historical Test Data Tracking**: Stores detailed records of tests run against various projects, providing a historical view of testing efforts.
2. **Latency and Performance Metrics**: Captures the time taken for each "It" block in Ginkgo tests, aiding in identifying performance bottlenecks.
3. **Data-Driven Analytics (To be implemented)**: Future feature to leverage data for analytics, including identification of frequently failing tests.
4. **Coverage and Test Evolution Analysis (To be implemented)**: Planned feature to offer insights into test coverage and the evolution of tests over time.
5. **Authorized Access to Test Reports (To be implemented)**: Upcoming feature to ensure secure access to test reports.

## Installation and Setup

Fern is a Golang Gin-based API that connects to a PostgreSQL database. It is designed to store metadata about Ginkgo test suites and has two main components:

1. **API Server**: A central server that stores test metadata. It needs to be deployed independently.
2. **Client Library**: Integrated into Ginkgo test suites to send test data to the API server.

### Setting Up the API Server

#### Prerequisites

- Golang environment.
- Docker for running the PostgreSQL database.
- Install gox for building the binary
   ```bash
   go install github.com/mitchellh/gox@latest
   ```

#### Deployment

1. **Clone the Fern Repository**: Clone the repository to your local machine.
   ```bash
   git clone git@github.com:Guidewire/fern-reporter.git
   ```
2. **Start the API Server**: Navigate to the project directory and start the server.
   ```bash
   cd fern-reporter
   make docker-run
   ```

### Integrating the Client into Ginkgo Test Suites

1. **Add the Fern dependency to your test project**:

   ```bash
   go get -u github.com/guidewire/fern-reporter
   ```
2. **Add the Fern Client to your Ginkgo test suite**:
   
   Import the fern client package
   ```go
   import fern "github.com/guidewire/fern-reporter/pkg/client"
   ```

   ```go
   var _ = ReportAfterSuite("", func(report Report) {
       f := fern.New("Example Test",
           fern.WithBaseURL("http://localhost:8080/"),
       )

       err := f.Report("example test", report)

       Expect(err).To(BeNil(), "Unable to create reporter file")
   })
   ```
   Replace `http://localhost:8080/` with your API server's URL and specify the project name in `f.Report`.

2. **Run Your Tests**: After adding the client, run your Ginkgo tests normally.

### Accessing Test Reports

- View reports at `http://[your-api-url]/reports/testruns`.
- If using `make docker-run`, reports are available at `http://localhost:8080/reports/testruns`.

### Additional Resources

- [Deploying fern reporter service in kubernetes using kubevela](docs/kubevela/README.md)
