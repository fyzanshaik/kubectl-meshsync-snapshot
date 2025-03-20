# kubectl MeshSync Snapshot Plugin

A kubectl plugin for capturing Kubernetes cluster state using MeshSync technology and saving it for offline analysis in Meshery.

## Overview

This plugin provides a lightweight alternative to a full Meshery deployment, allowing users to capture a point-in-time snapshot of their Kubernetes cluster's configuration. The snapshot can be
imported into Meshery for visualization and analysis without requiring permanent connectivity between Meshery Server and the cluster.

## Background

### What is Meshery?

Meshery is a self-service engineering platform for the collaborative design and operation of cloud and cloud native infrastructure. It provides a unified interface for managing Kubernetes clusters,
service meshes, and cloud resources.

### What is MeshSync?

MeshSync is a component of Meshery that:

-  Discovers and synchronizes resources from Kubernetes clusters
-  Operates as a Kubernetes controller under the management of Meshery Operator
-  Captures configuration state of resources and sends it to Meshery Server via a NATS broker

## Plugin Implementation

### Original Approach

My initial approach was based on the understanding that I needed to:

1. Temporarily deploy MeshSync to a Kubernetes cluster
2. Capture the discovered resources
3. Save them to a file for later import into Meshery
4. Remove the temporary MeshSync deployment

However, after discussions with the project maintainer, I learned that I could use MeshSync as a local binary instead of deploying it as a pod, simplifying my approach.

### Current Implementation

My current implementation:

1. Sets up a temporary NATS server for MeshSync to connect to
2. Runs the MeshSync binary with configuration pointing to this NATS server
3. Subscribes to the topics MeshSync publishes to
4. Captures and saves the discovered resources to a file
5. Cleans up all temporary resources

### Solutions Explored

Instead of modifying the MeshSync component, I explored several approaches to integrate with it:

#### Solution 1: Use MeshSync As-Is with Custom Environment

In this approach, I run the unmodified MeshSync binary but create a controlled environment around it:

-  Set up a temporary NATS server on localhost
-  Configure host entries to ensure MeshSync can connect to it
-  Apply necessary CRDs to satisfy MeshSync's requirements
-  Subscribe to MeshSync's published topics to capture data

This has the advantage of not requiring any changes to MeshSync but introduces complexity in managing the environment.

#### Solution 2: Fork and Modify MeshSync

If needed, I could fork the MeshSync repository and make specific changes:

-  Modify the `connectivityTest` function to be optional via an environment variable
-  Add an option to output discovered resources directly to a file
-  Remove the dependency on CRDs for basic operation
-  Add a "snapshot mode" that captures state without requiring broker connectivity

This approach gives more control but creates a maintenance burden of keeping the fork in sync with upstream changes.

#### Solution 3: Custom Implementation Using client-go

As an alternative approach, I implemented a direct resource discovery mechanism using Kubernetes client-go:

-  Directly query the Kubernetes API for resources
-  Avoid the complexity of MeshSync, NATS, and CRDs entirely
-  Format data in a compatible way for Meshery import
-  Provide a simpler, more reliable solution

This approach sacrifices some of MeshSync's advanced discovery capabilities but offers a more straightforward implementation.

### CRD Requirements

For the MeshSync integration, I needed to apply the CRDs that MeshSync expects. I found these in the Meshery repository:

```
/install/kubernetes/helm/meshery-operator/crds/crds.yaml
```

This file contains definitions for:

-  `brokers.meshery.io` - Defines the Broker custom resource
-  `meshsyncs.meshery.io` - Defines the MeshSync custom resource

These CRDs are essential because MeshSync looks for a `MeshSync` custom resource named `meshery-meshsync` in the `meshery` namespace to configure its operation. Without these CRDs and a corresponding
instance, MeshSync reports errors and may not function properly.

## Technical Challenges

During development, I encountered several challenges:

#### 1. NATS Server Configuration

MeshSync expected a NATS server with specific configuration. I needed to ensure:

-  The server was available on the correct hostname and port
-  The HTTP monitoring endpoint was enabled on port 8222
-  The server name was configured correctly

Solution: I implemented a NATS server that listens on both port 4222 (NATS protocol) and 8222 (HTTP monitoring).

#### 2. MeshSync CRD Requirements

MeshSync looks for Custom Resource Definitions (CRDs) in the cluster. When these were not found, MeshSync would report errors:

```
WARN[...] Missing or outdated CRD. app=meshsync ...
```

