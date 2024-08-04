![Fern](https://github.com/guidewire/fern-reporter/raw/main/docs/images/logo-no-background.png)

<p align="center">
    <a href="https://github.com/guidewire/fern-reporter/blob/main/LICENSE" alt="License">
        <img src="https://badgen.net/github/license/guidewire/fern-reporter/" /></a>
    <!-- <a href="https://github.com/guidewire/fern-reporter/actions" alt="Fern Reporter Release"> -->
    <!--     <img alt="GitHub Workflow Status (with event)" src="https://img.shields.io/github/actions/workflow/status/guidewire/fern-reporter/build.yml?event=pull_request"></a> -->
    <a href="https://goreportcard.com/report/github.com/guidewire/fern-reporter" alt="Go Report Card">
        <img src="https://goreportcard.com/badge/github.com/guidewire/fern-reporter" /></a>
    <a href="https://codecov.io/gh/guidewire-oss/fern-reporter" alt="Code Coverage">
        <img src="https://codecov.io/gh/guidewire/fern-reporter/branch/main/graph/badge.svg" /></a>
    <a href="https://github.com/guidewire/fern-reporter/graphs/contributors" alt="Release">
        <img alt="GitHub contributors" src="https://img.shields.io/github/contributors/guidewire/fern-reporter"></a>
</p>

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
   make docker-run-local
   ```

### Integrating the Client into Ginkgo Test Suites

* Refer the client repository to integrate the client to Ginkgo Test Suites: 
  https://github.com/Guidewire/fern-ginkgo-client
* After adding the client, run your Ginkgo tests normally.

### Accessing Test Reports using embedded HTML view

- View reports at `http://[your-api-url]/reports/testruns/`.
- If using `make docker-run-local`, reports are available at `http://localhost:8080/reports/testruns/`.

### Accessing Test Reports using Fern-UI

To view the test reports using the React-based Fern-UI frontend, follow these steps:

- Follow the instructions in the [Fern-UI repository](https://github.com/Guidewire/fern-ui) to set up and run the frontend application.
- Once the Fern-UI is running, access the dashboard at `http://[frontend-api-url]/testruns` to view the test reports.

- View reports at `http://[your-api-url]/reports/testruns`.
- If using `make docker-run-local`, reports are available at `http://localhost:8080/reports/testruns`.

### Accessing Test Reports using the API
Reports are also available as JSON at `http://[host-url]/api/reports/testruns`.

### Additional Resources

- [Deploying fern reporter service in kubernetes using kubevela](docs/kubevela/README.md)

## 🤩 Thanks to all our Contributors

Thanks to everyone, that is supporting this project. We are thankful, for evey contribution, no matter its size!

<a href="https://github.com/Guidewire/fern-reporter/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=Guidewire/fern-reporter" />
</a>
