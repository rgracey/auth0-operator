<br />
<div align="center">
  <div>
    <img src="./docs/images/auth0_logo.svg" alt="Auth0 logo" width="80" height="80">
    <img src="./docs/images/k8s_logo.svg" alt="Auth0 logo" width="80" height="80">
  </div>

  <h1 align="center" style="background: linear-gradient(to right, #eb5424, #326ce5); -webkit-background-clip: text; -webkit-text-fill-color: transparent; font-size: 48px;">Auth0 Operator</h1>

  <p align="center">
    Kubernetes operator for Auth0 management
  </p>
  <br />
</div>

[![Test 🧪](https://github.com/rgracey/auth0-operator/actions/workflows/test.yaml/badge.svg?branch=main&event=push)](https://github.com/rgracey/auth0-operator/actions/workflows/test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rgracey/auth0-operator)](https://goreportcard.com/report/github.com/rgracey/auth0-operator)

## About The Project

The Auth0 Kubernetes Operator is responsible for managing the lifecycle of Auth0 resources in a Kubernetes cluster.

It automates the deployment, configuration, and management of Auth0 resources, such as clients, connections, resource servers and more. (WIP)

### Built With

-   [Kubebuilder](https://book.kubebuilder.io/)
-   [go-auth0](https://github.com/auth0/go-auth0)

## Getting Started

TODO

### Prerequisites

-   Go 1.20+

### Installation

TODO

## Usage

See the [examples](./docs/examples) directory for usage examples.

## Roadmap

-   [ ] Clients `[WIP]`
    -   [ ] Client credentials
    -   [ ] Rotateable client secrets
    -   [ ] Client grants
-   [ ] Connections
-   [ ] Resource Servers
-   [ ] Helm chart