Solution: I implemented a CRD manager that:

-  Embeds the necessary CRD definitions
-  Applies them to the cluster before running MeshSync
-  Removes them after capture is complete
-  Creates a MeshSync custom resource instance that points to our NATS broker

#### 3. Data Capture from MeshSync

The most challenging aspect was capturing the data discovered by MeshSync. I investigated multiple approaches:

-  Using MeshSync as a library (importing it directly)
-  Using MeshSync as a binary (executing it as a subprocess)
-  Subscribing to the NATS topics MeshSync publishes to

I chose the NATS subscription approach as it required minimal changes to MeshSync while allowing me to capture its discovered resources.

#### 4. MeshSync Panic

Despite applying the CRDs and creating a MeshSync custom resource, I still encountered a panic in MeshSync:

```
panic: interface conversion: interface {} is nil, not map[string]cache.Store
```

This appears to be related to how MeshSync initializes its discovery pipeline. I continue to investigate this issue.

## Project Structure

```
kubectl-meshsync_snapshot/
├── cmd/
│   └── kubectl-meshsync_snapshot/      # Main executable code
│       └── main.go                     # Entry point for the plugin
├── pkg/
│   ├── crds/                           # CRD management code
│   │   └── manager.go                  # Applies/removes Meshery CRDs
│   ├── k8s/                            # Kubernetes utilities
│   │   └── snapshot.go                 # Direct resource capture (fallback)
│   ├── meshsync/                       # MeshSync integration
│   │   ├── runner.go                   # Runs MeshSync binary
│   │   └── subscriber.go               # Subscribes to MeshSync data
│   ├── models/                         # Data models
│   │   └── kubernetes.go               # Resource models
│   └── nats/                           # NATS server code
│       └── server.go                   # Temporary NATS server
├── go.mod                              # Go module definition
├── go.sum                              # Go dependencies checksum
├── Makefile                            # Build automation
└── README.md                           # This file
```

## Flow Diagram

```
┌─────────────────┐     ┌────────────────┐    ┌────────────────┐     ┌────────────────┐
│  kubectl plugin │────▶│  NATS Server   │◀───│  MeshSync      │────▶│  Snapshot File │
└─────────────────┘     └────────────────┘    └────────────────┘     └────────────────┘
        │                                             ▲
        │                                             │
        ▼                                             │
┌─────────────────┐                                   │
│  CRD Manager    │───────────────────────────────────┘
└─────────────────┘
```

1. The plugin starts a temporary NATS server
2. It applies the necessary CRDs and creates a MeshSync custom resource
3. It runs the MeshSync binary, which connects to the NATS server
4. MeshSync discovers resources and publishes them to NATS
5. The plugin subscribes to these resources and saves them to a file
6. All temporary resources are cleaned up

## Current Status and Issues

The current implementation faces a challenge with MeshSync crashing with a panic:

```
panic: interface conversion: interface {} is nil, not map[string]cache.Store
```

This occurs in the `startDiscovery` method in `meshsync/discovery.go`. The issue appears to be related to MeshSync's expectation of certain configuration or resources being present in the cluster.

I'm exploring two paths forward:

1. **Continue troubleshooting MeshSync integration**: Identify and fix the specific configuration MeshSync expects
2. **Implement a direct discovery approach**: Use the Kubernetes client-go library to discover resources directly

The direct discovery approach provides a more reliable solution in the short term, while continuing to work on the MeshSync integration for a more comprehensive solution.

## Usage

Currently only basic usage works these are some of the commands which we could use

```bash
# Basic usage
kubectl meshsync-snapshot

# Specify output file
kubectl meshsync-snapshot --output=my-snapshot.json

# Specify wait time (seconds)
kubectl meshsync-snapshot --wait=60

# Debug mode
kubectl meshsync-snapshot --debug
```

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/kubectl-meshsync_snapshot.git
cd kubectl-meshsync_snapshot

# Build the plugin
make build

# Install to your PATH
make install
```

### Via Krew (Coming Soon)(What the end project is supposed to be)

```bash
kubectl krew install meshsync-snapshot
```

## Development

To set up a development environment:

```bash
# Clone the repository
git clone https://github.com/yourusername/kubectl-meshsync_snapshot.git
cd kubectl-meshsync_snapshot

# Get dependencies
go mod download

# Run in development mode
go run cmd/kubectl-meshsync_snapshot/main.go --debug
```

For running the plugin make sure to have mesherysync binary file inside the folder, as we will be invoking it directly just for now.
